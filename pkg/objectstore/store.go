package objectstore

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Store 对象存储抽象（本地 / MinIO·S3 兼容 OSS）。
type Store interface {
	Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error
	PublicURL(key string) string
}

type localStore struct {
	root      string
	publicURL string
}

type minioStore struct {
	client    *minio.Client
	bucket    string
	publicURL string
}

func NewFromEnv() (Store, error) {
	driver := strings.ToLower(strings.TrimSpace(os.Getenv("OBJECT_STORE")))
	if driver == "" || driver == "local" {
		root := strings.TrimSpace(os.Getenv("MEDIA_ROOT"))
		if root == "" {
			root = "./data/media"
		}
		pub := strings.TrimRight(strings.TrimSpace(os.Getenv("CDN_PUBLIC_URL")), "/")
		if pub == "" {
			pub = strings.TrimRight(strings.TrimSpace(os.Getenv("MEDIA_PUBLIC_URL")), "/")
		}
		if pub == "" {
			pub = "/media"
		}
		return &localStore{root: root, publicURL: pub}, nil
	}

	if driver != "minio" {
		return nil, fmt.Errorf("unsupported OBJECT_STORE: %s", driver)
	}
	endpoint := os.Getenv("MINIO_ENDPOINT")
	accessKey := os.Getenv("MINIO_ACCESS_KEY")
	secretKey := os.Getenv("MINIO_SECRET_KEY")
	bucket := os.Getenv("MINIO_BUCKET")
	if endpoint == "" || accessKey == "" || secretKey == "" || bucket == "" {
		return nil, fmt.Errorf("minio env incomplete")
	}
	useSSL := os.Getenv("MINIO_USE_SSL") == "1"
	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, err
	}
	ctx := context.Background()
	exists, err := client.BucketExists(ctx, bucket)
	if err != nil {
		return nil, err
	}
	if !exists {
		if err := client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
			return nil, err
		}
	}
	pub := strings.TrimRight(strings.TrimSpace(os.Getenv("CDN_PUBLIC_URL")), "/")
	if pub == "" {
		pub = fmt.Sprintf("/oss/%s", bucket)
	}
	return &minioStore{client: client, bucket: bucket, publicURL: pub}, nil
}

func (s *localStore) Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	_ = ctx
	_ = size
	_ = contentType
	path := joinKey(s.root, key)
	if err := os.MkdirAll(dirOf(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, r)
	return err
}

func (s *localStore) PublicURL(key string) string {
	return s.publicURL + "/" + strings.TrimPrefix(strings.ReplaceAll(key, "\\", "/"), "/")
}

func (s *minioStore) Put(ctx context.Context, key string, r io.Reader, size int64, contentType string) error {
	_, err := s.client.PutObject(ctx, s.bucket, key, r, size, minio.PutObjectOptions{ContentType: contentType})
	return err
}

func (s *minioStore) PublicURL(key string) string {
	return s.publicURL + "/" + strings.TrimPrefix(key, "/")
}

func joinKey(root, key string) string {
	return strings.TrimRight(root, `/\`) + "/" + strings.TrimPrefix(strings.ReplaceAll(key, "\\", "/"), "/")
}

func dirOf(path string) string {
	if i := strings.LastIndexAny(path, `/\`); i >= 0 {
		return path[:i]
	}
	return path
}
