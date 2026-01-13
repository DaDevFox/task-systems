package domain

import (
	"fmt"
	"time"
)

// GroupRole indicates a member's privilege within a group
type GroupRole int

const (
	GroupRoleUnspecified GroupRole = iota
	GroupRoleMember
	GroupRoleAdmin
	GroupRoleOwner
)

func (r GroupRole) String() string {
	switch r {
	case GroupRoleMember:
		return "member"
	case GroupRoleAdmin:
		return "admin"
	case GroupRoleOwner:
		return "owner"
	default:
		return "unspecified"
	}
}

// Member describes a user's role inside a group
type Member struct {
	UserID string    `json:"user_id"`
	Role   GroupRole `json:"role"`
}

// Group models a hierarchical group with subsumption links
type Group struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	Members        map[string]Member `json:"members"`         // userID -> Member
	SubsumedGroups map[string]bool   `json:"subsumed_groups"` // child group IDs
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
}

// NewGroup constructs a new group with the provided id and name
func NewGroup(id, name, ownerUserID string) (*Group, error) {
	if id == "" {
		return nil, fmt.Errorf("group id cannot be empty")
	}
	if name == "" {
		return nil, fmt.Errorf("group name cannot be empty")
	}
	g := &Group{
		ID:             id,
		Name:           name,
		Members:        make(map[string]Member),
		SubsumedGroups: make(map[string]bool),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	if ownerUserID != "" {
		g.Members[ownerUserID] = Member{UserID: ownerUserID, Role: GroupRoleOwner}
	}
	return g, nil
}

// IsMember checks if a user is directly a member of the group
func (g *Group) IsMember(userID string) bool {
	if userID == "" {
		return false
	}
	_, ok := g.Members[userID]
	return ok
}

// GetRole returns the role for a user in the group if present
func (g *Group) GetRole(userID string) GroupRole {
	if m, ok := g.Members[userID]; ok {
		return m.Role
	}
	return GroupRoleUnspecified
}

// AddMember adds or updates a member with a role
func (g *Group) AddMember(userID string, role GroupRole) error {
	if userID == "" {
		return fmt.Errorf("user id cannot be empty")
	}
	if role == GroupRoleUnspecified {
		role = GroupRoleMember
	}
	g.Members[userID] = Member{UserID: userID, Role: role}
	g.UpdatedAt = time.Now()
	return nil
}

// RemoveMember removes a user from the group
func (g *Group) RemoveMember(userID string) {
	delete(g.Members, userID)
	g.UpdatedAt = time.Now()
}

// AddSubsumed adds a child group id to the subsumption set
func (g *Group) AddSubsumed(childID string) error {
	if childID == "" {
		return fmt.Errorf("child group id cannot be empty")
	}
	if childID == g.ID {
		return fmt.Errorf("group cannot subsume itself")
	}
	g.SubsumedGroups[childID] = true
	g.UpdatedAt = time.Now()
	return nil
}
