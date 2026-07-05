package repository

import (
	"context"
	"errors"

	"short-video-platform/video-service/internal/model"
)

var (
	ErrNotFound       = errors.New("video not found")
	ErrStatusConflict = errors.New("status transition conflict")
)

type VideoRepository interface {
	List(ctx context.Context, page, pageSize int) ([]model.Video, int, error)
	ListPublished(ctx context.Context, page, pageSize int) ([]model.Video, int, error)
	ListByStatus(ctx context.Context, status string, page, pageSize int) ([]model.Video, int, error)
	ListAll(ctx context.Context, limit int) ([]model.Video, error)
	ListByUser(ctx context.Context, userID string, page, pageSize int) ([]model.Video, int, error)
	ListByUserPublished(ctx context.Context, userID string, page, pageSize int) ([]model.Video, int, error)
	GetByID(ctx context.Context, id string) (*model.Video, error)
	GetByIDIncludingDeleted(ctx context.Context, id string) (*model.Video, error)
	Count(ctx context.Context) (int64, error)
	NextVideoID(ctx context.Context) (string, error)
	InsertMany(ctx context.Context, videos []model.Video) error
	Create(ctx context.Context, video *model.Video) error
	UpdateTranscodeResult(ctx context.Context, videoID, status string, duration int32, coverURL string, playURLs map[string]string, errMsg string, tags []string) (*model.Video, error)
	UpdateStatus(ctx context.Context, videoID, status string) error
	UpdateStatusIf(ctx context.Context, videoID, fromStatus, toStatus string) error
	AdminList(ctx context.Context, page, pageSize int, includeDeleted bool) ([]model.Video, int, error)
	ListReadyActive(ctx context.Context, limit int) ([]model.Video, error)
	ListDeleted(ctx context.Context, page, pageSize int) ([]model.Video, int, error)
	SoftDelete(ctx context.Context, videoID, deletedBy, reason string, deletedAt, purgeAt int64) (*model.Video, error)
	Restore(ctx context.Context, videoID string) (*model.Video, error)
	PermanentDelete(ctx context.Context, videoID string) error
	UpdateTags(ctx context.Context, videoID string, tags []string) error
}
