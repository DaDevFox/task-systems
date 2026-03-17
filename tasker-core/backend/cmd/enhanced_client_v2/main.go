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

    root := &cobra.Command{Use: "tasker-v2", Short: "Objective-frame task client v2"}
    root.PersistentFlags().StringVar(&serverAddr, "server", "localhost:8080", "gRPC server address")

    root.AddCommand(newResolveTaskCommand(&serverAddr))
    root.AddCommand(newResolveUserCommand(&serverAddr))
    root.AddCommand(newCreateUserCommand(&serverAddr))

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

func newResolveUserCommand(serverAddr *string) *cobra.Command {
    return &cobra.Command{
        Use:  "resolve-user <user-input>",
        Args: cobra.ExactArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            return withClient(*serverAddr, func(client pb.TaskServiceClient) error {
                ctx, cancel := callCtx()
                defer cancel()

                resp, err := client.ResolveUserID(ctx, &pb.ResolveUserIDRequest{UserInput: args[0]})
                if err != nil {
                    return err
                }
                fmt.Printf("resolved_id=%s name=%s suggestions=%v\n", resp.ResolvedId, resp.ResolvedName, resp.Suggestions)
                return nil
            })
        },
    }
}

func newCreateUserCommand(serverAddr *string) *cobra.Command {
    return &cobra.Command{
        Use:  "create-user <email> <name>",
        Args: cobra.ExactArgs(2),
        RunE: func(cmd *cobra.Command, args []string) error {
            return withClient(*serverAddr, func(client pb.TaskServiceClient) error {
                ctx, cancel := callCtx()
                defer cancel()

                resp, err := client.CreateUser(ctx, &pb.CreateUserRequest{Email: args[0], Name: args[1], NotificationSettings: []*pb.NotificationSetting{{Type: pb.NotificationType_NOTIFICATION_ON_ASSIGN, Enabled: true}}})
                if err != nil {
                    return err
                }
                fmt.Printf("created user: %s (%s)\n", resp.User.Name, resp.User.Id)
                return nil
            })
        },
    }
}
