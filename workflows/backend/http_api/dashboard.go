package httpapi

import (
	"fmt"
	log "github.com/sirupsen/logrus"
	"html/template"
	"net"
	"net/http"

	pb "github.com/DaDevFox/task-systems/workflows/backend/pkg/proto/workflows/v1"
)

func FlattenPiles(currPile *pb.Pile, piles *[]*pb.Pile) {
	*piles = append(*piles, currPile)

	for _, subPile := range currPile.Subpiles {
		FlattenPiles(subPile, piles)
	}
}

func ServeDashboard(state *pb.SystemState, port int) {
	tmpl := template.Must(template.ParseFiles("templates/dashboard.html"))
	log.Infof("Starting dashboard on http://localhost:%d\n", port)

	piles := []*pb.Pile{}

	for _, pile := range state.Piles {
		FlattenPiles(pile, &piles)
	}

	// log.Infof("%d piles, %d triggers/task groups active\n", len(piles), len(Tasks))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		Tasks := []*pb.TaskRecord{}
		for _, pipeline := range state.PipelineActivity {
			for _, work := range pipeline.PipelineWork {
				Tasks = append(Tasks, work.Task)
			}
		}

		log.WithField("taskc", len(Tasks)).Debug("serving dash")

		tmpl.Execute(w, struct {
			Host      string
			Piles     []*pb.Pile
			Pipelines []*pb.PipelineActivity
			Tasks     []*pb.TaskRecord
		}{
			Host:      GetLocalIP(),
			Piles:     piles,
			Pipelines: state.PipelineActivity,
			Tasks:     Tasks,
		})
	})

	// http.HandleFunc("/leaderboard", func(w http.ResponseWriter, r *http.Request) {
	// 	type row struct {
	// 		User         string
	// 		Completed    int
	// 		OnTime       int
	// 		AvgEfficiency float32
	// 	}
	// 	stats := map[string]*row{}
	// 	for _, t := range state.TaskHistory {
	// 		r, ok := stats[t.User]
	// 		if !ok {
	// 			r = &row{User: t.User}
	// 			stats[t.User] = r
	// 		}
	// 		if t.Status == "completed" || t.Status == "reviewed" {
	// 			r.Completed++
	// 			if t.OnTime {
	// 				r.OnTime++
	// 			}
	// 			r.AvgEfficiency += t.EfficiencyScore
	// 		}
	// 	}
	// 	var rows []row
	// 	for _, v := range stats {
	// 		if v.Completed > 0 {
	// 			v.AvgEfficiency /= float32(v.Completed)
	// 		}
	// 		rows = append(rows, *v)
	// 	}
	// 	for _, r := range rows {
	// 		w.Write([]byte(
	// 			r.User + " | Completed: " +
	// 			fmt.Sprintf("%d", r.Completed) + " | On Time: " +
	// 			fmt.Sprintf("%d", r.OnTime) + " | Avg Eff: " +
	// 			fmt.Sprintf("%.2f", r.AvgEfficiency) + "\n"))
	// 	}
	// })

	go http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}

func GetLocalIP() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "unknown"
	}
	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip != nil && !ip.IsLoopback() && ip.To4() != nil {
				return ip.String()
			}
		}
	}
	return "localhost"
}
