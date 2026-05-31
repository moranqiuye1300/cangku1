package model

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

const (
	RoleUser     = "user"
	RoleReviewer = "reviewer"
	RoleAdmin    = "admin"
)

type User struct {
	ID            string
	Username      string
	Password      string
	Nickname      string
	Avatar        string
	Role          string
	OAuthProvider string
	OAuthID       string
	CreatedAt     int64
}

func FormatUserID(id uint64) string {
	return fmt.Sprintf("u%d", id)
}

func ParseUserID(publicID string) (uint64, error) {
	publicID = strings.TrimSpace(publicID)
	if !strings.HasPrefix(publicID, "u") || len(publicID) <= 1 {
		return 0, fmt.Errorf("invalid user id")
	}
	return strconv.ParseUint(publicID[1:], 10, 64)
}

type UserRecord struct {
	ID            uint64    `gorm:"primaryKey;autoIncrement"`
	Username      string    `gorm:"size:50;not null;uniqueIndex"`
	PasswordHash  string    `gorm:"size:255;not null"`
	Nickname      string    `gorm:"size:50;not null;default:''"`
	Avatar        string    `gorm:"size:255;not null;default:''"`
	Role          string    `gorm:"size:16;not null;default:user;index"`
	OAuthProvider string    `gorm:"size:32;index:idx_oauth,priority:1"`
	OAuthID       string    `gorm:"size:128;index:idx_oauth,priority:2"`
	CreatedAt     time.Time `gorm:"autoCreateTime"`
}

func (UserRecord) TableName() string { return "users" }

func RecordToUser(r *UserRecord) *User {
	role := r.Role
	if role == "" {
		role = RoleUser
	}
	return &User{
		ID:        FormatUserID(r.ID),
		Username:  r.Username,
		Password:  r.PasswordHash,
		Nickname:  r.Nickname,
		Avatar:    r.Avatar,
		Role:      role,
		CreatedAt: r.CreatedAt.Unix(),
	}
}
