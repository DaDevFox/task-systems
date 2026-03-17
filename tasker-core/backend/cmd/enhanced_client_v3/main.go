package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/DaDevFox/task-systems/tasker-core/backend/internal/storage"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/DaDevFox/task-systems/tasker-core/backend/pkg/proto/taskcore/v1"
)

func main() {
	var serverAddr string

	root := &cobra.Command{Use: "tasker", Short: "Task client"}
	root.PersistentFlags().StringVar(&serverAddr, "server", "localhost:8080", "gRPC server address")
	root.AddCommand(newAddCommand(&serverAddr))
	root.AddCommand(newListCommand(&serverAddr))
	root.AddCommand(newGetCommand(&serverAddr))
	root.AddCommand(newCompleteCommand(&serverAddr))
	root.AddCommand(newUserCommand(&serverAddr))
	root.AddCommand(newDemoCompletionFlowCommand(&serverAddr))
	root.AddCommand(newPackAttachmentCommand())

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func dial(serverAddr string) (pb.TaskServiceClient, *grpc.ClientConn, error) {
	conn, err := grpc.Dial(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}
	return pb.NewTaskServiceClient(conn), conn, nil
}

func withClient(serverAddr string, fn func(client pb.TaskServiceClient) error) error {
	client, conn, err := dial(serverAddr)
	if err != nil {
		return err
	}
	defer conn.Close()
	return fn(client)
}

func callCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 10*time.Second)
}

func newAddCommand(serverAddr *string) *cobra.Command {
	return &cobra.Command{
		Use:  "add <name> <domain-id> [assignee]",
		Args: cobra.RangeArgs(2, 3),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withClient(*serverAddr, func(client pb.TaskServiceClient) error {
				assignees := []string{}
				if len(args) == 3 {
					assignees = append(assignees, args[2])
				}
				ctx, cancel := callCtx()
				defer cancel()
				resp, err := client.AddTask(ctx, &pb.AddTaskRequest{Task: &pb.Task{Name: args[0], DomainId: args[1], Assignees: assignees, Status: pb.TaskStatus_TASK_STATUS_TODO}})
				if err != nil {
					return err
				}
				fmt.Printf("created task: %s (%s)\n", resp.Task.Name, resp.Task.Id)
				return nil
			})
		},
	}
}

func newListCommand(serverAddr *string) *cobra.Command {
	return &cobra.Command{
		Use:  "list [domain-id]",
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withClient(*serverAddr, func(client pb.TaskServiceClient) error {
				domainID := ""
				if len(args) == 1 {
					domainID = args[0]
				}
				ctx, cancel := callCtx()
				defer cancel()
				resp, err := client.ListTasks(ctx, &pb.ListTasksRequest{DomainId: domainID})
				if err != nil {
					return err
				}
				for _, task := range resp.Tasks {
					fmt.Printf("%s | %s | %s | assignees=%v\n", task.Id, task.DomainId, task.Name, task.Assignees)
				}
				return nil
			})
		},
	}
}

func newGetCommand(serverAddr *string) *cobra.Command {
	return &cobra.Command{
		Use:  "get <task-id>",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withClient(*serverAddr, func(client pb.TaskServiceClient) error {
				ctx, cancel := callCtx()
				defer cancel()
				resp, err := client.GetTask(ctx, &pb.GetTaskRequest{TaskId: args[0]})
				if err != nil {
					return err
				}
				fmt.Printf("task=%s domain=%s status=%s assignees=%v\n", resp.Task.Name, resp.Task.DomainId, resp.Task.Status.String(), resp.Task.Assignees)
				return nil
			})
		},
	}
}

func newCompleteCommand(serverAddr *string) *cobra.Command {
	return &cobra.Command{
		Use:  "complete <task-id>",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withClient(*serverAddr, func(client pb.TaskServiceClient) error {
				ctx, cancel := callCtx()
				defer cancel()
				_, err := client.CompleteTask(ctx, &pb.CompleteTaskRequest{TaskId: args[0]})
				if err != nil {
					return err
				}
				fmt.Printf("completed %s\n", args[0])
				return nil
			})
		},
	}
}

func newUserCommand(serverAddr *string) *cobra.Command {
	userCmd := &cobra.Command{Use: "user", Short: "User commands"}
	userCmd.AddCommand(&cobra.Command{
		Use:  "create <email> <name>",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withClient(*serverAddr, func(client pb.TaskServiceClient) error {
				ctx, cancel := callCtx()
				defer cancel()
				resp, err := client.CreateUser(ctx, &pb.CreateUserRequest{Email: args[0], Name: args[1], NotificationSettings: []*pb.NotificationSetting{{Type: pb.NotificationType_NOTIFICATION_ON_ASSIGN, Enabled: true}}})
				if err != nil {
					return err
				}
				fmt.Printf("created user %s (%s)\n", resp.User.Name, resp.User.Id)
				return nil
			})
		},
	})
	userCmd.AddCommand(&cobra.Command{
		Use:  "get <id-or-email>",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withClient(*serverAddr, func(client pb.TaskServiceClient) error {
				ctx, cancel := callCtx()
				defer cancel()
				resp, err := client.GetUser(ctx, &pb.GetUserRequest{Identifier: &pb.GetUserRequest_Unknown{Unknown: args[0]}})
				if err != nil {
					return err
				}
				fmt.Printf("user %s <%s> settings=%d\n", resp.User.Id, resp.User.Email, len(resp.User.NotificationSettings))
				return nil
			})
		},
	})
	userCmd.AddCommand(&cobra.Command{
		Use:  "setting-set <id-or-email> <key> <value>",
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withClient(*serverAddr, func(client pb.TaskServiceClient) error {
				ctx, cancel := callCtx()
				defer cancel()

				getResp, err := client.GetUser(ctx, &pb.GetUserRequest{Identifier: &pb.GetUserRequest_Unknown{Unknown: args[0]}})
				if err != nil {
					return err
				}

				if getResp.User.SystemSettings == nil {
					getResp.User.SystemSettings = map[string]string{}
				}
				getResp.User.SystemSettings[args[1]] = args[2]

				_, err = client.UpdateUser(ctx, &pb.UpdateUserRequest{User: getResp.User})
				if err != nil {
					return err
				}

				fmt.Printf("updated setting %s for user %s\n", args[1], getResp.User.Id)
				return nil
			})
		},
	})
	return userCmd
}

func newDemoCompletionFlowCommand(serverAddr *string) *cobra.Command {
	return &cobra.Command{
		Use:  "demo-completion-flow <name> <domain-id> <assignee> <api-url> <regex> <file-path>",
		Args: cobra.ExactArgs(6),
		RunE: func(cmd *cobra.Command, args []string) error {
			return withClient(*serverAddr, func(client pb.TaskServiceClient) error {
				ctx, cancel := callCtx()
				defer cancel()

				addResp, err := client.AddTask(ctx, &pb.AddTaskRequest{Task: &pb.Task{
					Name:      args[0],
					DomainId:  args[1],
					Assignees: []string{args[2]},
					Status:    pb.TaskStatus_TASK_STATUS_TODO,
				}})
				if err != nil {
					return err
				}

				task := addResp.Task
				task.Resources = []*pb.Resource{{
					Resource: &pb.Resource_ApiResource{ApiResource: &pb.APIResource{Url: args[3], ResponseValidRegex: args[4]}},
				}, {
					Resource: &pb.Resource_FileAttachmentResource{FileAttachmentResource: &pb.FileAttachmentResource{FileLocation: args[5]}},
				}}
				task.Results = []*pb.TaskResult{{
					RequirementId: "proof.file",
					Complete:      true,
					Result: &pb.Result{Result: &pb.Result_FileAttachmentResult{FileAttachmentResult: &pb.FileAttachmentResult{FileLocation: args[5]}}},
				}}

				_, err = client.UpdateTask(ctx, &pb.UpdateTaskRequest{Task: task})
				if err != nil {
					return err
				}

				_, err = client.CompleteTask(ctx, &pb.CompleteTaskRequest{TaskId: task.Id})
				if err != nil {
					return err
				}

				fmt.Printf("demo flow completed task %s\n", task.Id)
				return nil
			})
		},
	}
}

func newPackAttachmentCommand() *cobra.Command {
	return &cobra.Command{
		Use:  "pack-attachment <source-file> [output-dir]",
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			outputDir := filepath.Join(".", "attachments")
			if len(args) == 2 {
				outputDir = args[1]
			}

			store := storage.NewAttachmentStore(outputDir)
			storedPath, err := store.StoreFile(args[0])
			if err != nil {
				return err
			}

			fmt.Printf("packed attachment path: %s\n", storedPath)
			return nil
		},
	}
}
