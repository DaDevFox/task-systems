package repository

import (
	"fmt"
	"strings"
)

// DatabaseType represents different database backend options
type DatabaseType string

const (
	DatabaseTypeBadger DatabaseType = "badger"
	DatabaseTypeBolt   DatabaseType = "bolt"
)

// NewInventoryRepository creates a new inventory repository with the specified database type
//
// Database Types:
// - badger: High-performance LSM-tree database (default), but creates large .vlog files
// - bolt: Compact B+ tree database, much smaller files, good for smaller datasets
func NewInventoryRepository(dbPath string, dbType DatabaseType) (InventoryRepository, error) {
	switch dbType {
	case DatabaseTypeBolt:
		// Use .bolt extension for BoltDB files
		if !strings.HasSuffix(dbPath, ".bolt") {
			dbPath = dbPath + ".bolt"
		}
		return NewBoltInventoryRepository(dbPath)

	case DatabaseTypeBadger:
		// BadgerDB uses directory-based storage
		return NewBadgerInventoryRepository(dbPath)

	default:
		return nil, fmt.Errorf("unsupported database type: %s", dbType)
	}
}

// NewCompactInventoryRepository creates a new inventory repository optimized for smaller file sizes
// This uses BoltDB which creates much smaller database files compared to BadgerDB
func NewCompactInventoryRepository(dbPath string) (InventoryRepository, error) {
	return NewInventoryRepository(dbPath, DatabaseTypeBolt)
}

// GetDatabaseInfo returns information about the different database options
func GetDatabaseInfo() map[DatabaseType]string {
	return map[DatabaseType]string{
		DatabaseTypeBadger: "High-performance LSM-tree database. Fast for writes and large datasets, but creates large .vlog files (>2GB common). Good for high-throughput applications.",
		DatabaseTypeBolt:   "Compact B+ tree database. Much smaller file sizes (typically <100MB), single file storage. Good for smaller datasets and embedded applications.",
	}
}
