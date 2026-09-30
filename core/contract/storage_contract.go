package contract

import (
	"context"
	"io"
)

// ObjectStorage defines the interface for object storage operations.
type ObjectStorage interface {
	// Upload stores data at the given path and returns a URI/URL.
	Upload(context.Context, string, io.ReadSeeker) (string, error)
	// Download retrieves data at the given path.
	Download(context.Context, string) (io.ReadCloser, error)
	// Delete removes the object at the given path.
	Delete(ctx context.Context, path string) error
	// Exists checks if an object exists at the given path.
	Exists(ctx context.Context, path string) (bool, error)
	// ComputeSHA256 calculates the SHA256 hash of the data.
	ComputeSHA256(data []byte) string
}
