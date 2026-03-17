package storage

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAttachmentStore(t *testing.T) {
	tempDir := t.TempDir()
	sourcePath := filepath.Join(tempDir, "input.txt")
	err := os.WriteFile(sourcePath, []byte("demo content"), 0o644)
	if err != nil {
		t.Fatalf("write source file failed: %v", err)
	}

	store := NewAttachmentStore(filepath.Join(tempDir, "attachments"))
	storedPath, err := store.StoreFile(sourcePath)
	if err != nil {
		t.Fatalf("store file failed: %v", err)
	}

	_, err = os.Stat(storedPath)
	if err != nil {
		t.Fatalf("stored file does not exist: %v", err)
	}
}
