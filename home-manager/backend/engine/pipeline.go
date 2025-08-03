package engine

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"home-tasker/notify"

	pb "home-tasker/goproto/hometasker/v1"

	log "github.com/sirupsen/logrus"
)

func init() {
	log.Debug("hi")
}

// TODO: validate config: no task id duplicates;
// task id as "template/blueprint" is a bad idea
// because here, task id is used as primary key
// into current work; can't refer uniquely to step

// CompleteTask completes a task in the current pipeline work:
// uses the task id to find the task in the current pipeline work, marks it as complete,
// assigns review (exclusive) or TODO: next step in pipeline of work
func (e *Engine) CompleteTask(taskId string, userId string) error {
	state := e.State
	config := e.Config

	parts := strings.Split(taskId, ".")
	if len(parts) != 3 {
		err := errors.New("could not complete task: invalid full task ID format; expected format is 'taskSystemId.pipelineId.taskId'")
		log.WithError(err).WithFields(map[string]any{
			taskId: taskId,
			userId: userId,
		}).Error()
		return err
	}
	taskSystemId, pipelineId, taskIdOnly := parts[0], parts[1],parts[2]

	taskSystemIndex := slices.IndexFunc(config.TaskSystems, func(ts *pb.TaskSystem) bool { return ts.Id == taskSystemId })
	if taskSystemIndex == -1 {
		err := errors.New("could not complete task: task system not found")
		log.WithError(err).WithFields(map[string]any{
			"task_id": taskId,
			"task_system_id": taskSystemId,
		}).Error()
		return err
	}
	taskSystem := e.Config.TaskSystems[taskSystemIndex]

	// for _, pipelineRecord := range state.PipelineActivity {
	pipelineActivityIndex := slices.IndexFunc(state.PipelineActivity, func(r *pb.PipelineActivity) bool {
		return r.PipelineId == pipelineId})
	if pipelineActivityIndex == -1 {
		err := errors.New("Pipeline activity not found")
		log.WithError(err).WithFields(map[string]any{
			"task_id": taskId,
			"pipeline_id": pipelineId,
		}).Error()
	}
	pipelineActivity := state.PipelineActivity[pipelineActivityIndex]

	// for _, pipelineRecord := range state.PipelineActivity {
	pipelineIndex := slices.IndexFunc(taskSystem.Pipelines, func(r *pb.Pipeline) bool {
		return r.Id == pipelineId})
	if pipelineIndex == -1 {
		err := errors.New("Pipeline not found")
		log.WithError(err).WithFields(map[string]any{
			"task_id": taskId,
			"pipeline_id": pipelineId,
		}).Error()
	}
	pipeline := taskSystem.Pipelines[pipelineIndex]

	if pipelineActivity.TaskSystemId != taskSystemId {
		log.WithFields(map[string]any{
			"task_id": taskId,
			"task_system_id_from_task_id": taskSystemId,
			"task_system_id_from_pipeline_activity": pipelineActivity.TaskSystemId,
		}).Warn("pipeline in task system from task id claims descendence from another pipeline")
		pipelineActivity.TaskSystemId = taskSystemId
	}

	workItemIndex := slices.IndexFunc(pipelineActivity.PipelineWork,
		func(workItem *pb.PipelineWork) bool {
			return taskIdOnly == workItem.Task.TaskId
		})

	if workItemIndex == -1 {
		err := fmt.Errorf("task not found in pipeline work")
		log.
			WithFields(map[string]any{
				"full_task_id": taskId,
				"task_id": taskIdOnly,
			}).
			WithError(err).
			Errorf("task not found in pipeline work")
		return err
	}
	workItem := pipelineActivity.PipelineWork[workItemIndex]
	taskRecord := workItem.Task

	stepIdx := slices.IndexFunc(pipeline.Steps, func(step *pb.TaskStep) bool {
		return step.Task.Id == taskRecord.TaskId
	})
	if stepIdx == -1 {
		err := errors.New("step not found in pipeline")
		log.
			WithError(err).
			Errorf("step with task id %s not found in pipeline %v, even though it's pointed by pipeline activity record which states some pipeline can be found in task system %s", taskRecord.TaskId, pipelineActivity, taskSystem.Id)
		return err
	}
	step := pipeline.Steps[stepIdx]

	if step.RequireReview {
		if taskRecord.Status == pb.TASK_STATUS_ASSIGNED {
			taskRecord.Status = pb.TASK_STATUS_REVIEW_ASSIGNED
		} else {
			taskRecord.Status = pb.TASK_STATUS_COMPLETE
		}
	} else {
		taskRecord.Status = pb.TASK_STATUS_COMPLETE
	}

	taskRecord.CompletedAt = time.Now().Unix()

	if !slices.Contains(taskRecord.Assignee, userId) {
		completerConfigUserIdx := slices.IndexFunc(config.Users, func(u *pb.User) bool { return u.Id == userId })
		completerFullname := config.Users[completerConfigUserIdx].FirstName + " " + config.Users[completerConfigUserIdx].LastName
		// TODO: set up policy for users which are not the assignee ("helper"/task-takeovers) completing a task
		// TEMP: consider them valid, notify the former assignee and the completer
		err := notify.Send(config, userId, fmt.Sprintf("You completed '%s' for assignee(s) %v :P\n\nThey owe you cheesecake now", taskRecord.TaskId, taskRecord.Assignee))
		if err != nil {
			fmt.Errorf("Failed to send notification: %w", err)
		}
		for _, assignee := range taskRecord.Assignee {
			err = notify.Send(config, assignee, fmt.Sprintf("Your task '%s' has been completed by %s :/", taskRecord.TaskId, completerFullname))
			if err != nil {
				fmt.Errorf("Failed to send notification: %w", err)
			}
			// http.Error(w, "Task not assigned to user", http.StatusForbidden)
		}

		return nil
	}
	return fmt.Errorf("Task %s not found", taskId)
}

func (e *Engine) getAssignees(sys *pb.TaskSystem, step *pb.TaskStep) ([]string, error) {
	if sys.AssigneePool == nil || len(sys.AssigneePool) == 0 {
		err := errors.New("no assignee pool defined")
		log.WithError(err).WithFields(map[string]interface{}{
			"task_system_id": sys.Id,
			"step_id":        step.Task.Id,
		}).Error()
		return nil, err
	}

	assignees := []*pb.UserSlot{}
	assignment := step.AssignmentBehavior

	switch a := assignment.Assignment.(type) {
	case *pb.TaskAssignment_NewAssigneeOrSameAsPrevious:
		assignees = []*pb.UserSlot{PickUser(sys.AssigneePool)}
		break
	case *pb.TaskAssignment_NewAssignee:
		assignees = []*pb.UserSlot{PickUser(sys.AssigneePool)}
		break
	case *pb.TaskAssignment_GroupAssignees:
		// For group assignment, pick users until total_capacity is met or pool is exhausted
		capacity := a.GroupAssignees.TotalCapacity
		var group []*pb.UserSlot
		total := int32(0)

		for user := PickUser(sys.AssigneePool); user != nil; user = PickUserExcluding(sys.AssigneePool, group) {
			weight := int32(1)
			if user.Weight != nil {
				weight = *user.Weight
			}
			if total+weight > capacity {
				break
			}
			group = append(group, user)
			total += weight
		}
		assignees = []*pb.UserSlot{PickUser(sys.AssigneePool)}
		break
	default:
		assignees = []*pb.UserSlot{PickUser(sys.AssigneePool)}
		break
	}

	assigneeUserIds := []string{}
	for _, assignee := range assignees {
		assigneeUserIds = append(assigneeUserIds, assignee.Id)
	}

	return assigneeUserIds, nil
}

func (e *Engine) getReviewers(sys *pb.TaskSystem, step *pb.TaskStep) ([]string, error) {
	reviewers := []string{}
	if !step.RequireReview {
		return reviewers, nil
	}

	if sys.ReviewerPool == nil || len(sys.ReviewerPool) == 0 {
		err := fmt.Errorf("no reviewer pool defined")
		log.WithError(err).WithFields(map[string]interface{}{
			"task_system_id": sys.Id,
			"step":           step.Task.Id,
		}).Error()
		return nil, err
	}
	reviewers = []string{PickUser(sys.ReviewerPool).Id}

	return reviewers, nil
}

func maxStepIdx(pipelineActivity *pb.PipelineActivity) uint32 {
	maxIdx := uint32(0)
	for _, work := range pipelineActivity.PipelineWork {
		if work.StepIdx > maxIdx {
			maxIdx = work.StepIdx
		}
	}
	return maxIdx
}

// evaluatTaskCreation evaluates the conflict policy for validation of creation for the proposed work
// in the pipeline activity at its step index. Returns true if the work is accepted, false + error as status message if it is blocked.
// Executes the action of creation (adding to pipeline work) in addition to reporting status; DO NOT add to work based on the result of this function
func (e *Engine) evaluateTaskCreation(pipelineActivity *pb.PipelineActivity, pipeline *pb.Pipeline, proposedWork *pb.PipelineWork) (bool, error) {
	newActivity := len(pipelineActivity.PipelineWork) > 0
	step := pipeline.Steps[proposedWork.StepIdx]
	assignees := proposedWork.Task.Assignee
	lastExistingStep := maxStepIdx(pipelineActivity)

	// if we would surpass existing work and we don't possess a conflict policy allowing this, block the work
	if pipeline.ConflictPolicy.GetSurpass() == nil && lastExistingStep < proposedWork.StepIdx {
		for _, assignee := range assignees {
			notify.Send(e.Config, assignee, "You cannot be assigned to the task "+step.Task.DisplayName+" since you'd surpass existing work on step "+fmt.Sprint(lastExistingStep+1)+", which is forbidden by the conflict policy for this work pipeline")
		}
		return false, fmt.Errorf("conflict policy SURPASS is not set on + creation would surpass existing work on step at %d", lastExistingStep)
	}

	// easy case: create new activity if it doesn't exist
	if newActivity {
		pipelineActivity.PipelineWork = append(pipelineActivity.PipelineWork, proposedWork)
		log.WithFields(map[string]any{
			"pipeline_id":   pipeline.Id,
			"for_task_step": proposedWork.StepIdx,
		}).Debug("pipeline activity created")
		return true, nil
	}

	// dificult case: conflict policy evaluation if activity exists

	// let the new task through for aggregate and stack policies
	if pipeline.ConflictPolicy.GetAggregate() != nil || pipeline.ConflictPolicy.GetStack() != nil {
		pipelineActivity.PipelineWork = append(pipelineActivity.PipelineWork, proposedWork)
		for _, assignee := range assignees {
			notify.Send(e.Config, assignee, "You've been assigned a new task: "+step.Task.DisplayName)
		}
		return true, nil
	}

	// block the new work if block is enabled
	if pipeline.ConflictPolicy.GetBlock() != nil {
		// TODO: track attempted new entrants
		// for _, assignee := range assignees {
		// 	notify.Send(e.Config, assignee, "You cannot be assigned to this task: "+step.Task.DisplayName+" due to conflict policy")
		// }
		return false, fmt.Errorf("conflict policy BLOCK is set on %v", pipeline)
	}

	// NOTE: default policy is "replace"; uncomment below to change this behavior
	// if pipeline.ConflictPolicy.GetReplace() != nil {
	// TODO: teardown procedure for old piplene work

	// replace the old work with the new one
	// notify old assignees that their task is replaced
	for _, oldWork := range pipelineActivity.PipelineWork {
		for _, oldAssignee := range oldWork.Task.Assignee {
			notify.Send(e.Config, oldAssignee, fmt.Sprintf("Your task %s has been replaced: %v will perform %s instead", oldWork.Task.TaskId, assignees, step.Task.DisplayName))
		}
	}

	// and new ones of the assignment
	pipelineActivity.PipelineWork = []*pb.PipelineWork{proposedWork}
	for _, assignee := range assignees {
		notify.Send(e.Config, assignee, fmt.Sprintf("You've been assigned a new task: %s", step.Task.DisplayName))
	}
	// }

	return true, nil
}

// evaluateTaskAggregation evaluates whether to aggregate a task in the current pipeline work and returns true in such a case.
// Errors are returned indendent of aggregation attempt/result.
// Executes the action of aggregation (removing from pipeline work) in addition to reporting status; DO NOT remove from work based on the result of this function
func (e *Engine) evaluateTaskAggregation(pipelineActivity *pb.PipelineActivity, pipeline *pb.Pipeline, proposedWork *pb.PipelineWork) (bool, error) {
	if pipeline.ConflictPolicy.GetAggregate() == nil {
		return false, nil
	}

	arrivals := slices.Clone(pipelineActivity.PipelineWork)
	arrivals = append(arrivals, proposedWork)

	mtu := pipeline.ConflictPolicy.GetAggregate().MaxTransmissionUnit
	aggregatedAny := false
	removeSet := []*pb.PipelineWork{}
	currSeqNo := 0
	var buildingWork *pb.PipelineWork

	for _, work := range arrivals {
		if work.StepIdx == proposedWork.StepIdx {
			if buildingWork == nil {
				buildingWork = work
				continue
			}

			// aggregate same-step workloads
			if buildingWork.Points+work.Points < mtu {
				buildingWork.Points += work.Points
				aggregatedAny = true

				var assigneesJoined map[string]bool
				// TODO: refactor out into set operations file
				for _, assignee := range buildingWork.Task.Assignee {
					assigneesJoined[assignee] = true
				}

				for _, assignee := range work.Task.Assignee {
					assigneesJoined[assignee] = true // TODO: handle multiple assignees
				}

				buildingWork.Task.Assignee = slices.Collect(maps.Keys(assigneesJoined))
				removeSet = append(removeSet, work)
				continue
			}

			// (reaching this point means same step, can't aggregate => start new workload chunk)
			// TODO: assess possibility for splitting (here, if buildingWork.Points > mtu)
			currSeqNo++
			buildingWork = work
			buildingWork.ArrivalOnStepSeqno = uint32(currSeqNo)
		}
	}

	slices.DeleteFunc(pipelineActivity.PipelineWork, func(w *pb.PipelineWork) bool {
		return slices.Contains(removeSet, w)
	})

	return aggregatedAny, nil
}

// AssignStep assigns a step to a user in the current task system:
// uses conflict policy to determine how/whether to assign the step
// if a step is already active in the pipeline,
func (e *Engine) AssignStep(step *pb.TaskStep, sys *pb.TaskSystem, points int32) error {
	piplineIdx := slices.IndexFunc(sys.Pipelines, func(pipeline *pb.Pipeline) bool {
		return slices.ContainsFunc(pipeline.Steps, func(s *pb.TaskStep) bool { return s.Task.Id == step.Task.Id })
	})
	if piplineIdx == -1 {
		return fmt.Errorf("task system with id %s does not contain a pipeline which contains step %v", sys.Id, step)
	}
	pipeline := sys.Pipelines[piplineIdx]

	stepIdx := slices.IndexFunc(pipeline.Steps, func(s *pb.TaskStep) bool { return s.Task.Id == step.Task.Id })

	assignees, err := e.getAssignees(sys, step)
	if err != nil {
		log.WithError(err).WithFields(map[string]any{
			"task_system_id": sys.Id,
			"step_id":        step.Task.Id,
		})
		return fmt.Errorf("couldn't assign step: %w", err)
	}

	reviewers, err := e.getReviewers(sys, step)
	if err != nil {
		log.WithError(err).WithFields(map[string]any{
			"task_system_id": sys.Id,
			"step_id":        step.Task.Id,
		})
		return fmt.Errorf("couldn't assign step: %w", err)
	}

	now := time.Now().Unix()
	record := &pb.TaskRecord{
		TaskId:     step.Task.Id,
		Assignee:   assignees,
		AssignedAt: now,
		Status:     pb.TASK_STATUS_ASSIGNED,
		Reviewer:   reviewers,
	}

	// Create new Pipeline Activity (pipeline meta) if it doesn't exist (sub-task system)
	var pipelineActivity *pb.PipelineActivity
	if !slices.ContainsFunc(e.State.PipelineActivity, func(a *pb.PipelineActivity) bool { return a.PipelineId == pipeline.Id && a.TaskSystemId == sys.Id }) {
		pipelineActivity = &pb.PipelineActivity{
			PipelineId:   pipeline.Id,
			TaskSystemId: sys.Id,
		}
		e.State.PipelineActivity = append(e.State.PipelineActivity, pipelineActivity)
	}
	pipelineActivity = e.State.PipelineActivity[slices.IndexFunc(e.State.PipelineActivity, func(a *pb.PipelineActivity) bool { return a.PipelineId == pipeline.Id && a.TaskSystemId == sys.Id })]

	// Create new Pipeline Work (task step meta) for insertion (sub-pipeline)
	work := new(pb.PipelineWork)
	work.Points = points
	work.StepIdx = uint32(stepIdx)
	work.Task = record

	added, err := e.evaluateTaskCreation(pipelineActivity, pipeline, work)
	if !added {
		log.WithError(err).Error("unable to assign task step")
		return fmt.Errorf("unable to assign task step: %w", err)
	}

	count := uint32(0)
	for _, w := range pipelineActivity.PipelineWork {
		if w.StepIdx == work.StepIdx {
			count++
		}
	}
	work.ArrivalOnStepSeqno = count + 1

	aggregated, err := e.evaluateTaskAggregation(pipelineActivity, pipeline, work)
	if err != nil {
		log.WithError(err).Error("unable to assign task step")
		return fmt.Errorf("unable to assign task step: %w", err)
	}
	if aggregated {
		log.WithField("task", record).Debugf("Aggregated work after creation")
	}

	// assured by ContainsFunc above
	return nil
}
