package blobstore

import (
	"context"
	"io"
	"os"
	"path/filepath"
)

// Blobstore provides file storage functionality
type Blobstore interface {
	// Put stores a file with the given key
	Put(ctx context.Context, key string, data io.Reader) error

	// Get retrieves a file by key
	Get(ctx context.Context, key string) (io.ReadCloser, error)

	// Delete removes a file by key
	Delete(ctx context.Context, key string) error
}

// FileBlobstore implements Blobstore using the filesystem
type FileBlobstore struct {
	basePath string
}

// NewFileBlobstore creates a new file-based blobstore
func NewFileBlobstore(basePath string) (*FileBlobstore, error) {
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, err
	}
	return &FileBlobstore{basePath: basePath}, nil
}

// Put stores a file
func (fb *FileBlobstore) Put(ctx context.Context, key string, data io.Reader) error {
	filePath := filepath.Join(fb.basePath, key)

	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Create file
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Copy data
	_, err = io.Copy(file, data)
	return err
}

// Get retrieves a file
func (fb *FileBlobstore) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	filePath := filepath.Join(fb.basePath, key)
	return os.Open(filePath)
}

// Delete removes a file
func (fb *FileBlobstore) Delete(ctx context.Context, key string) error {
	filePath := filepath.Join(fb.basePath, key)
	return os.Remove(filePath)
}
