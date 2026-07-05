package service

import (
	"context"
	"fmt"

	"short-video-platform/pkg/events"
	"short-video-platform/video-service/internal/model"
	"short-video-platform/video-service/internal/repository"
)

func (s *VideoService) ListPendingReview(ctx context.Context, stage string, page, pageSize int) ([]model.Video, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	status := model.StatusPendingSourceReview
	switch stage {
	case "source":
		status = model.StatusPendingSourceReview
	case "final":
		status = model.StatusPendingFinalReview
	default:
		return nil, 0, fmt.Errorf("invalid stage")
	}
	return s.repo.ListByStatus(ctx, status, page, pageSize)
}

func (s *VideoService) ApproveSourceReview(ctx context.Context, op AdminOp) (*model.Video, error) {
	v, err := s.repo.GetByID(ctx, op.VideoID)
	if err != nil {
		return nil, err
	}
	if v.Status == model.StatusTranscoding {
		return v, nil
	}
	if v.Status != model.StatusPendingSourceReview {
		return nil, fmt.Errorf("video not awaiting source review")
	}
	if s.producer == nil {
		return nil, fmt.Errorf("transcode queue unavailable")
	}
	if err := s.repo.UpdateStatusIf(ctx, v.ID, model.StatusPendingSourceReview, model.StatusTranscoding); err != nil {
		if err == repository.ErrStatusConflict {
			latest, getErr := s.repo.GetByID(ctx, v.ID)
			if getErr == nil && latest.Status == model.StatusTranscoding {
				return latest, nil
			}
			return nil, fmt.Errorf("video not awaiting source review")
		}
		return nil, err
	}
	if err := s.producer.PublishTranscode(events.TranscodeTask{
		VideoID:     v.ID,
		UserID:      v.UserID,
		SourcePath:  v.SourcePath,
		Title:       v.Title,
		Description: v.Description,
	}); err != nil {
		_ = s.repo.UpdateStatusIf(ctx, v.ID, model.StatusTranscoding, model.StatusPendingSourceReview)
		return nil, fmt.Errorf("publish transcode task: %w", err)
	}
	v.Status = model.StatusTranscoding
	_ = s.writeAudit(ctx, "video.source_approve", op, "video", op.VideoID, "")
	return v, nil
}

func (s *VideoService) RejectSourceReview(ctx context.Context, op AdminOp) (*model.Video, error) {
	v, err := s.repo.GetByID(ctx, op.VideoID)
	if err != nil {
		return nil, err
	}
	if v.Status != model.StatusPendingSourceReview {
		return nil, fmt.Errorf("video not awaiting source review")
	}
	deleted, err := s.SoftDeleteVideo(ctx, op)
	if err != nil {
		return nil, err
	}
	return deleted, nil
}

func (s *VideoService) ApprovePublishReview(ctx context.Context, op AdminOp) (*model.Video, error) {
	v, err := s.repo.GetByID(ctx, op.VideoID)
	if err != nil {
		return nil, err
	}
	if v.Status != model.StatusPendingFinalReview {
		return nil, fmt.Errorf("video not awaiting final review")
	}
	if err := s.repo.UpdateStatus(ctx, v.ID, model.StatusReady); err != nil {
		return nil, err
	}
	v, err = s.repo.GetByID(ctx, v.ID)
	if err != nil {
		return nil, err
	}
	s.indexVideo(ctx, v)
	_ = s.indexVideoToAI(ctx, v) // best-effort: embed into Chroma for RAG
	_ = s.writeAudit(ctx, "video.publish_approve", op, "video", op.VideoID, "")
	return v, nil
}

func (s *VideoService) RejectPublishReview(ctx context.Context, op AdminOp) (*model.Video, error) {
	v, err := s.repo.GetByID(ctx, op.VideoID)
	if err != nil {
		return nil, err
	}
	if v.Status != model.StatusPendingFinalReview {
		return nil, fmt.Errorf("video not awaiting final review")
	}
	deleted, err := s.SoftDeleteVideo(ctx, op)
	if err != nil {
		return nil, err
	}
	return deleted, nil
}

func (s *VideoService) requirePublished(ctx context.Context, videoID string) (*model.Video, error) {
	v, err := s.repo.GetByID(ctx, videoID)
	if err != nil {
		return nil, err
	}
	if v.Status != model.StatusReady {
		return nil, fmt.Errorf("video not published")
	}
	return v, nil
}

func (s *VideoService) GetPublicByID(ctx context.Context, id, viewerUserID string) (*model.Video, error) {
	v, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if v.Status == model.StatusReady {
		return v, nil
	}
	if viewerUserID != "" && viewerUserID == v.UserID {
		return v, nil
	}
	return nil, repository.ErrNotFound
}
