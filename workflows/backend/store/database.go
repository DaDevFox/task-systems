package store

import (
	"sync"
)

type DB struct {
	mu   sync.Mutex
	pile int
}

func NewDB(path string) *DB {
	return &DB{pile: 0}
}

func (db *DB) AssignTask(task, user string) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	// Insert logic here for task assignment
	return nil
}

func (db *DB) GetCurrentPile() int {
	db.mu.Lock()
	defer db.mu.Unlock()
	return db.pile
}
