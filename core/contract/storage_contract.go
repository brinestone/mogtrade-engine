package contract

// ObjectStorage defines the interface for object storage operations.
// KYC service depends on this interface, not on concrete implementations.
type ObjectStorage interface {
	// Upload stores data at the given path and returns a URI/URL.
	Upload(ctx string, path string, data []byte) (string, error)
	// Download retrieves data at the given path.
	Download(ctx string, path string) ([]byte, error)
	// Delete removes the object at the given path.
	Delete(ctx string, path string) error
	// Exists checks if an object exists at the given path.
	Exists(ctx string, path string) bool
	// ComputeSHA256 calculates the SHA256 hash of the data.
	ComputeSHA256(data []byte) string
}