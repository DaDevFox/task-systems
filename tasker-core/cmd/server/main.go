package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/DaDevFox/task-systems/task-core/internal/calendar"
	"github.com/DaDevFox/task-systems/task-core/internal/email"
	"github.com/DaDevFox/task-systems/task-core/internal/events"
	grpcserver "github.com/DaDevFox/task-systems/task-core/internal/grpc"
	"github.com/DaDevFox/task-systems/task-core/internal/repository"
	"github.com/DaDevFox/task-systems/task-core/internal/service"
	pb "github.com/DaDevFox/task-systems/task-core/proto/taskcore/v1"
)

func main() {
	var (
		port                     = flag.Int("port", 8080, "The server port")
		maxInboxSize             = flag.Int("max-inbox-size", 5, "Maximum number of tasks allowed in inbox")
		enableCalendarSync       = flag.Bool("enable-calendar", false, "Enable Google Calendar integration")
		enableEmailNotifications = flag.Bool("enable-email", false, "Enable email notifications")
		calendarClientID         = flag.String("calendar-client-id", "", "Google Calendar OAuth2 client ID")
		calendarClientSecret     = flag.String("calendar-client-secret", "", "Google Calendar OAuth2 client secret")
		calendarRedirectURL      = flag.String("calendar-redirect-url", "http://localhost:8080/auth/callback", "Calendar OAuth2 redirect URL")
		smtpHost                 = flag.String("smtp-host", "smtp.gmail.com", "SMTP server host")
		smtpPort                 = flag.String("smtp-port", "587", "SMTP server port")
		smtpUsername             = flag.String("smtp-username", "", "SMTP username")
		smtpPassword             = flag.String("smtp-password", "", "SMTP password")
		fromEmail                = flag.String("from-email", "", "From email address")
		reminderInterval         = flag.Duration("reminder-interval", 1*time.Hour, "Interval for checking due reminders")
	)
	flag.Parse()

	// Create repositories
	taskRepo := repository.NewInMemoryTaskRepository()
	userRepo := repository.NewInMemoryUserRepository()

	// Initialize structured logger
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	// Initialize event bus
	eventBus := events.NewPubSub(logger)

	// Initialize services (optional)
	var calendarService *calendar.CalendarService
	var emailService *email.EmailService

	if *enableCalendarSync {
		if *calendarClientID == "" || *calendarClientSecret == "" {
			log.Fatal("Calendar client ID and secret are required when calendar sync is enabled")
		}
		calendarService = calendar.NewCalendarService(*calendarClientID, *calendarClientSecret, *calendarRedirectURL)
		log.Println("Calendar sync enabled")
	}

	if *enableEmailNotifications {
		if *smtpUsername == "" || *smtpPassword == "" || *fromEmail == "" {
			log.Fatal("SMTP credentials and from email are required when email notifications are enabled")
		}
		emailService = email.NewEmailService(*smtpHost, *smtpPort, *smtpUsername, *smtpPassword, *fromEmail)
		log.Println("Email notifications enabled")
	}

	// Create unified task service with all features
	taskService := service.NewTaskService(
		taskRepo,
		*maxInboxSize,
		userRepo,
		calendarService,
		emailService,
		logger,
		eventBus,
	)

	// Create gRPC server
	taskServer := grpcserver.NewTaskServer(taskService)

	// Set up gRPC server
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterTaskServiceServer(s, taskServer)

	// Enable reflection for easier debugging
	reflection.Register(s)

	// Start reminder check routine if email service is enabled
	if emailService != nil {
		go func() {
			ticker := time.NewTicker(*reminderInterval)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					if err := taskService.CheckDueReminders(context.Background()); err != nil {
						log.Printf("Error checking due reminders: %v", err)
					}
				}
			}
		}()
		log.Printf("Started reminder check routine (interval: %v)", *reminderInterval)
	}

	// Set up graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Println("Shutting down server...")
		s.GracefulStop()
		cancel()
	}()

	log.Printf("Task service starting on port %d", *port)
	log.Printf("Max inbox size: %d", *maxInboxSize)

	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}

	<-ctx.Done()
	log.Println("Server stopped")
}
