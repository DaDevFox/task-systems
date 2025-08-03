package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	fuzzyfinder "github.com/ktr0731/go-fuzzyfinder"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/DaDevFox/task-systems/task-core/internal/config"
	"github.com/DaDevFox/task-systems/task-core/internal/dagview"
	"github.com/DaDevFox/task-systems/task-core/internal/domain"
	pb "github.com/DaDevFox/task-systems/task-core/proto/taskcore/v1"
)

var (
	serverAddr  string
	currentUser string
	userFlag    string
	client      pb.TaskServiceClient
	conn        *grpc.ClientConn
	cfg         *config.Config
)

// Constants for repeated strings
const (
	taskSelectionFailedMsg = "Task selection failed: %v"
	formatTwoStringMsg     = "  %s: %s\n"
	formatIDMsg            = "ID: %s\n"
	defaultUserID          = "default-user"
	failedSaveConfigMsg    = "Failed to save config: %v"
	failedResolveTaskIDMsg = "Failed to resolve task ID: %v"
	failedResolveUserMsg   = "Failed to resolve user: %v"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "tasker",
		Short: "Task management CLI client",
		Long:  "A comprehensive task management CLI client with advanced features",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Load configuration
			var err error
			cfg, err = config.LoadConfig()
			if err != nil {
				log.Printf("Warning: Failed to load config: %v", err)
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
				log.Fatalf("Failed to connect to server: %v", err)
			}
			client = pb.NewTaskServiceClient(conn)
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
		fmt.Println(err)
		os.Exit(1)
	}
}

func newAddCommand() *cobra.Command {
	var description string
	var userID string

	cmd := &cobra.Command{
		Use:   "add <task-name>",
		Short: "Add a new task",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			name := args[0]

			// Resolve user ID if provided, otherwise use current user
			var resolvedUserID string
			if userID != "" {
				var err error
				resolvedUserID, err = resolveUserInput(ctx, userID)
				if err != nil {
					log.Fatalf("Failed to resolve user: %v", err)
				}
			} else {
				resolvedUserID = currentUser
			}

			req := &pb.AddTaskRequest{
				Name:        name,
				Description: description,
				UserId:      resolvedUserID,
			}

			resp, err := client.AddTask(ctx, req)
			if err != nil {
				log.Fatalf("AddTask failed: %v", err)
			}

			fmt.Printf("Created task: %s (ID: %s)\n", resp.Task.Name, resp.Task.Id)
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
		Run: func(cmd *cobra.Command, args []string) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			// Use provided user or current user
			resolvedUserID := currentUser
			if userID != "" {
				var err error
				resolvedUserID, err = resolveUserInput(ctx, userID)
				if err != nil {
					log.Fatalf("Failed to resolve user: %v", err)
				}
			}

			stageEnum := parseStage(stage)
			req := &pb.ListTasksRequest{
				Stage:  stageEnum,
				UserId: resolvedUserID,
			}

			resp, err := client.ListTasks(ctx, req)
			if err != nil {
				log.Fatalf("ListTasks failed: %v", err)
			}

			fmt.Printf("Tasks for user %s (%d total):\n", resolvedUserID, len(resp.Tasks))
			for _, task := range resp.Tasks {
				fmt.Printf("  %s: %s - %s [%s]\n", task.Id, task.Name, task.Description, task.Stage.String())
			}
		},
	}

	cmd.Flags().StringVarP(&stage, "stage", "s", "pending", "Task stage (pending, staging, active, completed)")
	cmd.Flags().StringVarP(&userID, "user", "u", "", "User ID or name (uses current user if not specified)")

	return cmd
}

func newGetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <task-id-or-prefix>",
		Short: "Get task details (supports partial ID matching)",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			taskInput := args[0]

			// Resolve task ID using our resolver
			resolvedTaskID, err := resolveTaskInput(ctx, taskInput)
			if err != nil {
				log.Fatalf(failedResolveTaskIDMsg, err)
			}

			req := &pb.GetTaskRequest{Id: resolvedTaskID}
			resp, err := client.GetTask(ctx, req)
			if err != nil {
				log.Fatalf("GetTask failed: %v", err)
			}

			printTaskDetails(resp.Task)
		},
	}

	return cmd
}

func newStartCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start [task-id-or-prefix]",
		Short: "Start a task (with fuzzy picker if no ID provided, supports partial ID matching)",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			var taskID string
			if len(args) == 0 {
				// Use fuzzy picker to select task
				var err error
				taskID, err = fuzzySelectTask(ctx, pb.TaskStage_STAGE_STAGING)
				if err != nil {
					log.Fatalf(taskSelectionFailedMsg, err)
				}
			} else {
				// Resolve task ID using our resolver
				var err error
				taskID, err = resolveTaskInput(ctx, args[0])
				if err != nil {
					log.Fatalf(failedResolveTaskIDMsg, err)
				}
			}

			req := &pb.StartTaskRequest{Id: taskID}
			resp, err := client.StartTask(ctx, req)
			if err != nil {
				log.Fatalf("StartTask failed: %v", err)
			}

			fmt.Printf("Started task: %s\n", resp.Task.Name)
		},
	}

	return cmd
}

func newStopCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop [task-id-or-prefix]",
		Short: "Stop a task (with fuzzy picker if no ID provided, supports partial ID matching)",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			var taskID string
			if len(args) == 0 {
				// Use fuzzy picker to select task
				var err error
				taskID, err = fuzzySelectTask(ctx, pb.TaskStage_STAGE_ACTIVE)
				if err != nil {
					log.Fatalf(taskSelectionFailedMsg, err)
				}
			} else {
				// Resolve task ID using our resolver
				var err error
				taskID, err = resolveTaskInput(ctx, args[0])
				if err != nil {
					log.Fatalf(failedResolveTaskIDMsg, err)
				}
			}

			req := &pb.StopTaskRequest{Id: taskID}
			resp, err := client.StopTask(ctx, req)
			if err != nil {
				log.Fatalf("StopTask failed: %v", err)
			}

			fmt.Printf("Stopped task: %s (Completed: %t)\n", resp.Task.Name, resp.Completed)
		},
	}

	return cmd
}

func newCompleteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "complete [task-id-or-prefix]",
		Short: "Complete a task (with fuzzy picker if no ID provided, supports partial ID matching)",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			var taskID string
			if len(args) == 0 {
				// Use fuzzy picker to select task
				var err error
				taskID, err = fuzzySelectTask(ctx, pb.TaskStage_STAGE_ACTIVE)
				if err != nil {
					log.Fatalf(taskSelectionFailedMsg, err)
				}
			} else {
				// Resolve task ID using our resolver
				var err error
				taskID, err = resolveTaskInput(ctx, args[0])
				if err != nil {
					log.Fatalf(failedResolveTaskIDMsg, err)
				}
			}

			req := &pb.CompleteTaskRequest{Id: taskID}
			resp, err := client.CompleteTask(ctx, req)
			if err != nil {
				log.Fatalf("CompleteTask failed: %v", err)
			}

			fmt.Printf("Completed task: %s\n", resp.Task.Name)
		},
	}

	return cmd
}

func newStageCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stage",
		Short: "Stage management commands",
	}

	moveCmd := &cobra.Command{
		Use:   "move <source-task-id> [flags]",
		Short: "Move task to staging (supports partial ID matching)",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			taskInput := args[0]

			// Resolve source task ID
			resolvedTaskID, err := resolveTaskInput(ctx, taskInput)
			if err != nil {
				log.Fatalf(failedResolveTaskIDMsg, err)
			}

			req := &pb.MoveToStagingRequest{SourceId: resolvedTaskID}

			// Get destination flag
			destinationID, _ := cmd.Flags().GetString("destination")
			newLocation, _ := cmd.Flags().GetStringSlice("location")

			if destinationID != "" {
				// Resolve destination task ID
				resolvedDestID, err := resolveTaskInput(ctx, destinationID)
				if err != nil {
					log.Fatalf("Failed to resolve destination task ID: %v", err)
				}
				req.Destination = &pb.MoveToStagingRequest_DestinationId{
					DestinationId: resolvedDestID,
				}
			} else if len(newLocation) > 0 {
				req.Destination = &pb.MoveToStagingRequest_NewLocation{
					NewLocation: &pb.MoveToStagingRequest_NewLocationList{
						NewLocation: newLocation,
					},
				}
			}

			resp, err := client.MoveToStaging(ctx, req)
			if err != nil {
				log.Fatalf("MoveToStaging failed: %v", err)
			}

			fmt.Printf("Moved task to staging: %s\n", resp.Task.Name)
			if destinationID != "" {
				fmt.Printf("Destination: Task %s\n", destinationID)
			} else if len(newLocation) > 0 {
				fmt.Printf("Location: %v\n", newLocation)
			}
		},
	}

	// Add flags for destination specification
	moveCmd.Flags().StringP("destination", "d", "", "Destination task ID or prefix (for dependencies)")
	moveCmd.Flags().StringSliceP("location", "l", []string{}, "New location path (e.g., --location project --location backend)")

	cmd.AddCommand(moveCmd)

	return cmd
}

func newTagCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tag",
		Short: "Tag management commands",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "update <task-id-or-prefix>",
		Short: "Update task tags (shows current tags, supports partial ID matching)",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			taskInput := args[0]

			// Resolve task ID
			resolvedTaskID, err := resolveTaskInput(ctx, taskInput)
			if err != nil {
				log.Fatalf(failedResolveTaskIDMsg, err)
			}

			// Get current task to show existing tags
			getReq := &pb.GetTaskRequest{Id: resolvedTaskID}
			getResp, err := client.GetTask(ctx, getReq)
			if err != nil {
				log.Fatalf("Failed to get task: %v", err)
			}

			fmt.Printf("Current tags for task '%s':\n", getResp.Task.Name)
			for key, value := range getResp.Task.Tags {
				fmt.Printf(formatTwoStringMsg, key, value.String())
			}

			fmt.Printf("\nTo update tags, use the UpdateTaskTags RPC directly\n")
		},
	})

	return cmd
}

func newUserCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user",
		Short: "User management commands",
	}

	// Add create subcommand
	cmd.AddCommand(&cobra.Command{
		Use:   "create <email> <name>",
		Short: "Create a new user",
		Args:  cobra.ExactArgs(2),
		Run: func(cmd *cobra.Command, args []string) {
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
				log.Fatalf("CreateUser failed: %v", err)
			}
			fmt.Printf("Created user: %s (%s)\n", resp.User.Name, resp.User.Email)
			fmt.Printf(formatIDMsg, resp.User.Id)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "get <user-id-or-email>",
		Short: "Get user details by ID or email",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			userInput := args[0]
			var req *pb.GetUserRequest

			// Check if input looks like an email
			if strings.Contains(userInput, "@") {
				req = &pb.GetUserRequest{
					Identifier: &pb.GetUserRequest_Email{Email: userInput},
				}
			} else {
				req = &pb.GetUserRequest{
					Identifier: &pb.GetUserRequest_UserId{UserId: userInput},
				}
			}

			resp, err := client.GetUser(ctx, req)
			if err != nil {
				log.Fatalf("GetUser failed: %v", err)
			}
			fmt.Printf("User: %s (%s)\n", resp.User.Name, resp.User.Email)
			fmt.Printf(formatIDMsg, resp.User.Id)
			if len(resp.User.NotificationSettings) > 0 {
				fmt.Printf("Notification Settings:\n")
				for _, setting := range resp.User.NotificationSettings {
					fmt.Printf("  %s: %s\n", setting.Type.String(), setting.String())
				}
			}
		},
	})

	return cmd
}

func newSyncCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync <user-id>",
		Short: "Sync tasks with Google Calendar",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			userID := args[0]
			req := &pb.SyncCalendarRequest{UserId: userID}

			resp, err := client.SyncCalendar(ctx, req)
			if err != nil {
				log.Fatalf("SyncCalendar failed: %v", err)
			}

			fmt.Printf("Calendar sync completed for user %s\n", userID)
			fmt.Printf("Synced %d tasks\n", resp.TasksSynced)
			if len(resp.Errors) > 0 {
				fmt.Printf("Errors:\n")
				for _, errMsg := range resp.Errors {
					fmt.Printf("  %s\n", errMsg)
				}
			}
		},
	}

	return cmd
}

func newDAGCommand() *cobra.Command {
	var userID string
	var compact bool
	var watch bool

	cmd := &cobra.Command{
		Use:   "dag",
		Short: "View task dependency graph (DAG)",
		Long:  "Display tasks in a directed acyclic graph showing dependencies and relationships",
		Run: func(cmd *cobra.Command, args []string) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			// Use provided user or current user
			resolvedUserID := currentUser
			if userID != "" {
				var err error
				resolvedUserID, err = resolveUserInput(ctx, userID)
				if err != nil {
					log.Fatalf("Failed to resolve user: %v", err)
				}
			}

			req := &pb.GetTaskDAGRequest{UserId: resolvedUserID}
			resp, err := client.GetTaskDAG(ctx, req)
			if err != nil {
				log.Fatalf("GetTaskDAG failed: %v", err)
			}

			if len(resp.Tasks) == 0 {
				fmt.Println("No tasks found for DAG visualization")
				return
			}

			// Convert protobuf tasks to domain tasks for renderer
			tasks := make([]*domain.Task, len(resp.Tasks))
			for i, protoTask := range resp.Tasks {
				tasks[i] = protoTaskToDomain(protoTask)
			}

			// Create and render DAG
			renderer := dagview.NewDAGRenderer()
			renderer.BuildGraph(tasks)

			if compact {
				fmt.Println(renderer.RenderCompact())
			} else {
				fmt.Println(renderer.RenderASCII())
			}

			// Show stats
			stats := renderer.GetStats()
			fmt.Printf("\nDAG Stats: %d nodes, %d root tasks, depth: %d\n",
				stats["total_tasks"], stats["root_tasks"], stats["max_level"])
		},
	}

	cmd.Flags().StringVarP(&userID, "user", "u", "", "User ID or name (uses current user if not specified)")
	cmd.Flags().BoolVarP(&compact, "compact", "c", false, "Use compact rendering format")
	cmd.Flags().BoolVarP(&watch, "watch", "w", false, "Watch for changes (not implemented yet)")

	return cmd
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
		Run: func(cmd *cobra.Command, args []string) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			userInput := args[0]

			// Try to resolve user by name or ID
			resolvedUserID, err := resolveUserInput(ctx, userInput)
			if err != nil {
				log.Fatalf("Failed to resolve user: %v", err)
			}

			cfg.CurrentUser = resolvedUserID
			if err := cfg.SaveConfig(); err != nil {
				log.Fatalf(failedSaveConfigMsg, err)
			}

			fmt.Printf("Current user set to: %s\n", resolvedUserID)
		},
	})

	// Set server address
	cmd.AddCommand(&cobra.Command{
		Use:   "set-server <address>",
		Short: "Set the server address",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			cfg.ServerAddr = args[0]
			if err := cfg.SaveConfig(); err != nil {
				log.Fatalf(failedSaveConfigMsg, err)
			}

			fmt.Printf("Server address set to: %s\n", args[0])
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
		Run: func(cmd *cobra.Command, args []string) {
			cfg = config.DefaultConfig()
			if err := cfg.SaveConfig(); err != nil {
				log.Fatalf(failedSaveConfigMsg, err)
			}

			fmt.Println("Configuration reset to defaults")
		},
	})

	return cmd
}

// Helper functions

func fuzzySelectTask(ctx context.Context, stage pb.TaskStage) (string, error) {
	req := &pb.ListTasksRequest{Stage: stage}
	resp, err := client.ListTasks(ctx, req)
	if err != nil {
		return "", err
	}

	if len(resp.Tasks) == 0 {
		return "", fmt.Errorf("no tasks found in %s stage", stage.String())
	}

	idx, err := fuzzyfinder.Find(
		resp.Tasks,
		func(i int) string {
			return fmt.Sprintf("%s: %s", resp.Tasks[i].Id, resp.Tasks[i].Name)
		},
	)
	if err != nil {
		return "", err
	}

	return resp.Tasks[idx].Id, nil
}

func parseStage(stage string) pb.TaskStage {
	switch strings.ToLower(stage) {
	case "pending":
		return pb.TaskStage_STAGE_PENDING
	case "inbox":
		return pb.TaskStage_STAGE_INBOX
	case "staging":
		return pb.TaskStage_STAGE_STAGING
	case "active":
		return pb.TaskStage_STAGE_ACTIVE
	case "archived":
		return pb.TaskStage_STAGE_ARCHIVED
	default:
		return pb.TaskStage_STAGE_PENDING
	}
}

func printTaskDetails(task *pb.Task) {
	fmt.Printf("Task Details:\n")
	fmt.Printf(formatIDMsg, task.Id)
	fmt.Printf("Name: %s\n", task.Name)
	fmt.Printf("Description: %s\n", task.Description)
	fmt.Printf("Stage: %s\n", task.Stage.String())
	fmt.Printf("Status: %s\n", task.Status.String())
	fmt.Printf("User ID: %s\n", task.UserId)

	if len(task.Location) > 0 {
		fmt.Printf("Location: %s\n", strings.Join(task.Location, " > "))
	}

	if len(task.Tags) > 0 {
		fmt.Printf("Tags:\n")
		for key, value := range task.Tags {
			fmt.Printf(formatTwoStringMsg, key, value.String())
		}
	}

	if len(task.Inflows) > 0 {
		fmt.Printf("Dependencies: %s\n", strings.Join(task.Inflows, ", "))
	}

	if len(task.Outflows) > 0 {
		fmt.Printf("Dependents: %s\n", strings.Join(task.Outflows, ", "))
	}

	if task.GoogleCalendarEventId != "" {
		fmt.Printf("Calendar Event ID: %s\n", task.GoogleCalendarEventId)
	}
}

// protoTaskToDomain converts a protobuf task to a domain task
func protoTaskToDomain(protoTask *pb.Task) *domain.Task {
	task := &domain.Task{
		ID:                    protoTask.Id,
		Name:                  protoTask.Name,
		Description:           protoTask.Description,
		UserID:                protoTask.UserId,
		Stage:                 protoStageToDomain(protoTask.Stage),
		Status:                protoStatusToDomain(protoTask.Status),
		Location:              protoTask.Location,
		Inflows:               protoTask.Inflows,
		Outflows:              protoTask.Outflows,
		GoogleCalendarEventID: protoTask.GoogleCalendarEventId,
	}

	// Convert tags
	task.Tags = make(map[string]domain.TagValue)
	for key, protoTag := range protoTask.Tags {
		task.Tags[key] = protoTagToDomain(protoTag)
	}

	return task
}

// protoStageToDomain converts protobuf stage to domain stage
func protoStageToDomain(stage pb.TaskStage) domain.TaskStage {
	switch stage {
	case pb.TaskStage_STAGE_INBOX:
		return domain.StageInbox
	case pb.TaskStage_STAGE_PENDING:
		return domain.StagePending
	case pb.TaskStage_STAGE_STAGING:
		return domain.StageStaging
	case pb.TaskStage_STAGE_ACTIVE:
		return domain.StageActive
	case pb.TaskStage_STAGE_ARCHIVED:
		return domain.StageArchived
	default:
		return domain.StagePending
	}
}

// protoStatusToDomain converts protobuf status to domain status
func protoStatusToDomain(status pb.TaskStatus) domain.TaskStatus {
	switch status {
	case pb.TaskStatus_TASK_STATUS_TODO:
		return domain.StatusTodo
	case pb.TaskStatus_TASK_STATUS_IN_PROGRESS:
		return domain.StatusInProgress
	case pb.TaskStatus_TASK_STATUS_PAUSED:
		return domain.StatusPaused
	case pb.TaskStatus_TASK_STATUS_BLOCKED:
		return domain.StatusBlocked
	case pb.TaskStatus_TASK_STATUS_COMPLETED:
		return domain.StatusCompleted
	case pb.TaskStatus_TASK_STATUS_CANCELLED:
		return domain.StatusCancelled
	default:
		return domain.StatusTodo
	}
}

// protoTagToDomain converts protobuf tag to domain tag
func protoTagToDomain(protoTag *pb.TagValue) domain.TagValue {
	switch protoTag.Type {
	case pb.TagType_TAG_TYPE_LOCATION:
		return domain.TagValue{
			Type:      domain.TagTypeLocation,
			TextValue: protoTag.GetTextValue(), // Store as text for now
		}
	case pb.TagType_TAG_TYPE_TIME:
		return domain.TagValue{
			Type:      domain.TagTypeTime,
			TextValue: protoTag.GetTextValue(), // Store as text for now
		}
	case pb.TagType_TAG_TYPE_TEXT:
		fallthrough
	default:
		return domain.TagValue{
			Type:      domain.TagTypeText,
			TextValue: protoTag.GetTextValue(),
		}
	}
}

// Helper function to resolve user input by ID or name using server-side resolution
func resolveUserInput(ctx context.Context, userInput string) (string, error) {
	if userInput == "" {
		return currentUser, nil
	}

	// Use server-side user resolution
	req := &pb.ResolveUserIDRequest{UserInput: userInput}
	resp, err := client.ResolveUserID(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to resolve user '%s': %w", userInput, err)
	}

	return resp.ResolvedId, nil
}

// Helper function to resolve task input using server-side ID resolution
func resolveTaskInput(ctx context.Context, taskInput string) (string, error) {
	if taskInput == "" {
		return "", fmt.Errorf("empty task ID provided")
	}

	// Use server-side task resolution
	req := &pb.ResolveTaskIDRequest{TaskInput: taskInput}
	resp, err := client.ResolveTaskID(ctx, req)
	if err != nil {
		return "", fmt.Errorf("failed to resolve task '%s': %w", taskInput, err)
	}

	return resp.ResolvedId, nil
}
