package oss

import (
	"fmt"
	"io"
	"time"

	alioss "github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/gin-gonic/gin"
)

type Config struct {
	Endpoint        string
	AccessKeyID     string
	AccessKeySecret string
	BucketName      string
}

type Storage struct {
	bucket     *alioss.Bucket
	bucketName string
	endpoint   string
}

func NewStorage(cfg Config) (*Storage, error) {
	if cfg.Endpoint == "" || cfg.AccessKeyID == "" || cfg.AccessKeySecret == "" || cfg.BucketName == "" {
		return &Storage{bucketName: cfg.BucketName, endpoint: cfg.Endpoint}, nil
	}

	client, err := alioss.New(cfg.Endpoint, cfg.AccessKeyID, cfg.AccessKeySecret)
	if err != nil {
		return nil, err
	}
	bucket, err := client.Bucket(cfg.BucketName)
	if err != nil {
		return nil, err
	}
	return &Storage{bucket: bucket, bucketName: cfg.BucketName, endpoint: cfg.Endpoint}, nil
}

func (s *Storage) PresignPut(objectKey, contentType string, ttl time.Duration) (string, error) {
	if s.bucket == nil {
		return "", fmt.Errorf("OSS not configured")
	}
	return s.bucket.SignURL(objectKey, alioss.HTTPPut, int64(ttl.Seconds()),
		alioss.ContentType(contentType),
	)
}

func (s *Storage) PresignGet(objectKey string, ttl time.Duration) (string, error) {
	if s.bucket == nil {
		return "", fmt.Errorf("OSS not configured")
	}
	return s.bucket.SignURL(objectKey, alioss.HTTPGet, int64(ttl.Seconds()))
}

func (s *Storage) PutObject(objectKey string, reader io.Reader, contentType string) error {
	if s.bucket == nil {
		return fmt.Errorf("OSS not configured")
	}
	return s.bucket.PutObject(objectKey, reader,
		alioss.ContentType(contentType),
	)
}

func (s *Storage) PutFromMultipart(c *gin.Context, formKey, objectKey string) error {
	fh, err := c.FormFile(formKey)
	if err != nil {
		return err
	}
	src, err := fh.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	return s.bucket.PutObject(objectKey, src,
		alioss.ContentType(fh.Header.Get("Content-Type")),
	)
}

func (s *Storage) PublicURL(objectKey string) string {
	if s.endpoint == "" || s.bucketName == "" {
		return objectKey
	}
	return fmt.Sprintf("https://%s.%s/%s", s.bucketName, trimScheme(s.endpoint), objectKey)
}

func trimScheme(endpoint string) string {
	for _, prefix := range []string{"https://", "http://"} {
		if len(endpoint) > len(prefix) && endpoint[:len(prefix)] == prefix {
			return endpoint[len(prefix):]
		}
	}
	return endpoint
}
