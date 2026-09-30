package storage

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinIOStorage struct {
	MinIOConfig
	client *minio.Client
}
type MinIOConfig struct {
	Endpoint string
	UseSsl   bool
	Region   string
	Bucket   string
}

func (m *MinIOStorage) ComputeSHA256(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}
func (m *MinIOStorage) Exists(ctx context.Context, objectName string) (bool, error) {
	c, cancel := context.WithTimeout(ctx, time.Second*2000)
	defer cancel()
	_, ok := <-m.client.ListObjects(c, m.Bucket, minio.ListObjectsOptions{Prefix: objectName})
	return ok, c.Err()
}
func (m *MinIOStorage) Delete(ctx context.Context, objectName string) error {
	return m.client.RemoveObject(ctx, m.Bucket, objectName, minio.RemoveObjectOptions{ForceDelete: true})
}
func (m *MinIOStorage) Download(ctx context.Context, objectName string) (io.ReadCloser, error) {
	return m.client.GetObject(ctx, m.Bucket, objectName, minio.GetObjectOptions{})
}

func (m *MinIOStorage) Upload(ctx context.Context, path string, src io.ReadSeeker) (string, error) {
	buf := make([]byte, 20)
	rand.Read(buf)
	objectName := hex.EncodeToString(buf)
	if err := m.assertBucket(ctx); err != nil {
		return "", err
	}

	sample := make([]byte, 512)
	contentType := m.detectContentType(sample)
	src.Seek(0, io.SeekStart)

	_, err := m.client.PutObject(ctx, m.Bucket, objectName, src, -1, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/%s/%s", m.Endpoint, m.Bucket, objectName), nil
}

func (m *MinIOStorage) detectContentType(src []byte) string {
	return http.DetectContentType(src)
}

func (m *MinIOStorage) assertBucket(ctx context.Context) error {
	if err := m.client.MakeBucket(ctx, m.Bucket, minio.MakeBucketOptions{}); err != nil {
		exists, errBucketExists := m.client.BucketExists(ctx, m.Bucket)
		if errBucketExists != nil && !exists {
			return err
		}
	}
	return nil
}

func NewMinIOObjectStorage(accessKey, secretKey string, cfg MinIOConfig) (*MinIOStorage, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: cfg.UseSsl,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, err
	}
	return &MinIOStorage{
		client:      client,
		MinIOConfig: cfg,
	}, nil
}
