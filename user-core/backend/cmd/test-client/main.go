package main

import (
	"context"
	"fmt"
	"log"
	"time"

	pb "github.com/DaDevFox/task-systems/user-core/backend/pkg/proto/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	// Connect to the server
	conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	// Create client
	client := pb.NewUserServiceClient(conn)

	// Test CreateUser
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	createReq := &pb.CreateUserRequest{
		Email:     "test@example.com",
		Name:      "TestUser",
		FirstName: "Test",
		LastName:  "User",
		Role:      pb.UserRole_USER_ROLE_USER,
	}

	fmt.Println("Creating user...")
	createResp, err := client.CreateUser(ctx, createReq)
	if err != nil {
		log.Fatalf("CreateUser failed: %v", err)
	}

	fmt.Printf("User created: %+v\n", createResp.User)

	// Test GetUser
	fmt.Println("Getting user by ID...")
	getReq := &pb.GetUserRequest{
		Identifier: &pb.GetUserRequest_UserId{UserId: createResp.User.Id},
	}

	getResp, err := client.GetUser(ctx, getReq)
	if err != nil {
		log.Fatalf("GetUser failed: %v", err)
	}

	fmt.Printf("User retrieved: %+v\n", getResp.User)

	// Test ListUsers
	fmt.Println("Listing users...")
	listReq := &pb.ListUsersRequest{
		PageSize: 10,
	}

	listResp, err := client.ListUsers(ctx, listReq)
	if err != nil {
		log.Fatalf("ListUsers failed: %v", err)
	}

	fmt.Printf("Found %d users (total: %d)\n", len(listResp.Users), listResp.TotalCount)

	fmt.Println("All tests passed! ✅")
}
