package engine

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"math/rand"
	"slices"
	"time"

	pb "github.com/DaDevFox/task-systems/workflows/backend/pkg/proto/workflows/v1"
	"github.com/DaDevFox/task-systems/workflows/backend/notify"
)

var Singleton *Engine

type Engine struct {
	Config    *pb.Config
	State     *pb.SystemState
	Notifiers []notify.Notifier
	Systems   []*pb.TaskSystem
}

func NewEngine(cfg *pb.Config, systems []*pb.TaskSystem, state *pb.SystemState, n []notify.Notifier) *Engine {
	return &Engine{
		Config: cfg, Systems: systems, State: state, Notifiers: n,
	}
}

func Start(cfg *pb.Config, systems []*pb.TaskSystem, state *pb.SystemState, notifiers []notify.Notifier) {
	if Singleton == nil {
		Singleton = NewEngine(cfg, systems, state, notifiers)
	}

	e := Singleton

	// start passive triggers (pile accumulation)
	for _, trig := range e.Config.PassivePipeline {
		for _, cond := range trig.Trigger {
			go e.watchPassiveTriggerCondition(cond, trig.Result)
		}
	}

	// start active systems
	for _, sys := range e.Systems {
		for _, trig := range sys.Pipelines {
			for _, cond := range trig.Condition {
				go e.watchTriggerCondition(sys, cond, trig)
			}
		}
	}
}

func (e *Engine) watchPassiveTriggerCondition(trigger *pb.PassiveTrigger, result *pb.Result) {
	if trigger.GetWeeklySchedule() != nil {
		sched := trigger.GetWeeklySchedule()

		for {
			now := time.Now()
			next := nextScheduledTime(sched, now)
			log.WithFields(map[string]interface{}{
				"weekly_trigger": trigger,
				"nextTime":       next.String(),
			}).Debugf("Passive trigger scheduled")
			time.Sleep(time.Until(next))

			// Add to pile
			evalResult(result, e.State)
			log.WithFields(map[string]interface{}{
				"weekly_trigger": trigger,
				"result":         result.String(),
			}).Debugf("Passive trigger executed")
		}
	} else if trigger.GetInterval() != nil {
		ticker := time.NewTicker(time.Duration(trigger.GetInterval().GetInterval().Seconds) * time.Second)

		for {
			log.WithFields(map[string]interface{}{
				"trigger":        trigger,
				"time_to_repeat": trigger.GetInterval().Interval.AsDuration().String(),
			}).Debugf("Passive trigger scheduled")
			<-ticker.C

			evalResult(result, e.State)
			log.WithFields(map[string]interface{}{
				"trigger": trigger,
				"result":  result.String(),
			}).Debugf("Passive trigger executed")
		}
	}
}

func nextScheduledTime(sched *pb.TriggerWeeklySchedule, now time.Time) time.Time {
	targetDay := int(sched.Day)
	daysUntil := (targetDay - int(now.Weekday()) + 7) % 7
	next := time.Date(
		now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location(),
	).AddDate(0, 0, daysUntil).Add(time.Duration(sched.SecondsSinceMidnight) * time.Second)
	if next.Before(now) {
		next = next.Add(7 * 24 * time.Hour)
	}
	return next
}

func (e *Engine) calculateValue(pipeline *pb.Pipeline) (int32, error) {
	total := int32(0)

	values, err := e.calculateValuePerPile(pipeline)
	if err != nil {
		return 0, fmt.Errorf("failed to calculate total pipeline value: %w", err)
	}

	for _, value := range values {
		total += value
	}

	return total, nil
}

// gives a prediction (based on current target pile value) of the difference to the pile value after the pipeline is executed
func (e *Engine) calculateValuePerPile(pipeline *pb.Pipeline) (map[string]int32, error) {
	total := map[string]int32{}

	for _, step := range pipeline.Steps {
		if step.Task.Result == nil || step.Task.Result.GetModifyPile() == nil {
			continue // no result to evaluate
		}
		pile, err := FindPileFatal(step.Task.Result.GetModifyPile().PileId, e.State.Piles)
		if err != nil {
			return map[string]int32{}, err // pile not found, cannot evaluate
		}
		if _, ok := total[pile.Id]; !ok {
			total[pile.Id] = pile.Value // initialize with current pile value
		}

		switch step.Task.Result.GetModifyPile().Operand {
		case pb.Operand_OPERAND_ADD:
			total[pile.Id] += step.Task.Result.GetModifyPile().Value
			break
		case pb.Operand_OPERAND_SUBTRACT:
			total[pile.Id] -= step.Task.Result.GetModifyPile().Value
			break
		case pb.Operand_OPERAND_SET:
			total[pile.Id] = step.Task.Result.GetModifyPile().Value
			break
		case pb.Operand_OPERAND_MULTIPLY:
			total[pile.Id] *= step.Task.Result.GetModifyPile().Value
			break
		case pb.Operand_OPERAND_DIVIDE:
			total[pile.Id] /= step.Task.Result.GetModifyPile().Value
			break
		}
	}

	return total, nil
}

func (e *Engine) checkPileCond(pileid string, trigger *pb.TriggerPileThreshold) (bool, error) {
	pile, err := FindPileFatal(pileid, e.State.Piles)
	if err != nil {
		return false, err
	}
	switch trigger.Comparison {
	case pb.COMPARISON_GREATER_THAN:
		if pile.Value > trigger.Threshold {
			return true, nil
		}
		break

	case pb.COMPARISON_LESS_THAN:
		if pile.Value < trigger.Threshold {
			return true, nil
		}
		break
	case pb.COMPARISON_EQUALS:
		if pile.Value == trigger.Threshold {
		}
		break
	}
	return false, nil
}

// once the trigger condition is met, the result is analyzed for any effect on the target pile and the "points" given to the task are the additive effect of that operation on the pile
func (e *Engine) watchTriggerCondition(sys *pb.TaskSystem, cond *pb.Trigger, pipeline *pb.Pipeline) error {
	pileThresholdCheckInterval := 120 * time.Second // default polling interval for pile threshold checks

	totalValue, err := e.calculateValue(pipeline)
	if err != nil {
		// log.Debugf("Failed to calculate value for pipeline with steps %v: %w", pipeline.Steps, err)
		totalValue = 0 // fallback to 0 if calculation fails
	}

	if pileThreshold := cond.GetPileThreshold(); pileThreshold != nil {
		log.WithFields(map[string]interface{}{
			"pile.id": cond.GetPileThreshold().PileId,
		}).Debugf("Configuring trigger watcher")
		pile, err := FindPileFatal(cond.GetPileThreshold().PileId, e.State.Piles)
		if err != nil {
			log.
				WithError(err).
				WithFields(map[string]interface{}{
					"pile.id": cond.GetPileThreshold().PileId,
				}).Errorf("Failed to configure trigger watcher")
			return err
		}

		ticker := time.NewTicker(pileThresholdCheckInterval)
		for {
			shouldFire, err := e.checkPileCond(cond.GetPileThreshold().PileId, cond.GetPileThreshold())
			if err != nil {
				log.WithError(err).WithFields(map[string]interface{}{
					"pile.id": cond.GetPileThreshold().PileId,
					"pile":    cond.GetPileThreshold().String(),
				}).Errorf("Failed to check trigger")
			}
			if shouldFire {
				log.WithFields(map[string]interface{}{
					"pile.id":        pile.Id,
					"condition":      cond.GetPileThreshold().String(),
					"step_executing": pipeline.Steps[0].Task.Id,
				}).Infof("Trigger executing")
				e.AssignStep(pipeline.Steps[0], sys, totalValue)
			}
			<-ticker.C
		}
	}
	if triggerInterval := cond.GetInterval(); triggerInterval != nil {
		ticker := time.NewTicker(time.Duration(cond.GetInterval().GetInterval().AsDuration()))

		for {
			<-ticker.C

			e.AssignStep(pipeline.Steps[0], sys, totalValue)
			log.WithFields(map[string]interface{}{
				"interval":       cond.GetInterval().GetInterval().String(),
				"condition":      cond.GetInterval().String(),
				"step_executing": pipeline.Steps[0].Task.Id,
			}).Infof("Trigger executing")
		}
	}
	if weeklySched := cond.GetWeeklySchedule(); weeklySched != nil {
		for {
			now := time.Now()
			next := nextScheduledTime(weeklySched, now)
			log.WithFields(map[string]interface{}{
				"recurrent_target": cond.GetWeeklySchedule().String(),
				"next_time":        next.String(),
			}).Debugf("Trigger scheduled")
			time.Sleep(time.Until(next))

			log.WithFields(map[string]interface{}{
				"recurrent_target": cond.GetWeeklySchedule().String(),
				"condition":        cond.GetWeeklySchedule().String(),
				"step_executing":   pipeline.Steps[0].Task.Id,
			}).Infof("Trigger executing", cond.GetWeeklySchedule())
			e.AssignStep(pipeline.Steps[0], sys, totalValue)
		}
	}
	return nil
}

func evalResult(result *pb.Result, state *pb.SystemState) {
	switch r := result.Result.(type) {
	case *pb.Result_ModifyPile:
		var op func(*pb.Pile)
		switch result.GetModifyPile().Operand {
		case pb.Operand_OPERAND_ADD:
			op = func(pile *pb.Pile) { pile.Value += r.ModifyPile.Value }
			break
		case pb.Operand_OPERAND_SUBTRACT:
			op = func(pile *pb.Pile) { pile.Value -= r.ModifyPile.Value }
			break
		case pb.Operand_OPERAND_SET:
			op = func(pile *pb.Pile) { pile.Value = r.ModifyPile.Value }
			break
		case pb.Operand_OPERAND_MULTIPLY:
			op = func(pile *pb.Pile) { pile.Value *= r.ModifyPile.Value }
			break
		case pb.Operand_OPERAND_DIVIDE:
			op = func(pile *pb.Pile) { pile.Value /= r.ModifyPile.Value }
			break
		}

		queue := make([]*pb.Pile, 0)
		for _, pile := range state.Piles {
			queue = append(queue, pile)
		}

		flag := false
		for len(queue) > 0 {
			// take first pile from queue; remove it
			curr := queue[0]
			queue = queue[1:]

			// if we've found the target or are traversing its children, run the op
			if curr.Id == r.ModifyPile.PileId || flag {
				op(curr)
			}

			if curr.Id == r.ModifyPile.PileId {
				// this is it if we don't care about subpiles
				if result.GetModifyPile().IncludeSubpiles == nil || !*result.GetModifyPile().IncludeSubpiles {
					break
				}

				// otherwise, traverse (only) the subpiles of this one
				// from now on blindly performing the operation on all traversed piles
				// clear the queue (but don't break)
				// now the children will be the only remaining traversees
				queue = make([]*pb.Pile, 0)
				flag = true
			}

			// add children (BFS)
			for _, subpile := range curr.Subpiles {
				queue = append(queue, subpile)
			}
		}

		break
	}

}

func (e *Engine) checkTriggerCondition(cond *pb.Trigger) bool {
	switch c := cond.Condition.(type) {
	case *pb.Trigger_PileThreshold:
		for _, pile := range e.State.Piles {
			if pile.Id == c.PileThreshold.PileId {
				switch c.PileThreshold.Threshold {
				default:
					return pile.Value > c.PileThreshold.Threshold
				}
			}
		}
	case *pb.Trigger_Interval:
		return true // always true; fire every interval cycle (you'd want to use time.Ticker)
	}
	return false
}

// TODO: assign next step in pipeline after completion

func PickUserExcluding(pool []*pb.UserSlot, excluding []*pb.UserSlot) *pb.UserSlot {
	// despite the signature, DeleteFunc modifies the original slice in place, so it'll take a copy of pool
	if poolPruned := slices.DeleteFunc(slices.Clone(pool),
		func(slot *pb.UserSlot) bool {
			if slot == nil {
				return true // delete nil slots in the slice of interst
			}
			return slices.ContainsFunc(excluding, func(otherSlot *pb.UserSlot) bool {
				// don't count other missing slots (in some other array we're checking) against this array
				if otherSlot == nil {
					return false
				}
				return slot.Id == otherSlot.Id
			})
		}); len(poolPruned) > 0 {
		return poolPruned[rand.Intn(len(poolPruned)-1)]
	}
	return nil
}

func PickUser(pool []*pb.UserSlot) *pb.UserSlot {
	if len(pool) == 0 {
		return nil
	}
	return pool[rand.Intn(len(pool))]
}
