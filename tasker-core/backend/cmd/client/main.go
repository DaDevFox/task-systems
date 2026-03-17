package main

import (
    "context"
    "fmt"
    "os"
    "time"

    "github.com/spf13/cobra"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"

    pb "github.com/DaDevFox/task-systems/tasker-core/backend/pkg/proto/taskcore/v1"
)

func main() {
    var serverAddr string

    root := &cobra.Command{Use: "tasker-client", Short: "Tasker objective-frame client"}
    root.PersistentFlags().StringVar(&serverAddr, "server", "localhost:8080", "gRPC server address")

    root.AddCommand(newAddCommand(&serverAddr))
    root.AddCommand(newListCommand(&serverAddr))
    root.AddCommand(newGetCommand(&serverAddr))
    root.AddCommand(newCompleteCommand(&serverAddr))
    root.AddCommand(newResolveTaskCommand(&serverAddr))

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

                resp, err := client.AddTask(ctx, &pb.AddTaskRequest{Task: &pb.Task{
                    Name:      args[0],
                    DomainId:  args[1],
                    Assignees: assignees,
                    Status:    pb.TaskStatus_TASK_STATUS_TODO,
                }})
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
                    fmt.Printf("%s | %s | %s | %s | assignees=%v\n", task.Id, task.DomainId, task.Name, task.Status.String(), task.Assignees)
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

func newResolveTaskCommand(serverAddr *string) *cobra.Command {
    return &cobra.Command{
        Use:  "resolve-task <task-input>",
        Args: cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            return withClient(*serverAddr, func(client pb.TaskServiceClient) error {
                ctx, cancel := callCtx()
                defer cancel()

                resp, err := client.ResolveTaskID(ctx, &pb.ResolveTaskIDRequest{TaskInput: args[0]})
                if err != nil {
                    return err
                }
                fmt.Printf("resolved=%s min_prefix=%s suggestions=%v\n", resp.ResolvedId, resp.MinimumPrefix, resp.Suggestions)
                return nil
            })
        },
    }
}
