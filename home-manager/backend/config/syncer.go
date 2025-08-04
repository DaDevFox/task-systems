package config

import (
	log "github.com/sirupsen/logrus"
	"time"

	pb "home-tasker/goproto/hometasker/v1"
)

// SyncConfig is a thread (blocing function) which checks the current system state
// against the provided configuration at a fixed interval
// and syncs any piles recently added to the config which aren't in the state
func SyncConfig(config *pb.Config, st *pb.SystemState, syncInterval time.Duration) {
	ticker := time.NewTicker(syncInterval)
	defer ticker.Stop()

	for {
		checkAddedPiles(config, st)
		checkDeletedPiles(config, st)
		<-ticker.C
	}
}

// TODO: verify below
// checkDeletedPiles removes piles from state that are no longer present in the config.
func checkDeletedPiles(config *pb.Config, st *pb.SystemState) {
	// Build a set of valid pile IDs from the config using BFS
	validIDs := make(map[string]struct{})
	queue := make([]*pb.PileConfig, 0)
	for _, pileConfig := range config.Piles {
		queue = append(queue, pileConfig)
	}
	for len(queue) > 0 {
		pileConfig := queue[0]
		queue = queue[1:]
		validIDs[pileConfig.Id] = struct{}{}
		for _, subpile := range pileConfig.Subpiles {
			queue = append(queue, subpile)
		}
	}

	// Filter st.Piles to only those present in validIDs
	filtered := make([]*pb.Pile, 0, len(st.Piles))
	for _, pile := range st.Piles {
		if _, ok := validIDs[pile.Id]; ok {
			filtered = append(filtered, pile)
		} else {
			log.WithFields(map[string]interface{}{
				"pile.id": pile.Id,
				"pile":    pile.String(),
			}).Debugf("Config change detected: Removed pile [missing from config]")
		}
	}
	st.Piles = filtered
}

func checkAddedPiles(config *pb.Config, st *pb.SystemState) {
	configQueue := make([](struct {
		*pb.PileConfig
		*pb.Pile
	}), 0)
	for _, pileConfig := range config.Piles {
		configQueue = append(configQueue, struct {
			*pb.PileConfig
			*pb.Pile
		}{
			pileConfig,
			nil,
		})
	}

	for len(configQueue) > 0 {
		stateParent := configQueue[0].Pile
		pileConfig := configQueue[0].PileConfig
		// log.Debugf("Checking %v\n", pileConfig)
		configQueue = configQueue[1:]

		found := false
		var currPile *pb.Pile
		if stateParent != nil {
			for _, pile := range stateParent.Subpiles {
				if pile.Id == pileConfig.Id {
					found = true
					currPile = pile
					break
				}
			}
		} else {
			for _, pile := range st.Piles {
				if pile.Id == pileConfig.Id {
					found = true
					currPile = pile
					break
				}
			}
		}
		if !found {
			var res *pb.Pile
			res = new(pb.Pile)
			// TODO: decide inclusion of appending inhereted parent IDs
			// if parent != nil {
			// 	res.Id = strings.Join([]string{parent.Id, pileConfig.Id}, ".")
			// } else {
			res.Id = pileConfig.Id
			// }
			res.DisplayName = pileConfig.Name
			res.Value = pileConfig.InitialValue
			res.MaxValue = pileConfig.MaxValue

			if stateParent != nil {
				stateParent.Subpiles = append(stateParent.Subpiles, res)
			} else {
				st.Piles = append(st.Piles, res)
			}
			currPile = res
			log.WithFields(map[string]interface{}{
				"pile.id":       res.Id,
				"pile":          res.String(),
				"initial_value": pileConfig.InitialValue,
			}).Debugf("Config change detected: Added pile [missing from state]")
		}

		// Enqueue subpiles for BFS
		for _, subpile := range pileConfig.Subpiles {
			configQueue = append(configQueue, struct {
				*pb.PileConfig
				*pb.Pile
			}{subpile, currPile})
		}
	}

}
