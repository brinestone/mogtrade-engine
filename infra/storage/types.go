package storage

import "fmt"

type StorageConfig struct {
	Endpoint string
	UseSsl   bool
	Region   string
	Bucket   string
}

type StoragePlatform string

const (
	StoragePlatformMinio StoragePlatform = "minio"
	StoragePlatformS3    StoragePlatform = "s3"
)

func (s *StoragePlatform) Scan(v any) error {
	switch t := v.(type) {
	case []byte:
		*s = StoragePlatform(t)
	case string:
		*s = StoragePlatform(t)
	default:
		return fmt.Errorf("unsupported scan type for StoragePlatform: %T", v)
	}
	return nil
}
