package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/DaDevFox/task-systems/task-core/backend/internal/config"
	"github.com/DaDevFox/task-systems/task-core/backend/internal/dagview"
	"github.com/DaDevFox/task-systems/task-core/backend/internal/domain"
	"github.com/DaDevFox/task-systems/task-core/backend/internal/idresolver"
	pb "github.com/DaDevFox/task-systems/tasker-core/pkg/proto/taskcore/v1"
)

var (
	serverAddr   string
	currentUser  string
	userFlag     string
	client       pb.TaskServiceClient
	conn         *grpc.ClientConn
	cfg          *config.Config
	taskResolver *idresolver.TaskIDResolver
	userResolver *idresolver.UserResolver
)

// Constants for repeated strings
const (
	defaultUserID            = "default-user"
	warnFailedUpdateResolver = "Warning: Failed to update resolvers: %v\n"
	taskSelectionFailed      = "task selection failed: %w"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "tasker",
		Short: "Enhanced Task management CLI client with robust ID resolution",
		Long:  "A comprehensive task management CLI client with robust short ID and user name resolution",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if err := initializeClient(); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to initialize client: %v\n", err)
				os.Exit(1)
			}
		},
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			if conn != nil {
				conn.Close()
			}
		},
	}

	// Global flags
	rootCmd.PersistentFlags().StringVar(&serverAddr, "server", "", "Server address (overrides config)")
	rootCmd.PersistentFlags().StringVarP(&userFlag, "user", "u", "", "User ID or name (overrides config)")

	// Add commands
	rootCmd.AddCommand(newConfigCommand())
	rootCmd.AddCommand(newAddCommand())
	rootCmd.AddCommand(newListCommand())
	rootCmd.AddCommand(newGetCommand())
	rootCmd.AddCommand(newStartCommand())
	rootCmd.AddCommand(newStopCommand())
	rootCmd.AddCommand(newCompleteCommand())
	rootCmd.AddCommand(newStageCommand())
	rootCmd.AddCommand(newTagCommand())
	rootCmd.AddCommand(newUserCommand())
	rootCmd.AddCommand(newSyncCommand())
	rootCmd.AddCommand(newDAGCommand())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Command execution failed: %v\n", err)
		os.Exit(1)
	}
}

func initializeClient() error {
	// Load configuration
	var err error
	cfg, err = config.LoadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to load config: %v\n", err)
		cfg = config.DefaultConfig()
	}

	// Use server address from flag, config, or default
	if serverAddr == "" {
		serverAddr = cfg.ServerAddr
	}

	// Set current user from flag, config, or default
	if userFlag != "" {
		currentUser = userFlag
	} else if cfg.CurrentUser != "" {
		currentUser = cfg.CurrentUser
	} else {
		currentUser = defaultUserID
	}

	// Connect to server
	conn, err = grpc.Dial(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to connect to server at %s: %w", serverAddr, err)
	}
	client = pb.NewTaskServiceClient(conn)

	// Initialize ID resolvers
	taskResolver = idresolver.NewTaskIDResolver()
	userResolver = idresolver.NewUserResolver()

	// Populate resolvers with initial data
	if err := updateResolvers(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to update resolvers: %v\n", err)
	}

	return nil
}

func updateResolvers() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Update task resolver with all tasks from all users
	taskReq := &pb.ListTasksRequest{} // Get all tasks
	taskResp, err := client.ListTasks(ctx, taskReq)
	if err != nil {
		return fmt.Errorf("failed to list tasks for resolver update: %w", err)
	}

	// Convert to domain tasks
	domainTasks := make([]*domain.Task, len(taskResp.Tasks))
	for i, protoTask := range taskResp.Tasks {
		domainTasks[i] = protoTaskToDomain(protoTask)
	}
	taskResolver.UpdateTasks(domainTasks)

	// Update user resolver - we need to get users from tasks since there's no ListUsers endpoint
	// In a real implementation, you'd have a proper ListUsers endpoint
	userMap := make(map[string]*domain.User)
	for _, task := range taskResp.Tasks {
		if task.UserId != "" && userMap[task.UserId] == nil { // Try to get user details
			userReq := &pb.GetUserRequest{
				Identifier: &pb.GetUserRequest_UserId{UserId: task.UserId},
			}
			userResp, err := client.GetUser(ctx, userReq)
			if err == nil {
				domainUser := protoUserToDomain(userResp.User)
				userMap[task.UserId] = domainUser
			}
		}
	}

	// Convert map to slice
	users := make([]*domain.User, 0, len(userMap))
	for _, user := range userMap {
		users = append(users, user)
	}

	if err := userResolver.UpdateUsers(users); err != nil {
		return fmt.Errorf("failed to update user resolver: %w", err)
	}

	return nil
}

func resolveUserID(ctx context.Context, userInput string) (string, error) {
	if userInput == "" {
		return currentUser, nil
	}

	// Try to resolve using the user resolver
	resolvedUser, err := userResolver.ResolveUser(userInput, true, true)
	if err == nil {
		return resolvedUser.ID, nil
	}

	// If resolver fails, try direct API call
	userReq := &pb.GetUserRequest{UserId: userInput}
	_, err = client.GetUser(ctx, userReq)
	if err == nil {
		return userInput, nil // Found by ID
	}

	return "", fmt.Errorf("failed to resolve user '%s': %w", userInput, err)
}

func resolveTaskID(ctx context.Context, taskInput string) (string, error) {
	if taskInput == "" {
		return "", fmt.Errorf("empty task ID provided")
	}

	// Try to resolve using the task resolver
	resolvedID, err := taskResolver.ResolveTaskID(taskInput)
	if err == nil {
		return resolvedID, nil
	}

	// If resolver fails, refresh and try again
	if err := updateResolvers(); err != nil {
		return "", fmt.Errorf("failed to refresh task data: %w", err)
	}

	resolvedID, err = taskResolver.ResolveTaskID(taskInput)
	if err != nil {
		return "", fmt.Errorf("task resolution failed for '%s': %w", taskInput, err)
	}

	return resolvedID, nil
}

func newConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Configuration management commands",
	}

	// Set current user
	cmd.AddCommand(&cobra.Command{
		Use:   "set-user <user-id-or-name>",
		Short: "Set the current user (resolved by ID or unique name)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			userInput := args[0]

			// Try to resolve user by name or ID
			resolvedUserID, err := resolveUserID(ctx, userInput)
			if err != nil {
				return fmt.Errorf("failed to resolve user '%s': %w", userInput, err)
			}

			cfg.CurrentUser = resolvedUserID
			if err := cfg.SaveConfig(); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}

			fmt.Printf("Current user set to: %s\n", resolvedUserID)
			return nil
		},
	})

	// Set server address
	cmd.AddCommand(&cobra.Command{
		Use:   "set-server <address>",
		Short: "Set the server address",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg.ServerAddr = args[0]
			if err := cfg.SaveConfig(); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}

			fmt.Printf("Server address set to: %s\n", args[0])
			return nil
		},
	})

	// Show current config
	cmd.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "Show current configuration",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Current Configuration:\n")
			fmt.Printf("  Current User: %s\n", cfg.CurrentUser)
			fmt.Printf("  Server Address: %s\n", cfg.ServerAddr)
		},
	})

	// Reset config to defaults
	cmd.AddCommand(&cobra.Command{
		Use:   "reset",
		Short: "Reset configuration to defaults",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg = config.DefaultConfig()
			if err := cfg.SaveConfig(); err != nil {
				return fmt.Errorf("failed to save default config: %w", err)
			}

			fmt.Println("Configuration reset to defaults")
			return nil
		},
	})

	return cmd
}

func newAddCommand() *cobra.Command {
	var description string
	var userID string

	cmd := &cobra.Command{
		Use:   "add <task-name>",
		Short: "Add a new task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			name := args[0]

			// Resolve user ID if provided, otherwise use current user
			resolvedUserID, err := resolveUserID(ctx, userID)
			if err != nil {
				return fmt.Errorf("failed to resolve user: %w", err)
			}

			req := &pb.AddTaskRequest{
				Name:        name,
				Description: description,
				UserId:      resolvedUserID,
			}

			resp, err := client.AddTask(ctx, req)
			if err != nil {
				return fmt.Errorf("add task operation failed: %w", err)
			}

			fmt.Printf("Created task: %s (ID: %s)\n", resp.Task.Name, resp.Task.Id)

			// Update resolvers with new task
			if err := updateResolvers(); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Failed to update resolvers: %v\n", err)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&description, "description", "d", "", "Task description")
	cmd.Flags().StringVarP(&userID, "user", "u", "", "User ID or name (overrides current user)")

	return cmd
}

func newListCommand() *cobra.Command {
	var stage string
	var userID string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List tasks for current user or specified user",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			// Use provided user or current user
			resolvedUserID, err := resolveUserID(ctx, userID)
			if err != nil {
				return fmt.Errorf("failed to resolve user: %w", err)
			}

			stageEnum := parseStage(stage)
			req := &pb.ListTasksRequest{
				Stage:  stageEnum,
				UserId: resolvedUserID,
			}

			resp, err := client.ListTasks(ctx, req)
			if err != nil {
				return fmt.Errorf("list tasks operation failed: %w", err)
			}

			fmt.Printf("Tasks for user %s (%d total):\n", resolvedUserID, len(resp.Tasks))
			for _, task := range resp.Tasks {
				// Show both full ID and minimum unique prefix
				prefix := taskResolver.GetMinimumUniquePrefix(task.Id)
				fmt.Printf("  %s (%s): %s - %s [%s]\n",
					task.Id, prefix, task.Name, task.Description, task.Stage.String())
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&stage, "stage", "s", "", "Filter by stage (pending, active, done, staging)")
	cmd.Flags().StringVarP(&userID, "user", "u", "", "User ID or name (overrides current user)")

	return cmd
}

func newGetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <task-id-or-prefix>",
		Short: "Get task details by ID or unique prefix",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			taskInput := args[0]

			// Resolve task ID
			resolvedTaskID, err := resolveTaskID(ctx, taskInput)
			if err != nil {
				return err
			}

			req := &pb.GetTaskRequest{Id: resolvedTaskID}
			resp, err := client.GetTask(ctx, req)
			if err != nil {
				return fmt.Errorf("get task operation failed for task '%s': %w", resolvedTaskID, err)
			}

			task := resp.Task
			fmt.Printf("Task Details:\n")
			fmt.Printf("  ID: %s\n", task.Id)
			fmt.Printf("  Short ID: %s\n", taskResolver.GetMinimumUniquePrefix(task.Id))
			fmt.Printf("  Name: %s\n", task.Name)
			fmt.Printf("  Description: %s\n", task.Description)
			fmt.Printf("  Stage: %s\n", task.Stage.String())
			fmt.Printf("  Status: %s\n", task.Status.String())
			fmt.Printf("  User ID: %s\n", task.UserId)

			return nil
		},
	}

	return cmd
}

func newStartCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start [task-id-or-prefix]",
		Short: "Start a task (interactive selection if no ID provided)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			var taskID string
			var err error

			if len(args) == 0 {
				// Interactive task selection
				taskID, err = selectTaskInteractively(ctx, pb.TaskStage_STAGE_PENDING)
				if err != nil {
					return fmt.Errorf("task selection failed: %w", err)
				}
			} else {
				taskID, err = resolveTaskID(ctx, args[0])
				if err != nil {
					return err
				}
			}

			req := &pb.StartTaskRequest{Id: taskID}
			_, err = client.StartTask(ctx, req)
			if err != nil {
				return fmt.Errorf("start task operation failed for task '%s': %w", taskID, err)
			}

			fmt.Printf("Started task: %s\n", taskID)
			return nil
		},
	}

	return cmd
}

func newStopCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop [task-id-or-prefix]",
		Short: "Stop a task (interactive selection if no ID provided)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			var taskID string
			var err error

			if len(args) == 0 {
				// Interactive task selection
				taskID, err = selectTaskInteractively(ctx, pb.TaskStage_STAGE_ACTIVE)
				if err != nil {
					return fmt.Errorf("task selection failed: %w", err)
				}
			} else {
				taskID, err = resolveTaskID(ctx, args[0])
				if err != nil {
					return err
				}
			}

			req := &pb.StopTaskRequest{Id: taskID}
			_, err = client.StopTask(ctx, req)
			if err != nil {
				return fmt.Errorf("stop task operation failed for task '%s': %w", taskID, err)
			}

			fmt.Printf("Stopped task: %s\n", taskID)
			return nil
		},
	}

	return cmd
}

func newCompleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "complete [task-id-or-prefix]",
		Short: "Complete a task (interactive selection if no ID provided)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			var taskID string
			var err error

			if len(args) == 0 {
				// Interactive task selection - can complete active or pending tasks
				taskID, err = selectTaskInteractively(ctx, pb.TaskStage_STAGE_UNSPECIFIED) // Any stage
				if err != nil {
					return fmt.Errorf(taskSelectionFailed, err)
				}
			} else {
				taskID, err = resolveTaskID(ctx, args[0])
				if err != nil {
					return err
				}
			}

			req := &pb.CompleteTaskRequest{Id: taskID}
			_, err = client.CompleteTask(ctx, req)
			if err != nil {
				return fmt.Errorf("complete task operation failed for task '%s': %w", taskID, err)
			}

			fmt.Printf("Completed task: %s\n", taskID)
			return nil
		},
	}

	return cmd
}

func newStageCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stage <task-id-or-prefix>",
		Short: "Move task to staging",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			taskInput := args[0]

			resolvedTaskID, err := resolveTaskID(ctx, taskInput)
			if err != nil {
				return err
			}

			req := &pb.MoveToStagingRequest{SourceId: resolvedTaskID}
			_, err = client.MoveToStaging(ctx, req)
			if err != nil {
				return fmt.Errorf("move to staging operation failed for task '%s': %w", resolvedTaskID, err)
			}

			fmt.Printf("Moved task to staging: %s\n", resolvedTaskID)
			return nil
		},
	}

	return cmd
}

func newTagCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tag <task-id-or-prefix>",
		Short: "View task tags and structure",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			taskInput := args[0]

			resolvedTaskID, err := resolveTaskID(ctx, taskInput)
			if err != nil {
				return err
			}

			req := &pb.GetTaskRequest{Id: resolvedTaskID}
			resp, err := client.GetTask(ctx, req)
			if err != nil {
				return fmt.Errorf("get task operation failed for task '%s': %w", resolvedTaskID, err)
			}

			task := resp.Task
			fmt.Printf("Tags and Structure for task: %s\n", task.Name)

			// Display tags
			if len(task.Tags) > 0 {
				fmt.Printf("Tags:\n")
				for key, value := range task.Tags {
					fmt.Printf("  %s: %s\n", key, formatTagValue(value))
				}
			} else {
				fmt.Printf("No tags set\n")
			}

			// Display location hierarchy
			if len(task.Location) > 0 {
				fmt.Printf("Location: %s\n", strings.Join(task.Location, " > "))
			}

			return nil
		},
	}

	return cmd
}

func newUserCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user",
		Short: "User management commands",
	}

	// Create user
	cmd.AddCommand(&cobra.Command{
		Use:   "create <email> <name>",
		Short: "Create a new user",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			email := args[0]
			name := args[1]

			req := &pb.CreateUserRequest{
				Email: email,
				Name:  name,
			}

			resp, err := client.CreateUser(ctx, req)
			if err != nil {
				return fmt.Errorf("create user operation failed: %w", err)
			}

			fmt.Printf("Created user: %s (%s)\n", resp.User.Name, resp.User.Email)
			fmt.Printf("ID: %s\n", resp.User.Id)

			// Update resolvers with new user
			if err := updateResolvers(); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Failed to update resolvers: %v\n", err)
			}

			return nil
		},
	})

	// Get user
	cmd.AddCommand(&cobra.Command{
		Use:   "get <user-id-or-name>",
		Short: "Get user details by ID or name",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			userInput := args[0]

			resolvedUserID, err := resolveUserID(ctx, userInput)
			if err != nil {
				return err
			}

			req := &pb.GetUserRequest{UserId: resolvedUserID}
			resp, err := client.GetUser(ctx, req)
			if err != nil {
				return fmt.Errorf("get user operation failed for user '%s': %w", resolvedUserID, err)
			}

			user := resp.User
			fmt.Printf("User: %s (%s)\n", user.Name, user.Email)
			fmt.Printf("ID: %s\n", user.Id)
			if len(user.NotificationSettings) > 0 {
				fmt.Printf("Notification Settings:\n")
				for _, setting := range user.NotificationSettings {
					fmt.Printf("  %s: %s\n", setting.Type.String(), setting.String())
				}
			}
			return nil
		},
	})

	return cmd
}

func newSyncCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync [user-id-or-name]",
		Short: "Sync tasks with Google Calendar (uses current user if not specified)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			var userInput string
			if len(args) > 0 {
				userInput = args[0]
			}

			resolvedUserID, err := resolveUserID(ctx, userInput)
			if err != nil {
				return err
			}

			req := &pb.SyncCalendarRequest{UserId: resolvedUserID}
			_, err = client.SyncCalendar(ctx, req)
			if err != nil {
				return fmt.Errorf("calendar sync operation failed for user '%s': %w", resolvedUserID, err)
			}

			fmt.Printf("Calendar sync completed for user: %s\n", resolvedUserID)
			return nil
		},
	}

	return cmd
}

func newDAGCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dag [user-id-or-name]",
		Short: "View task dependency graph (uses current user if not specified)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			var userInput string
			if len(args) > 0 {
				userInput = args[0]
			}

			resolvedUserID, err := resolveUserID(ctx, userInput)
			if err != nil {
				return err
			}

			req := &pb.GetTaskDAGRequest{UserId: resolvedUserID}
			resp, err := client.GetTaskDAG(ctx, req)
			if err != nil {
				return fmt.Errorf("get task DAG operation failed for user '%s': %w", resolvedUserID, err)
			}
			// Convert tasks to domain format for rendering
			domainTasks := make([]*domain.Task, len(resp.Tasks))
			for i, protoTask := range resp.Tasks {
				domainTasks[i] = protoTaskToDomain(protoTask)
			}

			// Render the DAG
			renderer := dagview.NewDAGRenderer()
			renderer.BuildGraph(domainTasks)
			output := renderer.RenderASCII()

			fmt.Printf("Task Dependency Graph for user %s:\n", resolvedUserID)
			fmt.Print(output)
			return nil
		},
	}

	return cmd
}

// Helper functions

func selectTaskInteractively(ctx context.Context, stage pb.TaskStage) (string, error) {
	req := &pb.ListTasksRequest{
		Stage:  stage,
		UserId: currentUser,
	}

	resp, err := client.ListTasks(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to list tasks for selection: %w", err)
	}

	if len(resp.Tasks) == 0 {
		return "", fmt.Errorf("no tasks found for selection")
	}

	// Simple selection for now - just take the first task
	// In a real implementation, you'd implement a proper selection UI
	task := resp.Tasks[0]
	prefix := taskResolver.GetMinimumUniquePrefix(task.Id)
	fmt.Printf("Selected task: %s (%s): %s\n", task.Id, prefix, task.Name)

	return task.Id, nil
}

func parseStage(stage string) pb.TaskStage {
	switch strings.ToLower(stage) {
	case "pending":
		return pb.TaskStage_STAGE_PENDING
	case "inbox":
		return pb.TaskStage_STAGE_INBOX
	case "active":
		return pb.TaskStage_STAGE_ACTIVE
	case "staging":
		return pb.TaskStage_STAGE_STAGING
	case "archived":
		return pb.TaskStage_STAGE_ARCHIVED
	default:
		return pb.TaskStage_STAGE_UNSPECIFIED
	}
}

func formatTagValue(tagValue *pb.TagValue) string {
	switch v := tagValue.Value.(type) {
	case *pb.TagValue_TextValue:
		return v.TextValue
	default:
		return "unknown"
	}
}

// Conversion functions
func protoTaskToDomain(protoTask *pb.Task) *domain.Task {
	domainTask := &domain.Task{
		ID:          protoTask.Id,
		Name:        protoTask.Name,
		Description: protoTask.Description,
		UserID:      protoTask.UserId,
		Location:    protoTask.Location,
		Inflows:     protoTask.Inflows,
		Outflows:    protoTask.Outflows,
	}

	// Convert stage
	switch protoTask.Stage {
	case pb.TaskStage_STAGE_PENDING:
		domainTask.Stage = domain.StagePending
	case pb.TaskStage_STAGE_INBOX:
		domainTask.Stage = domain.StageInbox
	case pb.TaskStage_STAGE_ACTIVE:
		domainTask.Stage = domain.StageActive
	case pb.TaskStage_STAGE_STAGING:
		domainTask.Stage = domain.StageStaging
	case pb.TaskStage_STAGE_ARCHIVED:
		domainTask.Stage = domain.StageArchived
	}

	// Convert status
	switch protoTask.Status {
	case pb.TaskStatus_TASK_STATUS_TODO:
		domainTask.Status = domain.StatusTodo
	case pb.TaskStatus_TASK_STATUS_IN_PROGRESS:
		domainTask.Status = domain.StatusInProgress
	case pb.TaskStatus_TASK_STATUS_PAUSED:
		domainTask.Status = domain.StatusPaused
	case pb.TaskStatus_TASK_STATUS_BLOCKED:
		domainTask.Status = domain.StatusBlocked
	case pb.TaskStatus_TASK_STATUS_COMPLETED:
		domainTask.Status = domain.StatusCompleted
	case pb.TaskStatus_TASK_STATUS_CANCELLED:
		domainTask.Status = domain.StatusCancelled
	}

	// Convert tags
	domainTask.Tags = make(map[string]domain.TagValue)
	for key, protoTag := range protoTask.Tags {
		domainTask.Tags[key] = protoTagToDomain(protoTag)
	}

	return domainTask
}

func protoTagToDomain(protoTag *pb.TagValue) domain.TagValue {
	switch v := protoTag.Value.(type) {
	case *pb.TagValue_TextValue:
		return domain.TagValue{Type: domain.TagTypeText, TextValue: v.TextValue}
	default:
		return domain.TagValue{Type: domain.TagTypeText, TextValue: ""}
	}
}

func protoUserToDomain(protoUser *pb.User) *domain.User {
	return &domain.User{
		ID:                  protoUser.Id,
		Email:               protoUser.Email,
		Name:                protoUser.Name,
		GoogleCalendarToken: protoUser.GoogleCalendarToken,
	}
}
