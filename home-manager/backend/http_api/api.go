package httpapi

import (
	"fmt"
	"home-tasker/engine"
	"net/http"

	pb "home-tasker/goproto/hometasker/v1"
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
