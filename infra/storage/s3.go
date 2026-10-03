package storage

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"github.com/brinestone/mogtrade/core/contract"
)

type S3Storage struct {
	StorageConfig
	client *s3.Client
}

func (s *S3Storage) GeneratePresignedUploadURL(ctx context.Context, params contract.PresignUrlParams) (string, error) {
	if params.Bucket == "" {
		params.Bucket = s.Bucket
	}

	if len(params.ContentType) == 0 {
		params.ContentType = append(params.ContentType, "application/octet-stream")
	}

	if params.ObjectName == "" {
		params.ObjectName = generateObjectName()
	}

	if params.Window == 0 {
		params.Window = time.Second
	}
	presignClient := s3.NewPresignClient(s.client)
	url, err := presignClient.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket: &params.Bucket,
		Key:    &params.ObjectName,
	}, func(po *s3.PresignOptions) {
		po.Expires = params.Window
	})
	if err != nil {
		return "", err
	}
	return url.URL, nil
}

func (s *S3Storage) Upload(ctx context.Context, name string, src io.ReadSeeker) (string, error) {
	buf := make([]byte, 20)
	rand.Read(buf)
	objectName := hex.EncodeToString(buf)
	if err := s.assertBucket(ctx, s.Bucket); err != nil {
		return "", err
	}

	contentType := s.detectContenttype(src)
	src.Seek(0, io.SeekStart)

	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      &s.Bucket,
		Key:         &objectName,
		ContentType: &contentType,
		Body:        src,
	})
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/%s/%s", s.Endpoint, s.Bucket, objectName), nil
}

func (s *S3Storage) detectContenttype(src io.Reader) string {
	l := 512
	buf := make([]byte, l)
	src.Read(buf)
	return http.DetectContentType(buf)
}

func (s *S3Storage) Download(ctx context.Context, objectName string) (io.ReadCloser, error) {
	output, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: &s.Bucket,
		Key:    &objectName,
	})
	if err != nil {
		return nil, err
	}
	body := output.Body
	return body, nil
}

func (s *S3Storage) Delete(ctx context.Context, objectName string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: &s.Bucket,
		Key:    &objectName,
	})
	if err != nil {
		return err
	}
	return nil
}

func (s *S3Storage) ComputeSHA256(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func (s *S3Storage) Exists(ctx context.Context, objectName string) (bool, error) {
	err := s.assertBucket(ctx, s.Bucket)
	if err != nil {
		return false, err
	}

	_, err = s.client.HeadObject(ctx, &s3.HeadObjectInput{Bucket: &s.Bucket})
	if err != nil {
		var apiError smithy.APIError
		if errors.As(err, &apiError) {
			switch apiError.(type) {
			case *types.NotFound:
				return false, nil
			default:
				return false, err
			}
		} else {
			return false, err
		}
	}

	return true, nil
}

func (s *S3Storage) assertBucket(ctx context.Context, bucket string) error {
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: &bucket,
	})
	exists := true
	if err != nil {
		var apiError smithy.APIError
		if errors.As(err, &apiError) {
			switch apiError.(type) {
			case *types.NotFound:
				exists = false
				err = nil
			}
		} else {
			return err
		}
	}

	if exists {
		return nil
	}

	_, err = s.client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: &bucket,
	})
	return err
}

func NewS3Storage(accessKey, secretKey string, cfg StorageConfig) (*S3Storage, error) {
	c, err := config.LoadDefaultConfig(context.TODO(), config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")))
	if err != nil {
		return nil, err
	}

	client := s3.NewFromConfig(c)
	return &S3Storage{
		client:        client,
		StorageConfig: cfg,
	}, nil
}
