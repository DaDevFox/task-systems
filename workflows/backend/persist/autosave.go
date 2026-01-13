package persist

import (
	log "github.com/sirupsen/logrus"
	"time"

	pb "github.com/DaDevFox/task-systems/workflows/backend/pkg/proto/workflows/v1"
	"github.com/DaDevFox/task-systems/workflows/backend/state"
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
