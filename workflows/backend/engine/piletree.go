package engine

import (
	"errors"
	pb "github.com/DaDevFox/task-systems/workflows/backend/pkg/proto/workflows/v1"
	"slices"
)

// FindPile searches for a pile by ID in the provided tree of piles.
func FindPile(id string, tree []*pb.Pile) *pb.Pile {
	queue := make([]*pb.Pile, 0)
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		if curr.Id == id {
			return curr
		}

		for _, sp := range curr.Subpiles {
			queue = append(queue, sp)
		}
	}
	return nil
}

// FindPileFatal searches for a pile by ID in the provided tree of piles.
// It return san error if the pile is not found -- use if you want a thread to kill if essential information is missing
func FindPileFatal(id string, tree []*pb.Pile) (*pb.Pile, error) {
	queue := slices.Clone(tree)

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		if curr.Id == id {
			return curr, nil
		}

		for _, sp := range curr.Subpiles {
			queue = append(queue, sp)
		}
	}

	return nil, errors.New("Pile not found")
}
