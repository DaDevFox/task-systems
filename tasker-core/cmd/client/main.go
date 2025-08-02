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

	"github.com/DaDevFox/task-systems/task-core/internal/dagview"
	"github.com/DaDevFox/task-systems/task-core/internal/domain"
	pb "github.com/DaDevFox/task-systems/task-core/proto/taskcore/v1"
)

var (
	serverAddr string
	client     pb.TaskServiceClient
	conn       *grpc.ClientConn
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "tasker",
		Short: "Task management CLI client",
		Long:  "A comprehensive task management CLI client with advanced features",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Connect to server
			var err error
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
	rootCmd.PersistentFlags().StringVar(&serverAddr, "server", "localhost:8080", "Server address")

	// Add commands
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

	cmd := &cobra.Command{
		Use:   "add <task-name>",
		Short: "Add a new task",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			name := args[0]

			req := &pb.AddTaskRequest{
				Name:        name,
				Description: description,
			}

			resp, err := client.AddTask(ctx, req)
			if err != nil {
				log.Fatalf("AddTask failed: %v", err)
			}

			fmt.Printf("Created task: %s (ID: %s)\n", resp.Task.Name, resp.Task.Id)
		},
	}

	cmd.Flags().StringVarP(&description, "description", "d", "", "Task description")

	return cmd
}

func newListCommand() *cobra.Command {
	var stage string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List tasks",
		Run: func(cmd *cobra.Command, args []string) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			stageEnum := parseStage(stage)
			req := &pb.ListTasksRequest{
				Stage: stageEnum,
			}

			resp, err := client.ListTasks(ctx, req)
			if err != nil {
				log.Fatalf("ListTasks failed: %v", err)
			}

			fmt.Printf("Tasks (%d total):\n", len(resp.Tasks))
			for _, task := range resp.Tasks {
				fmt.Printf("  %s: %s - %s [%s]\n", task.Id, task.Name, task.Description, task.Stage.String())
			}
		},
	}

	cmd.Flags().StringVarP(&stage, "stage", "s", "pending", "Task stage (pending, staging, active, completed)")

	return cmd
}

func newGetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <task-id>",
		Short: "Get task details",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			taskID := args[0]
			req := &pb.GetTaskRequest{Id: taskID}

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
		Use:   "start [task-id]",
		Short: "Start a task (with fuzzy picker if no ID provided)",
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
					log.Fatalf("Task selection failed: %v", err)
				}
			} else {
				taskID = args[0]
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
		Use:   "stop [task-id]",
		Short: "Stop a task (with fuzzy picker if no ID provided)",
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
					log.Fatalf("Task selection failed: %v", err)
				}
			} else {
				taskID = args[0]
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
		Use:   "complete [task-id]",
		Short: "Complete a task (with fuzzy picker if no ID provided)",
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
					log.Fatalf("Task selection failed: %v", err)
				}
			} else {
				taskID = args[0]
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

	cmd.AddCommand(&cobra.Command{
		Use:   "move <task-id>",
		Short: "Move task to staging",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			taskID := args[0]
			req := &pb.MoveToStagingRequest{SourceId: taskID}

			resp, err := client.MoveToStaging(ctx, req)
			if err != nil {
				log.Fatalf("MoveToStaging failed: %v", err)
			}

			fmt.Printf("Moved task to staging: %s\n", resp.Task.Name)
		},
	})

	return cmd
}

func newTagCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tag",
		Short: "Tag management commands",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "update <task-id>",
		Short: "Update task tags (shows current tags)",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			taskID := args[0]

			// Get current task to show existing tags
			getReq := &pb.GetTaskRequest{Id: taskID}
			getResp, err := client.GetTask(ctx, getReq)
			if err != nil {
				log.Fatalf("Failed to get task: %v", err)
			}

			fmt.Printf("Current tags for task '%s':\n", getResp.Task.Name)
			for key, value := range getResp.Task.Tags {
				fmt.Printf("  %s: %s\n", key, value.String())
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

	cmd.AddCommand(&cobra.Command{
		Use:   "get <user-id>",
		Short: "Get user details",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			userID := args[0]
			req := &pb.GetUserRequest{UserId: userID}

			resp, err := client.GetUser(ctx, req)
			if err != nil {
				log.Fatalf("GetUser failed: %v", err)
			}

			fmt.Printf("User: %s (%s)\n", resp.User.Name, resp.User.Email)
			fmt.Printf("ID: %s\n", resp.User.Id)
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

			if userID == "" {
				userID = "default-user"
			}

			req := &pb.GetTaskDAGRequest{UserId: userID}
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

	cmd.Flags().StringVarP(&userID, "user", "u", "", "User ID (defaults to 'default-user')")
	cmd.Flags().BoolVarP(&compact, "compact", "c", false, "Use compact rendering format")
	cmd.Flags().BoolVarP(&watch, "watch", "w", false, "Watch for changes (not implemented yet)")

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
	fmt.Printf("ID: %s\n", task.Id)
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
			fmt.Printf("  %s: %s\n", key, value.String())
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
		ID:          protoTask.Id,
		Name:        protoTask.Name,
		Description: protoTask.Description,
		UserID:      protoTask.UserId,
		Stage:       protoStageToDomain(protoTask.Stage),
		Status:      protoStatusToDomain(protoTask.Status),
		Location:    protoTask.Location,
		Inflows:     protoTask.Inflows,
		Outflows:    protoTask.Outflows,
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
	case pb.TagType_TAG_TYPE_TEXT:
		return domain.TagValue{
			Type:      domain.TagTypeText,
			TextValue: protoTag.GetTextValue(),
		}
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
	default:
		return domain.TagValue{
			Type:      domain.TagTypeText,
			TextValue: protoTag.GetTextValue(),
		}
	}
}
