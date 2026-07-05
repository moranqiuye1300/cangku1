package storage

import (
	"context"
	"os"
	"path/filepath"
	"short-video-platform/pkg/objectstore"
	"strings"
	"testing"
)

func TestChunkUploadSessionAssemblesFile(t *testing.T) {
	mediaRoot := t.TempDir()
	session, err := CreateUploadSession(mediaRoot, "u1", "upload-1", "demo.mp4", "video/mp4", 11, 4)
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	if err := SaveUploadChunk(mediaRoot, session.ID, 0, strings.NewReader("hello"), 5); err != nil {
		t.Fatalf("save chunk0: %v", err)
	}
	if err := SaveUploadChunk(mediaRoot, session.ID, 1, strings.NewReader(" world"), 6); err != nil {
		t.Fatalf("save chunk1: %v", err)
	}
	if err := SaveUploadChunk(mediaRoot, session.ID, 2, strings.NewReader("!"), 1); err != nil {
		t.Fatalf("save chunk2: %v", err)
	}

	relPath, err := CompleteUploadFromChunks(mediaRoot, session.ID, "u1")
	if err != nil {
		t.Fatalf("complete upload: %v", err)
	}

	absPath := filepath.Join(mediaRoot, filepath.FromSlash(relPath))
	data, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatalf("read assembled file: %v", err)
	}
	if string(data) != "hello world!" {
		t.Fatalf("unexpected assembled content: %q", string(data))
	}
}

func TestSaveUploadWritesFileToMediaRoot(t *testing.T) {
	mediaRoot := t.TempDir()
	data := strings.NewReader("video-bytes")

	relPath, err := SaveUpload(mediaRoot, "u1", "tmp-1", "demo.mp4", data)
	if err != nil {
		t.Fatalf("SaveUpload: %v", err)
	}

	absPath := filepath.Join(mediaRoot, filepath.FromSlash(relPath))
	content, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if string(content) != "video-bytes" {
		t.Fatalf("unexpected content: %q", string(content))
	}
}

func TestMaybePutToObjectStoreUsesLocalStoreByDefault(t *testing.T) {
	mediaRoot := t.TempDir()
	absPath := filepath.Join(mediaRoot, "demo.mp4")
	if err := os.WriteFile(absPath, []byte("x"), 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	if err := maybePutToObjectStore("uploads/demo.mp4", absPath); err != nil {
		t.Fatalf("maybePutToObjectStore: %v", err)
	}
}

func TestObjectStorePutIsInvokedWhenConfigured(t *testing.T) {
	oldValue := os.Getenv("OBJECT_STORE")
	defer os.Setenv("OBJECT_STORE", oldValue)
	if err := os.Setenv("OBJECT_STORE", "minio"); err != nil {
		t.Fatalf("set OBJECT_STORE: %v", err)
	}
	oldEndpoint := os.Getenv("MINIO_ENDPOINT")
	defer os.Setenv("MINIO_ENDPOINT", oldEndpoint)
	oldKey := os.Getenv("MINIO_ACCESS_KEY")
	defer os.Setenv("MINIO_ACCESS_KEY", oldKey)
	oldSecret := os.Getenv("MINIO_SECRET_KEY")
	defer os.Setenv("MINIO_SECRET_KEY", oldSecret)
	oldBucket := os.Getenv("MINIO_BUCKET")
	defer os.Setenv("MINIO_BUCKET", oldBucket)
	if err := os.Setenv("MINIO_ENDPOINT", "127.0.0.1:9000"); err != nil {
		t.Fatalf("set MINIO_ENDPOINT: %v", err)
	}
	if err := os.Setenv("MINIO_ACCESS_KEY", "minioadmin"); err != nil {
		t.Fatalf("set MINIO_ACCESS_KEY: %v", err)
	}
	if err := os.Setenv("MINIO_SECRET_KEY", "minioadmin123"); err != nil {
		t.Fatalf("set MINIO_SECRET_KEY: %v", err)
	}
	if err := os.Setenv("MINIO_BUCKET", "svp-media"); err != nil {
		t.Fatalf("set MINIO_BUCKET: %v", err)
	}

	store, err := objectstore.NewFromEnv()
	if err != nil {
		t.Fatalf("NewFromEnv: %v", err)
	}
	ctx := context.Background()
	if err := store.Put(ctx, "uploads/test-objectstore.txt", strings.NewReader("hello"), int64(len("hello")), "text/plain"); err != nil {
		t.Fatalf("store.Put: %v", err)
	}
}
