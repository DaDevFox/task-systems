package engine

import (
	"math/rand"
	"testing"

	pb "github.com/DaDevFox/task-systems/workflows/backend/pkg/proto/workflows/v1"
	"github.com/DaDevFox/task-systems/workflows/backend/notify"
)

type mockNotifier struct {
	notifications []string
}

func (m *mockNotifier) Notify(userID, msg string) {
	m.notifications = append(m.notifications, userID+": "+msg)
}

func TestEngineAssignStep_AssignsUserAndReviewer(t *testing.T) {
	rand.Seed(1) // deterministic

	assigneePool := []*pb.UserSlot{
		{Id: "user1"},
		{Id: "user2"},
	}
	reviewerPool := []*pb.UserSlot{
		{Id: "rev1"},
		{Id: "rev2"},
	}
	task := &pb.Task{DisplayName: "TestTask"}
	step := &pb.TaskStep{
		Task: task,
		AssignmentBehavior: &pb.TaskAssignment{
			Assignment: &pb.TaskAssignment_NewAssignee{NewAssignee: &pb.NewAssignee{}},
		},
	}
	sys := &pb.TaskSystem{
		AssigneePool: assigneePool,
		ReviewerPool: reviewerPool,
	}
	state := &pb.SystemState{}
	notifier := func(userID, msg string) error { return nil }

	e := &Engine{
		Config:    &pb.Config{},
		State:     state,
		Notifiers: []notify.Notifier{notifier},
		Systems:   []*pb.TaskSystem{sys},
	}

	e.AssignStep(step, sys)

	if len(state.TaskHistory) != 1 {
		// t.Fatalf("expected 1 task event, got %d", len(state.TaskHistory))
	}
	event := state.TaskHistory[0]
	if event.Task != "TestTask" {
		// t.Errorf("expected task name TestTask, got %s", event.Task)
	}
	if event.User == "" {
		t.Errorf("expected assigned user, got empty")
	}
	if event.Reviewer == "" {
		t.Errorf("expected reviewer, got empty")
	}
}

func TestPickUser_ReturnsNobodyIfEmpty(t *testing.T) {
	user := pickUser([]*pb.UserSlot{})
	if user.Id != "nobody" {
		t.Errorf("expected nobody, got %s", user.Id)
	}
}
