package clients

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	inventorypb "github.com/DaDevFox/task-systems/inventory-core/proto/inventory/v1"
	taskpb "github.com/DaDevFox/task-systems/task-core/proto/taskcore/v1"
)

// InventoryClient wraps the gRPC inventory service client
type InventoryClient struct {
	client inventorypb.InventoryServiceClient
	conn   *grpc.ClientConn
}

// NewInventoryClient creates a new inventory service client
func NewInventoryClient(address string) (*InventoryClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to inventory service: %w", err)
	}

	client := inventorypb.NewInventoryServiceClient(conn)

	return &InventoryClient{
		client: client,
		conn:   conn,
	}, nil
}

// Close closes the gRPC connection
func (c *InventoryClient) Close() error {
	return c.conn.Close()
}

// GetInventoryStatus retrieves current inventory status
func (c *InventoryClient) GetInventoryStatus(ctx context.Context) (*inventorypb.InventoryStatus, error) {
	req := &inventorypb.GetInventoryStatusRequest{}
	resp, err := c.client.GetInventoryStatus(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get inventory status: %w", err)
	}
	return resp.Status, nil
}

// UpdateInventoryLevel updates an item's inventory level
func (c *InventoryClient) UpdateInventoryLevel(ctx context.Context, itemID string, newLevel float64, reason string) (*inventorypb.InventoryItem, error) {
	req := &inventorypb.UpdateInventoryLevelRequest{
		ItemId:            itemID,
		NewLevel:          newLevel,
		Reason:            reason,
		RecordConsumption: true,
	}

	resp, err := c.client.UpdateInventoryLevel(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to update inventory level: %w", err)
	}

	return resp.Item, nil
}

// HealthCheck verifies the inventory service is responding
func (c *InventoryClient) HealthCheck(ctx context.Context) error {
	// Use a simple status call to verify connectivity
	_, err := c.GetInventoryStatus(ctx)
	if err != nil {
		return fmt.Errorf("inventory service health check failed: %w", err)
	}
	return nil
}

// TaskClient wraps the gRPC task service client
type TaskClient struct {
	client taskpb.TaskServiceClient
	conn   *grpc.ClientConn
}

// NewTaskClient creates a new task service client
func NewTaskClient(address string) (*TaskClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to task service: %w", err)
	}

	client := taskpb.NewTaskServiceClient(conn)

	return &TaskClient{
		client: client,
		conn:   conn,
	}, nil
}

// Close closes the gRPC connection
func (c *TaskClient) Close() error {
	return c.conn.Close()
}

// AddTask creates a new task
func (c *TaskClient) AddTask(ctx context.Context, name, description, userID string) (*taskpb.Task, error) {
	req := &taskpb.AddTaskRequest{
		Name:        name,
		Description: description,
		UserId:      userID,
	}

	resp, err := c.client.AddTask(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to add task: %w", err)
	}

	return resp.Task, nil
}

// GetTask retrieves a task by ID
func (c *TaskClient) GetTask(ctx context.Context, taskID string) (*taskpb.Task, error) {
	req := &taskpb.GetTaskRequest{
		Id: taskID,
	}

	resp, err := c.client.GetTask(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	return resp.Task, nil
}

// StartTask begins work on a task
func (c *TaskClient) StartTask(ctx context.Context, taskID string) (*taskpb.Task, error) {
	req := &taskpb.StartTaskRequest{
		Id: taskID,
	}

	resp, err := c.client.StartTask(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to start task: %w", err)
	}

	return resp.Task, nil
}

// CompleteTask marks a task as completed
func (c *TaskClient) CompleteTask(ctx context.Context, taskID string) (*taskpb.Task, error) {
	req := &taskpb.CompleteTaskRequest{
		Id: taskID,
	}

	resp, err := c.client.CompleteTask(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to complete task: %w", err)
	}

	return resp.Task, nil
}

// HealthCheck verifies the task service is responding
func (c *TaskClient) HealthCheck(ctx context.Context) error {
	// Use a simple list call to verify connectivity
	_, err := c.client.ListTasks(ctx, &taskpb.ListTasksRequest{
		UserId: "health-check",
		Stage:  taskpb.TaskStage_STAGE_PENDING, // Use a valid stage for health check
	})
	if err != nil {
		return fmt.Errorf("task service health check failed: %w", err)
	}
	return nil
}
