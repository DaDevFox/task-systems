// server/leaderboard_service.go
package server

import (
	"context"
	"sort"

	pb "home-tasker/goproto/hometasker/v1"
)

func NewLeaderboardService(state *pb.SystemState) *HometaskerServiceServer {
	return &HometaskerServiceServer{State: state}
}

func (s *HometaskerServiceServer) GetLeaderboard(ctx context.Context, req *pb.LeaderboardRequest) (*pb.LeaderboardResponse, error) {
	userStats := make(map[string]*pb.LeaderboardEntry)
	for _, event := range s.State.TaskHistory {
		entry, exists := userStats[event.User]
		if !exists {
			entry = &pb.LeaderboardEntry{User: event.User}
			userStats[event.User] = entry
		}
		if event.Status == "completed" || event.Status == "reviewed" {
			entry.Completed++
			entry.AvgEfficiency += event.EfficiencyScore
			if event.OnTime {
				entry.OnTime++
			}
		}
	}

	// Finalize efficiency score
	entries := []*pb.LeaderboardEntry{}
	for _, e := range userStats {
		if e.Completed > 0 {
			e.AvgEfficiency /= float32(e.Completed)
		}
		entries = append(entries, e)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].AvgEfficiency > entries[j].AvgEfficiency
	})

	return &pb.LeaderboardResponse{Entries: entries}, nil
}

