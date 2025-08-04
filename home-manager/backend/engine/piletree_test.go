package engine_test

import (
	"cmp"
	"home-tasker/engine"
	pb "home-tasker/goproto/hometasker/v1"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestFindPileFatal(t *testing.T) {
	tree := []*pb.Pile{
		{
			Id: "root1",
			Subpiles: []*pb.Pile{
				{
					Id: "a",
					Subpiles: []*pb.Pile{
						{Id: "a1"},
						{Id: "a2"},
					},
				},
				{Id: "b"},
			},
		},
		{
			Id: "root2",
			Subpiles: []*pb.Pile{
				{
					Id: "a",
					Subpiles: []*pb.Pile{
						{Id: "a1"},
						{Id: "a2"},
					},
				},
				{Id: "b0"},
				{Id: "b"},
				{Id: "b1"},
				{Id: "b2"},
			},
		},
	}

	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		id      string
		tree    []*pb.Pile
		want    *pb.Pile
		wantErr bool
	}{
		{
			id:      "root2.a",
			tree:    tree,
			want:    tree[1].Subpiles[0],
			wantErr: false,
		},
		{
			id:      "root2.aa",
			tree:    tree,
			want:    nil,
			wantErr: true,
		},
		{
			id:      "root1.a",
			tree:    tree,
			want:    tree[0].Subpiles[0],
			wantErr: false,
		},
		{
			id:      "root1.a.aa",
			tree:    tree,
			want:    tree[0].Subpiles[0].Subpiles[0],
			wantErr: false,
		},
		// TODO: Add more test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotErr := engine.FindPileFatal(tt.id, tt.tree)
			if gotErr != nil {
				if !tt.wantErr {
					t.Errorf("FindPileFatal() failed: %v", gotErr)
				}
				return
			}
			if tt.wantErr {
				t.Fatal("FindPileFatal() succeeded unexpectedly")
			}
			// TODO: update the condition below to compare got with tt.want.
			if true {
				t.Errorf("FindPileFatal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFindPile(t *testing.T) {
	tree := []*pb.Pile{
		{
			Id: "root1",
			Subpiles: []*pb.Pile{
				{
					Id: "a",
					Subpiles: []*pb.Pile{
						{Id: "a1"},
						{Id: "a2"},
					},
				},
				{Id: "b"},
			},
		},
		{
			Id: "root2",
			Subpiles: []*pb.Pile{
				{
					Id: "a",
					Subpiles: []*pb.Pile{
						{Id: "a1"},
						{Id: "a2"},
					},
				},
				{Id: "b0"},
				{Id: "b"},
				{Id: "b1"},
				{Id: "b2"},
			},
		},
	}

	tests := []struct {
		name string // description of this test case
		// Named input parameters for target function.
		id   string
		tree []*pb.Pile
		want *pb.Pile
	}{
		{
			id:   "root2.a",
			tree: tree,
			want: tree[1].Subpiles[0],
		},
		{
			id:   "root2.aa",
			tree: tree,
			want: nil,
		},
		{
			id:   "root1.a",
			tree: tree,
			want: tree[0].Subpiles[0],
		},
		{
			id:   "root1.a.aa",
			tree: tree,
			want: tree[0].Subpiles[0].Subpiles[0],
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := engine.FindPile(tt.id, tt.tree)
			diff := cmp.Diff(got, tt.want)
			if cmp.Compare(got, tt.want) != 0 {
				t.Errorf("FindPile() = %v, want %v", got, tt.want)
			}
		})
	}
}
