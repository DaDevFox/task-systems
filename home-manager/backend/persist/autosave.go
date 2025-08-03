package persist

import (
	log "github.com/sirupsen/logrus"
	"time"

	pb "home-tasker/goproto/hometasker/v1"
	"home-tasker/state"
)

func StartAutoSave(statePath string, s *pb.SystemState, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			if err := state.SaveState(statePath, s); err != nil {
				log.Println("Error saving state:", err)
			}
		}
	}()
}

