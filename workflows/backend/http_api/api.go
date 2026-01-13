package httpapi

import (
	"fmt"
	"github.com/DaDevFox/task-systems/workflows/backend/engine"
	"net/http"

	pb "github.com/DaDevFox/task-systems/workflows/backend/pkg/proto/workflows/v1"
)

func Serve(state *pb.SystemState, config *pb.Config, port int) {
	http.HandleFunc("/task/complete", func(w http.ResponseWriter, r *http.Request) {
		task := r.URL.Query().Get("task")
		user := r.URL.Query().Get("user")

		engine.Singleton.CompleteTask(task, user)
	})

	// NFC tag could also hit pile trigger: /pile/add?pile_id=trash&delta=5

	go http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}
