package storage

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"

	"github.com/brinestone/mogtrade/core/contract"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type MinIOStorage struct {
	StorageConfig
	client *minio.Client
}

func (m *MinIOStorage) GeneratePresignedUploadURL(ctx context.Context, params contract.PresignUrlParams) (string, error) {
	if params.Bucket == "" {
		params.Bucket = m.Bucket
	}

	if len(params.ContentType) == 0 {
		params.ContentType = append(params.ContentType, "application/octet-stream")
	}

	if params.ObjectName == "" {
		params.ObjectName = generateObjectName()
	}

	if params.Window <= 0 {
		params.Window = time.Second
	}

	params.Window = time.Duration(math.Min(float64(time.Hour*168), float64(params.Window)))
	url, err := m.client.PresignedPutObject(ctx, params.Bucket, params.ObjectName, params.Window)
	if err != nil {
		return "", err
	}
	return url.String(), nil
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

func (m *MinIOStorage) Upload(ctx context.Context, objectName string, src io.ReadSeeker) (string, error) {
	if objectName == "" {
		objectName = generateObjectName()
	}
	if err := m.assertBucket(ctx); err != nil {
		return "", err
	}

	contentType := m.detectContentType(src)
	src.Seek(0, io.SeekStart)

	_, err := m.client.PutObject(ctx, m.Bucket, objectName, src, -1, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/%s/%s", m.Endpoint, m.Bucket, objectName), nil
}

func (m *MinIOStorage) detectContentType(src io.Reader) string {
	l := 512
	var buf []byte = make([]byte, l)
	src.Read(buf)
	return http.DetectContentType(buf)
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
func generateObjectName() string {
	buf := make([]byte, 20)
	rand.Read(buf)
	return fmt.Sprintf("def_%s", hex.EncodeToString(buf))
}

func NewMinIOObjectStorage(accessKey, secretKey string, cfg StorageConfig) (*MinIOStorage, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: cfg.UseSsl,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, err
	}
	return &MinIOStorage{
		client:        client,
		StorageConfig: cfg,
	}, nil
}
