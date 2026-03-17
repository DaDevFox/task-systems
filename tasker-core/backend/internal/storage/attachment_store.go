package storage

import (
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// AttachmentStore writes compressed file attachments to local storage.
type AttachmentStore struct {
	rootDir string
}

func NewAttachmentStore(rootDir string) *AttachmentStore {
	return &AttachmentStore{rootDir: rootDir}
}

func (s *AttachmentStore) StoreFile(sourcePath string) (string, error) {
	if strings.TrimSpace(sourcePath) == "" {
		return "", fmt.Errorf("source path is required")
	}

	source, err := os.Open(sourcePath)
	if err != nil {
		return "", fmt.Errorf("open source file failed: %w", err)
	}
	defer source.Close()

	err = os.MkdirAll(s.rootDir, 0o755)
	if err != nil {
		return "", fmt.Errorf("create attachment root directory failed: %w", err)
	}

	fileName := fmt.Sprintf("%d_%s.gz", time.Now().UTC().UnixNano(), filepath.Base(sourcePath))
	destinationPath := filepath.Join(s.rootDir, fileName)

	destination, err := os.Create(destinationPath)
	if err != nil {
		return "", fmt.Errorf("create destination file failed: %w", err)
	}
	defer destination.Close()

	gzipWriter := gzip.NewWriter(destination)
	_, err = io.Copy(gzipWriter, source)
	if err != nil {
		_ = gzipWriter.Close()
		return "", fmt.Errorf("compress attachment failed: %w", err)
	}

	err = gzipWriter.Close()
	if err != nil {
		return "", fmt.Errorf("finalize compressed attachment failed: %w", err)
	}

	return destinationPath, nil
}
