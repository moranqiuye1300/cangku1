package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func SaveUpload(mediaRoot, videoID, originalName string, r io.Reader) (string, error) {
	ext := filepath.Ext(originalName)
	if ext == "" {
		ext = ".mp4"
	}
	rel := filepath.ToSlash(filepath.Join("uploads", videoID, "source"+ext))
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
