package auth

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ValidateSourcePathForUser ensures upload paths cannot escape or impersonate another user.
func ValidateSourcePathForUser(userID, sourcePath string) error {
	if userID == "" || sourcePath == "" {
		return fmt.Errorf("user_id and source_path required")
	}
	clean := filepath.ToSlash(filepath.Clean(sourcePath))
	if clean != sourcePath || strings.Contains(clean, "..") {
		return fmt.Errorf("invalid source path")
	}
	prefix := "uploads/u" + userID + "/"
	if !strings.HasPrefix(clean, prefix) {
		return fmt.Errorf("source path must be under uploader directory")
	}
	base := filepath.Base(clean)
	if !strings.HasPrefix(base, "source") {
		return fmt.Errorf("invalid source filename")
	}
	return nil
}

// CanAccessUploadPath returns true if role may read raw upload files at relPath.
func CanAccessUploadPath(role, userID, relPath string) bool {
	clean := filepath.ToSlash(filepath.Clean(relPath))
	if strings.Contains(clean, "..") || !strings.HasPrefix(clean, "uploads/") {
		return false
	}
	if role == RoleAdmin || role == RoleReviewer {
		return true
	}
	if userID == "" {
		return false
	}
	return strings.HasPrefix(clean, "uploads/u"+userID+"/")
}
