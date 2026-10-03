package contract

import (
	"context"
	"io"
	"time"
)

type PresignUrlParams struct {
	Bucket      string
	ObjectName  string
	ContentType []string
	Window      time.Duration
}

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
	// GeneratePresignedUploadURL generates a presigned URL for uploading an object
	// with content type validation. The urlExpires after the specified duration.
	// allowedContentTypes specifies the MIME types that are allowed for the upload.
	GeneratePresignedUploadURL(ctx context.Context, params PresignUrlParams) (string, error)
}
