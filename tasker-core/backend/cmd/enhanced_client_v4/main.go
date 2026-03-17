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

	root := &cobra.Command{Use: "tasker-v4", Short: "Objective-frame client v4"}
	root.PersistentFlags().StringVar(&serverAddr, "server", "localhost:8080", "gRPC server address")

	newClient := func() (pb.TaskServiceClient, *grpc.ClientConn, error) {
		conn, err := grpc.Dial(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return nil, nil, err
		}
		return pb.NewTaskServiceClient(conn), conn, nil
	}

	root.AddCommand(&cobra.Command{
		Use:  "resolve-task <task-input>",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, conn, err := newClient()
			if err != nil {
				return err
			}
			defer conn.Close()

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			resp, err := client.ResolveTaskID(ctx, &pb.ResolveTaskIDRequest{TaskInput: args[0]})
			if err != nil {
				return err
			}
			fmt.Printf("resolved=%s min_prefix=%s suggestions=%v\n", resp.ResolvedId, resp.MinimumPrefix, resp.Suggestions)
			return nil
		},
	})

	root.AddCommand(&cobra.Command{
		Use:  "resolve-user <user-input>",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, conn, err := newClient()
			if err != nil {
				return err
			}
			defer conn.Close()

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			resp, err := client.ResolveUserID(ctx, &pb.ResolveUserIDRequest{UserInput: args[0]})
			if err != nil {
				return err
			}
			fmt.Printf("resolved_id=%s name=%s suggestions=%v\n", resp.ResolvedId, resp.ResolvedName, resp.Suggestions)
			return nil
		},
	})

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
