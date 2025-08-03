package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/DaDevFox/task-systems/task-core/internal/config"
	"github.com/DaDevFox/task-systems/task-core/internal/idresolver"
	pb "github.com/DaDevFox/task-systems/task-core/proto/taskcore/v1"
)

var (
	serverAddr   string
	userID       string
	client       pb.TaskServiceClient
	conn         *grpc.ClientConn
	taskResolver *idresolver.TaskIDResolver
	userResolver *idresolver.UserResolver
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "tasker",
		Short: "Enhanced task management CLI client",
		Long:  "A comprehensive task management CLI client with ID resolution, current user support, and enhanced features",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			initClient()
		},
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			if conn != nil {
				conn.Close()
			}
		},
	}

	// Global flags
	rootCmd.PersistentFlags().StringVar(&serverAddr, "server", "localhost:8080", "Server address")
	rootCmd.PersistentFlags().StringVarP(&userID, "user", "u", "", "User ID or name (optional if current user is set)")

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
		log.Fatal(err)
	}
}

func initClient() {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Printf("Warning: failed to load config: %v", err)
		cfg = config.DefaultConfig()
	}

	// Use server from config if not specified
	if serverAddr == "" || serverAddr == "localhost:8080" {
		if cfg.ServerAddr != "" {
			serverAddr = cfg.ServerAddr
		}
	}

	// Resolve user ID
	resolvedUserID, err := resolveUserID(cfg)
	if err != nil {
		log.Fatalf("Failed to resolve user: %v", err)
	}
	userID = resolvedUserID

	// Connect to server
	conn, err = grpc.Dial(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to server: %v", err)
	}

	client = pb.NewTaskServiceClient(conn)

	// Initialize resolvers
	taskResolver = idresolver.NewTaskIDResolver()
	userResolver = idresolver.NewUserResolver()

	// Update resolvers with current data
	updateResolvers()
}

func resolveUserID(cfg *config.Config) (string, error) {
	// If user is explicitly provided, use it
	if userID != "" {
		return userID, nil
	}

	// If current user is set in config, use it
	if cfg.CurrentUser != "" {
		return cfg.CurrentUser, nil
	}

	// No user specified and no current user configured
	return "", fmt.Errorf("no user specified. Use --user flag or set current user with 'tasker config current-user <user>'")
}

func updateResolvers() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Update user resolver
	resp, err := client.ListTasks(ctx, &pb.ListTasksRequest{
		Stage:  pb.TaskStage_STAGE_PENDING, // Get any stage to list users
		UserId: userID,
	})
	if err == nil && len(resp.Tasks) > 0 {
		// Extract unique users from tasks
		userMap := make(map[string]*pb.User)
		for _, task := range resp.Tasks {
			if task.UserId != "" {
				// We'd need to get user details, but for now just store IDs
				userMap[task.UserId] = &pb.User{Id: task.UserId}
			}
		}

		// Convert to domain users (simplified)
		// In a real implementation, you'd fetch full user details
	}

	// Update task resolver with all user's tasks
	allTasks := []*pb.Task{}
	stages := []pb.TaskStage{
		pb.TaskStage_STAGE_PENDING,
		pb.TaskStage_STAGE_INBOX,
		pb.TaskStage_STAGE_STAGING,
		pb.TaskStage_STAGE_ACTIVE,
		pb.TaskStage_STAGE_ARCHIVED,
	}

	for _, stage := range stages {
		resp, err := client.ListTasks(ctx, &pb.ListTasksRequest{
			Stage:  stage,
			UserId: userID,
		})
		if err == nil {
			allTasks = append(allTasks, resp.Tasks...)
		}
	}

	// Convert to domain tasks and update resolver
	// This is simplified - in a real implementation you'd convert properly
	if len(allTasks) > 0 {
		// taskResolver.UpdateTasks(domainTasks)
	}
}

func resolveTaskID(partialID string) (string, error) {
	if taskResolver == nil {
		updateResolvers()
	}

	// For now, just return the partial ID as-is
	// In a full implementation, this would use the trie-based resolver
	return partialID, nil
}

func resolveUser(identifier string) (string, error) {
	if userResolver == nil {
		updateResolvers()
	}

	// For now, just return the identifier as-is
	// In a full implementation, this would use the user resolver
	return identifier, nil
}

// Configuration command
func newConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage client configuration",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "current-user <user-id-or-name>",
		Short: "Set the current user",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			userIdentifier := args[0]

			// Resolve user ID
			resolvedUserID, err := resolveUser(userIdentifier)
			if err != nil {
				log.Fatalf("Failed to resolve user: %v", err)
			}

			// Load current config
			cfg, err := config.LoadConfig()
			if err != nil {
				cfg = config.DefaultConfig()
			}

			// Update current user
			cfg.CurrentUser = resolvedUserID
			cfg.ServerAddr = serverAddr

			// Save config
			if err := cfg.SaveConfig(); err != nil {
				log.Fatalf("Failed to save config: %v", err)
			}

			fmt.Printf("Current user set to: %s\n", resolvedUserID)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "Show current configuration",
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := config.LoadConfig()
			if err != nil {
				log.Printf("Warning: failed to load config: %v", err)
				cfg = config.DefaultConfig()
			}

			fmt.Printf("Current Configuration:\n")
			fmt.Printf("  Server: %s\n", cfg.ServerAddr)
			fmt.Printf("  Current User: %s\n", cfg.CurrentUser)
		},
	})

	return cmd
}

// Enhanced commands with ID resolution
func newAddCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "add <name> [description]",
		Short: "Add a new task",
		Args:  cobra.RangeArgs(1, 2),
		Run: func(cmd *cobra.Command, args []string) {
			name := args[0]
			description := ""
			if len(args) > 1 {
				description = args[1]
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			req := &pb.AddTaskRequest{
				Name:        name,
				Description: description,
				UserId:      userID,
			}

			resp, err := client.AddTask(ctx, req)
			if err != nil {
				log.Fatalf("AddTask failed: %v", err)
			}

			fmt.Printf("✓ Created task: %s\n", resp.Task.Name)
			fmt.Printf("  ID: %s\n", resp.Task.Id)
			fmt.Printf("  Stage: %s\n", resp.Task.Stage.String())

			// Update resolvers
			updateResolvers()
		},
	}
}

func newGetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "get <task-id-or-prefix>",
		Short: "Get task details by ID or unique prefix",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			partialID := args[0]

			// Resolve task ID
			taskID, err := resolveTaskID(partialID)
			if err != nil {
				log.Fatalf("Failed to resolve task ID '%s': %v", partialID, err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			req := &pb.GetTaskRequest{Id: taskID}
			resp, err := client.GetTask(ctx, req)
			if err != nil {
				log.Fatalf("GetTask failed: %v", err)
			}

			task := resp.Task
			fmt.Printf("Task Details:\n")
			fmt.Printf("  ID: %s\n", task.Id)
			fmt.Printf("  Name: %s\n", task.Name)
			fmt.Printf("  Description: %s\n", task.Description)
			fmt.Printf("  Stage: %s\n", task.Stage.String())
			fmt.Printf("  Status: %s\n", task.Status.String())
			fmt.Printf("  User: %s\n", task.UserId)
			if len(task.Location) > 0 {
				fmt.Printf("  Location: %s\n", task.Location)
			}
			if len(task.Points) > 0 {
				fmt.Printf("  Points: %d total\n", len(task.Points))
			}
		},
	}
}

// Placeholder commands - implement similar to existing enhanced client
func newListCommand() *cobra.Command {
	var stage string

	cmd := &cobra.Command{
		Use:   "list [stage]",
		Short: "List tasks in a stage",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) > 0 {
				stage = args[0]
			}

			// Implementation similar to existing enhanced client
			fmt.Printf("List command not fully implemented yet for stage: %s\n", stage)
		},
	}

	cmd.Flags().StringVarP(&stage, "stage", "s", "pending", "Task stage")
	return cmd
}

func newStartCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "start <task-id-or-prefix>",
		Short: "Start working on a task",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			partialID := args[0]
			taskID, err := resolveTaskID(partialID)
			if err != nil {
				log.Fatalf("Failed to resolve task ID '%s': %v", partialID, err)
			}

			fmt.Printf("Start command not fully implemented yet for task: %s\n", taskID)
		},
	}
}

func newStopCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "stop <task-id-or-prefix>",
		Short: "Stop working on a task",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			partialID := args[0]
			taskID, err := resolveTaskID(partialID)
			if err != nil {
				log.Fatalf("Failed to resolve task ID '%s': %v", partialID, err)
			}

			fmt.Printf("Stop command not fully implemented yet for task: %s\n", taskID)
		},
	}
}

func newCompleteCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "complete <task-id-or-prefix>",
		Short: "Mark a task as completed",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			partialID := args[0]
			taskID, err := resolveTaskID(partialID)
			if err != nil {
				log.Fatalf("Failed to resolve task ID '%s': %v", partialID, err)
			}

			fmt.Printf("Complete command not fully implemented yet for task: %s\n", taskID)
		},
	}
}

func newStageCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "stage <task-id-or-prefix>",
		Short: "Move a task to staging",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			partialID := args[0]
			taskID, err := resolveTaskID(partialID)
			if err != nil {
				log.Fatalf("Failed to resolve task ID '%s': %v", partialID, err)
			}

			fmt.Printf("Stage command not fully implemented yet for task: %s\n", taskID)
		},
	}
}

func newTagCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "tags",
		Short: "Manage task tags",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Tags command not fully implemented yet")
		},
	}
}

func newUserCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "user",
		Short: "User management commands",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("User command not fully implemented yet")
		},
	}
}

func newSyncCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Sync with Google Calendar",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Sync command not fully implemented yet")
		},
	}
}

func newDAGCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "dag",
		Short: "Show dependency graph",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("DAG command not fully implemented yet")
		},
	}
}
