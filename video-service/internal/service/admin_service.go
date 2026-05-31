package service

import (
	"context"
	"fmt"
	"time"

	"short-video-platform/video-service/internal/model"
)

const complianceRetentionDays = 365 * 2

type AdminOp struct {
	VideoID          string
	OperatorID       string
	OperatorUsername string
	Reason           string
	IP               string
	UserAgent        string
}

func (s *VideoService) AdminListVideos(ctx context.Context, page, pageSize int, includeDeleted bool) ([]model.Video, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.repo.AdminList(ctx, page, pageSize, includeDeleted)
}

func (s *VideoService) ListRecycleBin(ctx context.Context, page, pageSize int) ([]model.Video, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.repo.ListDeleted(ctx, page, pageSize)
}

func (s *VideoService) ListAuditLogs(ctx context.Context, page, pageSize int, targetType string) ([]model.AuditLog, int, error) {
	if s.auditRepo == nil {
		return nil, 0, fmt.Errorf("audit unavailable")
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.auditRepo.List(ctx, page, pageSize, targetType)
}

func (s *VideoService) SoftDeleteVideo(ctx context.Context, op AdminOp) (*model.Video, error) {
	v, err := s.repo.GetByID(ctx, op.VideoID)
	if err != nil {
		return nil, err
	}

	now := time.Now().Unix()
	purgeAt := time.Now().AddDate(0, 0, model.RecycleBinDays).Unix()
	deleted, err := s.repo.SoftDelete(ctx, op.VideoID, op.OperatorID, op.Reason, now, purgeAt)
	if err != nil {
		return nil, err
	}

	cleared, err := s.interactRepo.ClearAllForVideo(ctx, op.VideoID)
	if err != nil {
		return nil, err
	}
	_ = s.barrageRepo.DeleteByVideo(ctx, op.VideoID)
	if s.search != nil {
		_ = s.search.Delete(ctx, op.VideoID)
	}

	if s.archiveRepo != nil {
		retentionUntil := time.Now().AddDate(0, 0, complianceRetentionDays).Unix()
		_ = s.archiveRepo.Save(ctx, &model.VideoArchive{
			VideoID:        op.VideoID,
			VideoSnapshot:  *v,
			LikeCount:      cleared.LikeCount,
			CommentCount:   cleared.CommentCount,
			FavoriteCount:  cleared.FavoriteCount,
			DeletedAt:      now,
			DeletedBy:      op.OperatorID,
			DeleteReason:   op.Reason,
			RetentionUntil: retentionUntil,
		})
	}

	detail := fmt.Sprintf("reason=%s; cleared_likes=%d; cleared_comments=%d; cleared_favorites=%d",
		op.Reason, cleared.LikeCount, cleared.CommentCount, cleared.FavoriteCount)
	_ = s.writeAudit(ctx, "video.soft_delete", op, "video", op.VideoID, detail)
	return deleted, nil
}

func (s *VideoService) RestoreVideo(ctx context.Context, op AdminOp) (*model.Video, error) {
	v, err := s.repo.GetByIDIncludingDeleted(ctx, op.VideoID)
	if err != nil {
		return nil, err
	}
	if v.DeletedAt == 0 {
		return nil, fmt.Errorf("video not in recycle bin")
	}
	if time.Now().Unix() > v.PurgeAt {
		return nil, fmt.Errorf("recycle bin expired, cannot restore")
	}
	restored, err := s.repo.Restore(ctx, op.VideoID)
	if err != nil {
		return nil, err
	}
	s.indexVideo(ctx, restored)
	_ = s.writeAudit(ctx, "video.restore", op, "video", op.VideoID, "")
	return restored, nil
}

func (s *VideoService) PermanentDeleteVideo(ctx context.Context, op AdminOp) error {
	v, err := s.repo.GetByIDIncludingDeleted(ctx, op.VideoID)
	if err != nil {
		return err
	}
	if v.DeletedAt == 0 {
		return fmt.Errorf("video must be soft-deleted first")
	}
	if err := s.repo.PermanentDelete(ctx, op.VideoID); err != nil {
		return err
	}
	_ = s.writeAudit(ctx, "video.permanent_delete", op, "video", op.VideoID, "archive retained for compliance")
	return nil
}

func (s *VideoService) writeAudit(ctx context.Context, action string, op AdminOp, targetType, targetID, detail string) error {
	if s.auditRepo == nil {
		return nil
	}
	return s.auditRepo.Create(ctx, &model.AuditLog{
		Action:        action,
		ActorID:       op.OperatorID,
		ActorUsername: op.OperatorUsername,
		TargetType:    targetType,
		TargetID:      targetID,
		IP:            op.IP,
		UserAgent:     op.UserAgent,
		Detail:        detail,
	})
}
