package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"short-video-platform/pkg/objectstore"
)

type UploadSession struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	OriginalName string    `json:"original_name"`
	ContentType  string    `json:"content_type"`
	TotalSize    int64     `json:"total_size"`
	ChunkSize    int64     `json:"chunk_size"`
	ChunkCount   int       `json:"chunk_count"`
	Uploaded     []int     `json:"uploaded,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	Completed    bool      `json:"completed"`
	FinalPath    string    `json:"final_path,omitempty"`
}

const defaultChunkSize = 5 << 20

func SaveUpload(mediaRoot, userID, uploadID, originalName string, r io.Reader) (string, error) {
	ext := filepath.Ext(originalName)
	if ext == "" {
		ext = ".mp4"
	}
	rel := filepath.ToSlash(filepath.Join("uploads", "u"+userID, uploadID, "source"+ext))
	abs := filepath.Join(mediaRoot, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return "", err
	}
	f, err := os.Create(abs)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := io.Copy(f, r); err != nil {
		return "", err
	}
	if err := maybePutToObjectStore(rel, abs); err != nil {
		return "", err
	}
	return rel, nil
}

func CreateUploadSession(mediaRoot, userID, uploadID, originalName, contentType string, totalSize, chunkSize int64) (*UploadSession, error) {
	if userID == "" {
		return nil, fmt.Errorf("user_id required")
	}
	if totalSize <= 0 {
		return nil, fmt.Errorf("total size must be positive")
	}
	if chunkSize <= 0 {
		chunkSize = defaultChunkSize
	}
	if chunkSize > 50<<20 {
		chunkSize = 50 << 20
	}
	if uploadID == "" {
		uploadID = fmt.Sprintf("session-%d", time.Now().UnixNano())
	}
	session := &UploadSession{
		ID:           uploadID,
		UserID:       userID,
		OriginalName: originalName,
		ContentType:  contentType,
		TotalSize:    totalSize,
		ChunkSize:    chunkSize,
		ChunkCount:   int((totalSize + chunkSize - 1) / chunkSize),
		CreatedAt:    time.Now().UTC(),
	}
	if err := saveUploadSession(mediaRoot, session); err != nil {
		return nil, err
	}
	return session, nil
}

func GetUploadSessionStatus(mediaRoot, sessionID string) (*UploadSession, error) {
	if sessionID == "" {
		return nil, fmt.Errorf("session_id required")
	}
	session, err := loadUploadSession(mediaRoot, sessionID)
	if err != nil {
		return nil, err
	}
	chunkDir := filepath.Join(mediaRoot, "uploads", ".chunks", session.ID)
	uploaded := make([]int, 0, session.ChunkCount)
	for i := 0; i < session.ChunkCount; i++ {
		chunkPath := filepath.Join(chunkDir, fmt.Sprintf("%d.bin", i))
		if _, err := os.Stat(chunkPath); err == nil {
			uploaded = append(uploaded, i)
		}
	}
	session.Uploaded = uploaded
	return session, nil
}

func SaveUploadChunk(mediaRoot, sessionID string, chunkIndex int, r io.Reader, size int64) error {
	if sessionID == "" {
		return fmt.Errorf("session_id required")
	}
	session, err := loadUploadSession(mediaRoot, sessionID)
	if err != nil {
		return err
	}
	if chunkIndex < 0 || chunkIndex >= session.ChunkCount {
		return fmt.Errorf("invalid chunk index")
	}
	chunkDir := filepath.Join(mediaRoot, "uploads", ".chunks", session.ID)
	if err := os.MkdirAll(chunkDir, 0o755); err != nil {
		return err
	}
	chunkPath := filepath.Join(chunkDir, fmt.Sprintf("%d.bin", chunkIndex))
	tmpPath := chunkPath + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return err
	}
	defer os.Remove(tmpPath)
	defer f.Close()
	if _, err := io.Copy(f, r); err != nil {
		return err
	}
	if size > 0 {
		if info, err := f.Stat(); err == nil && info.Size() != size {
			return fmt.Errorf("chunk size mismatch")
		}
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, chunkPath); err != nil {
		return err
	}
	return nil
}

func CompleteUploadFromChunks(mediaRoot, sessionID, userID string) (string, error) {
	if sessionID == "" {
		return "", fmt.Errorf("session_id required")
	}
	session, err := loadUploadSession(mediaRoot, sessionID)
	if err != nil {
		return "", err
	}
	if session.UserID != "" && session.UserID != userID {
		return "", fmt.Errorf("session belongs to another user")
	}
	if session.Completed && session.FinalPath != "" {
		return session.FinalPath, nil
	}
	for i := 0; i < session.ChunkCount; i++ {
		chunkPath := filepath.Join(mediaRoot, "uploads", ".chunks", session.ID, fmt.Sprintf("%d.bin", i))
		if _, err := os.Stat(chunkPath); err != nil {
			return "", fmt.Errorf("missing chunk %d", i)
		}
	}
	ext := filepath.Ext(session.OriginalName)
	if ext == "" {
		ext = ".mp4"
	}
	rel := filepath.ToSlash(filepath.Join("uploads", "u"+userID, session.ID, "source"+ext))
	abs := filepath.Join(mediaRoot, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return "", err
	}
	out, err := os.Create(abs)
	if err != nil {
		return "", err
	}
	for i := 0; i < session.ChunkCount; i++ {
		chunkPath := filepath.Join(mediaRoot, "uploads", ".chunks", session.ID, fmt.Sprintf("%d.bin", i))
		chunkFile, err := os.Open(chunkPath)
		if err != nil {
			_ = out.Close()
			return "", err
		}
		if _, err := io.Copy(out, chunkFile); err != nil {
			_ = chunkFile.Close()
			_ = out.Close()
			return "", err
		}
		_ = chunkFile.Close()
	}
	if err := out.Close(); err != nil {
		return "", err
	}
	if err := maybePutToObjectStore(rel, abs); err != nil {
		return "", err
	}
	session.Completed = true
	session.FinalPath = rel
	if err := saveUploadSession(mediaRoot, session); err != nil {
		return "", err
	}
	return rel, nil
}

func SaveAvatar(mediaRoot, userID, originalName string, r io.Reader) (string, error) {
	ext := strings.ToLower(filepath.Ext(originalName))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp", ".gif":
	default:
		return "", fmt.Errorf("unsupported image type")
	}
	rel := filepath.ToSlash(filepath.Join("avatars", userID+ext))
	abs := filepath.Join(mediaRoot, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		return "", err
	}
	f, err := os.Create(abs)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := io.Copy(f, r); err != nil {
		return "", err
	}
	return "/media/" + rel, nil
}

func MediaRoot() string {
	root := strings.TrimSpace(os.Getenv("MEDIA_ROOT"))
	if root == "" {
		return "./data/media"
	}
	return root
}

func NextUploadVideoID() string {
	return fmt.Sprintf("tmp-%d", os.Getpid())
}

func loadUploadSession(mediaRoot, sessionID string) (*UploadSession, error) {
	path := filepath.Join(mediaRoot, "uploads", ".sessions", sessionID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var session UploadSession
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, err
	}
	return &session, nil
}

func saveUploadSession(mediaRoot string, session *UploadSession) error {
	path := filepath.Join(mediaRoot, "uploads", ".sessions", session.ID+".json")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(session, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func maybePutToObjectStore(relPath, absPath string) error {
	driver := strings.ToLower(strings.TrimSpace(os.Getenv("OBJECT_STORE")))
	if driver == "" || driver == "local" {
		return nil
	}
	store, err := objectstore.NewFromEnv()
	if err != nil {
		return err
	}
	file, err := os.Open(absPath)
	if err != nil {
		return err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	return store.Put(context.Background(), relPath, file, info.Size(), "video/mp4")
}
