package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/DaDevFox/task-systems/task-core/proto/taskcore/v1"
)

func main() {
	var (
		serverAddr = flag.String("server", "localhost:8080", "Server address")
		command    = flag.String("cmd", "list", "Command to execute (add, list, start, stop, complete)")
		taskName   = flag.String("name", "", "Task name")
		taskDesc   = flag.String("desc", "", "Task description")
		taskID     = flag.String("id", "", "Task ID")
		stage      = flag.String("stage", "pending", "Task stage for listing")
	)
	flag.Parse()

	// Connect to server
	conn, err := grpc.Dial(*serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewTaskServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	switch *command {
	case "add":
		addTask(ctx, client, *taskName, *taskDesc)
	case "list":
		listTasks(ctx, client, *stage)
	case "get":
		getTask(ctx, client, *taskID)
	case "start":
		startTask(ctx, client, *taskID)
	case "stop":
		stopTask(ctx, client, *taskID)
	case "complete":
		completeTask(ctx, client, *taskID)
	case "move":
		moveToStaging(ctx, client, *taskID)
	default:
		fmt.Printf("Unknown command: %s\n", *command)
		fmt.Println("Available commands: add, list, get, start, stop, complete, move")
	}
}

func addTask(ctx context.Context, client pb.TaskServiceClient, name, desc string) {
	if name == "" {
		log.Fatal("Task name is required for add command")
	}

	req := &pb.AddTaskRequest{
		Name:        name,
		Description: desc,
	}

	resp, err := client.AddTask(ctx, req)
	if err != nil {
		log.Fatalf("AddTask failed: %v", err)
	}

	fmt.Printf("Created task: %s (ID: %s)\n", resp.Task.Name, resp.Task.Id)
}

func listTasks(ctx context.Context, client pb.TaskServiceClient, stageStr string) {
	stage := parseStage(stageStr)

	req := &pb.ListTasksRequest{
		Stage: stage,
	}

	resp, err := client.ListTasks(ctx, req)
	if err != nil {
		log.Fatalf("ListTasks failed: %v", err)
	}

	fmt.Printf("Tasks in %s stage (%d total):\n", stageStr, len(resp.Tasks))
	for _, task := range resp.Tasks {
		fmt.Printf("  %s: %s - %s\n", task.Id, task.Name, task.Description)
	}
}

func getTask(ctx context.Context, client pb.TaskServiceClient, id string) {
	if id == "" {
		log.Fatal("Task ID is required for get command")
	}

	req := &pb.GetTaskRequest{
		Id: id,
	}

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
	fmt.Printf("  Location: %v\n", task.Location)
	fmt.Printf("  Points: %v\n", task.Points)
	fmt.Printf("  Inflows: %v\n", task.Inflows)
	fmt.Printf("  Outflows: %v\n", task.Outflows)
	fmt.Printf("  Tags: %v\n", task.Tags)
}

func startTask(ctx context.Context, client pb.TaskServiceClient, id string) {
	if id == "" {
		log.Fatal("Task ID is required for start command")
	}

	req := &pb.StartTaskRequest{
		Id: id,
	}

	resp, err := client.StartTask(ctx, req)
	if err != nil {
		log.Fatalf("StartTask failed: %v", err)
	}

	fmt.Printf("Started task: %s\n", resp.Task.Name)
}

func stopTask(ctx context.Context, client pb.TaskServiceClient, id string) {
	if id == "" {
		log.Fatal("Task ID is required for stop command")
	}

	req := &pb.StopTaskRequest{
		Id: id,
		// For demo purposes, we'll stop without specifying completed points
		PointsCompleted: []*pb.Point{},
	}

	resp, err := client.StopTask(ctx, req)
	if err != nil {
		log.Fatalf("StopTask failed: %v", err)
	}

	fmt.Printf("Stopped task: %s (Completed: %t)\n", resp.Task.Name, resp.Completed)
}

func completeTask(ctx context.Context, client pb.TaskServiceClient, id string) {
	if id == "" {
		log.Fatal("Task ID is required for complete command")
	}

	req := &pb.CompleteTaskRequest{
		Id: id,
	}

	resp, err := client.CompleteTask(ctx, req)
	if err != nil {
		log.Fatalf("CompleteTask failed: %v", err)
	}

	fmt.Printf("Completed task: %s\n", resp.Task.Name)
}

func moveToStaging(ctx context.Context, client pb.TaskServiceClient, id string) {
	if id == "" {
		log.Fatal("Task ID is required for move command")
	}

	req := &pb.MoveToStagingRequest{
		SourceId: id,
		Destination: &pb.MoveToStagingRequest_NewLocation{
			NewLocation: &pb.MoveToStagingRequest_NewLocationList{
				NewLocation: []string{"project", "backend"},
			},
		},
		Points: []*pb.Point{
			{Title: "implementation", Value: 8},
			{Title: "testing", Value: 3},
		},
	}

	resp, err := client.MoveToStaging(ctx, req)
	if err != nil {
		log.Fatalf("MoveToStaging failed: %v", err)
	}

	fmt.Printf("Moved task to staging: %s\n", resp.Task.Name)
}

func parseStage(stageStr string) pb.TaskStage {
	switch stageStr {
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
