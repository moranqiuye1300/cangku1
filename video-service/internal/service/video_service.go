package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"short-video-platform/pkg/auth"
	"short-video-platform/video-service/internal/kafka"
	"short-video-platform/video-service/internal/model"
	"short-video-platform/video-service/internal/repository"
	"short-video-platform/video-service/internal/search"
	"short-video-platform/video-service/internal/tagging"
)

type VideoService struct {
	repo         repository.VideoRepository
	barrageRepo  *repository.BarrageRepository
	interactRepo *repository.InteractionRepository
	auditRepo    *repository.AuditRepository
	archiveRepo  *repository.ArchiveRepository
	prefRepo     *repository.PreferenceRepository
	producer     *kafka.Producer
	search       *search.Client
}

func NewVideoService(repo repository.VideoRepository, barrageRepo *repository.BarrageRepository, interactRepo *repository.InteractionRepository, auditRepo *repository.AuditRepository, archiveRepo *repository.ArchiveRepository, prefRepo *repository.PreferenceRepository, producer *kafka.Producer, searchClient *search.Client) *VideoService {
	return &VideoService{repo: repo, barrageRepo: barrageRepo, interactRepo: interactRepo, auditRepo: auditRepo, archiveRepo: archiveRepo, prefRepo: prefRepo, producer: producer, search: searchClient}
}

func (s *VideoService) ListBarrages(ctx context.Context, videoID string, page, pageSize int) ([]model.Barrage, int, error) {
	if _, err := s.requirePublished(ctx, videoID); err != nil {
		return nil, 0, err
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 100
	}
	return s.barrageRepo.ListByVideo(ctx, videoID, page, pageSize)
}

func (s *VideoService) PostBarrage(ctx context.Context, videoID, userID, username, content string, timeMs int32) (*model.Barrage, error) {
	if _, err := s.requirePublished(ctx, videoID); err != nil {
		return nil, err
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, fmt.Errorf("content required")
	}
	b := &model.Barrage{
		VideoID:  videoID,
		UserID:   userID,
		Username: username,
		Content:  content,
		TimeMs:   timeMs,
	}
	if err := s.barrageRepo.Create(ctx, b); err != nil {
		return nil, err
	}
	return b, nil
}

func (s *VideoService) GetEngagement(ctx context.Context, videoID, viewerUserID string) (*model.Engagement, error) {
	if _, err := s.requirePublished(ctx, videoID); err != nil {
		return nil, err
	}
	stats, err := s.interactRepo.GetStats(ctx, videoID)
	if err != nil {
		return nil, err
	}
	liked, err := s.interactRepo.HasLike(ctx, videoID, viewerUserID)
	if err != nil {
		return nil, err
	}
	favorited, err := s.interactRepo.HasFavorite(ctx, videoID, viewerUserID)
	if err != nil {
		return nil, err
	}
	return &model.Engagement{
		VideoID:       videoID,
		LikeCount:     stats.LikeCount,
		CommentCount:  stats.CommentCount,
		FavoriteCount: stats.FavoriteCount,
		Liked:         liked,
		Favorited:     favorited,
	}, nil
}

func (s *VideoService) ToggleLike(ctx context.Context, videoID, userID string) (bool, int64, error) {
	v, err := s.requirePublished(ctx, videoID)
	if err != nil {
		return false, 0, err
	}
	liked, count, err := s.interactRepo.ToggleLike(ctx, videoID, userID)
	if err != nil {
		return false, 0, err
	}
	if liked {
		s.trackPreference(ctx, userID, v.Tags, 2.0)
	} else {
		s.trackPreference(ctx, userID, v.Tags, -2.0)
	}
	return liked, count, nil
}

func (s *VideoService) ToggleFavorite(ctx context.Context, videoID, userID string) (bool, int64, error) {
	v, err := s.requirePublished(ctx, videoID)
	if err != nil {
		return false, 0, err
	}
	favorited, count, err := s.interactRepo.ToggleFavorite(ctx, videoID, userID)
	if err != nil {
		return false, 0, err
	}
	if favorited {
		s.trackPreference(ctx, userID, v.Tags, 3.0)
	} else {
		s.trackPreference(ctx, userID, v.Tags, -3.0)
	}
	return favorited, count, nil
}

func (s *VideoService) ListComments(ctx context.Context, videoID string, page, pageSize int) ([]model.Comment, int, error) {
	if _, err := s.requirePublished(ctx, videoID); err != nil {
		return nil, 0, err
	}
	return s.interactRepo.ListComments(ctx, videoID, page, pageSize)
}

func (s *VideoService) PostComment(ctx context.Context, videoID, userID, username, content string) (*model.Comment, error) {
	if _, err := s.requirePublished(ctx, videoID); err != nil {
		return nil, err
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return nil, fmt.Errorf("content required")
	}
	c := &model.Comment{
		VideoID:  videoID,
		UserID:   userID,
		Username: username,
		Content:  content,
	}
	if err := s.interactRepo.CreateComment(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

func (s *VideoService) ListLikedByUser(ctx context.Context, userID string, page, pageSize int) ([]model.Video, int, error) {
	ids, total, err := s.interactRepo.ListLikedVideoIDs(ctx, userID, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	return s.videosByIDs(ctx, ids), total, nil
}

func (s *VideoService) ListFavoritesByUser(ctx context.Context, userID string, page, pageSize int) ([]model.Video, int, error) {
	ids, total, err := s.interactRepo.ListFavoriteVideoIDs(ctx, userID, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	return s.videosByIDs(ctx, ids), total, nil
}

func (s *VideoService) videosByIDs(ctx context.Context, ids []string) []model.Video {
	out := make([]model.Video, 0, len(ids))
	for _, id := range ids {
		v, err := s.repo.GetByID(ctx, id)
		if err != nil {
			if err == repository.ErrNotFound {
				continue
			}
			continue
		}
		if v.Status != model.StatusReady {
			continue
		}
		out = append(out, *v)
	}
	return out
}

func (s *VideoService) ReindexSearch(ctx context.Context) error {
	if s.search == nil {
		return nil
	}
	list, err := s.repo.ListAll(ctx, 500)
	if err != nil {
		return err
	}
	return s.search.ReindexAll(ctx, publishedOnly(list))
}

func publishedOnly(list []model.Video) []model.Video {
	out := make([]model.Video, 0, len(list))
	for i := range list {
		if list[i].Status == model.StatusReady {
			out = append(out, list[i])
		}
	}
	return out
}

func (s *VideoService) Search(ctx context.Context, keyword string, page, pageSize int) ([]model.Video, int, error) {
	if s.search == nil {
		return nil, 0, fmt.Errorf("search unavailable")
	}
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return nil, 0, fmt.Errorf("keyword required")
	}
	ids, total, err := s.search.Search(ctx, keyword, page, pageSize)
	if err != nil {
		return nil, 0, err
	}
	out := make([]model.Video, 0, len(ids))
	for _, id := range ids {
		v, err := s.repo.GetByID(ctx, id)
		if err != nil {
			if err == repository.ErrNotFound {
				continue
			}
			return nil, 0, err
		}
		if v.Status != model.StatusReady {
			continue
		}
		out = append(out, *v)
	}
	return out, total, nil
}

func (s *VideoService) indexVideo(ctx context.Context, v *model.Video) {
	if s.search != nil && v != nil && v.Status == model.StatusReady {
		_ = s.search.Index(ctx, v)
	}
}

func (s *VideoService) List(ctx context.Context, page, pageSize int) ([]model.Video, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 10
	}
	return s.repo.List(ctx, page, pageSize)
}

func (s *VideoService) ListByUser(ctx context.Context, userID, viewerUserID string, page, pageSize int) ([]model.Video, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 10
	}
	if viewerUserID != "" && viewerUserID == userID {
		return s.repo.ListByUser(ctx, userID, page, pageSize)
	}
	return s.repo.ListByUserPublished(ctx, userID, page, pageSize)
}

func (s *VideoService) GetByID(ctx context.Context, id string) (*model.Video, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *VideoService) Create(ctx context.Context, userID, title, description, sourcePath string) (*model.Video, error) {
	if err := auth.ValidateSourcePathForUser(userID, sourcePath); err != nil {
		return nil, err
	}
	nextID, err := s.repo.NextVideoID(ctx)
	if err != nil {
		return nil, err
	}
	video := &model.Video{
		ID:          nextID,
		UserID:      userID,
		Title:       title,
		Description: description,
		CoverURL:    "",
		Status:      model.StatusPendingSourceReview,
		Duration:    0,
		CreatedAt:   time.Now().Unix(),
		PlayURLs:    map[string]string{},
		SourcePath:  sourcePath,
	}
	if err := s.repo.Create(ctx, video); err != nil {
		return nil, err
	}
	return video, nil
}

func (s *VideoService) UpdateTranscodeResult(ctx context.Context, videoID, status string, duration int32, coverURL string, playURLs map[string]string, errMsg string, incomingTags []string) (*model.Video, error) {
	current, err := s.repo.GetByIDIncludingDeleted(ctx, videoID)
	if err != nil {
		return nil, err
	}
	switch status {
	case model.StatusPendingFinalReview, model.StatusFailed:
		if current.Status != model.StatusTranscoding {
			return nil, fmt.Errorf("invalid status transition from %s to %s", current.Status, status)
		}
	case model.StatusReady:
		return nil, fmt.Errorf("cannot publish via transcode callback; awaiting final review")
	default:
		return nil, fmt.Errorf("invalid transcode status: %s", status)
	}

	tags := incomingTags
	if status == model.StatusPendingFinalReview || status == model.StatusReady {
		current, err := s.repo.GetByIDIncludingDeleted(ctx, videoID)
		if err == nil && current != nil {
			localTags := tagging.Extract(current.Title, current.Description)
			tags = tagging.MergeTags(incomingTags, localTags)
		}
	}
	v, err := s.repo.UpdateTranscodeResult(ctx, videoID, status, duration, coverURL, playURLs, errMsg, tags)
	if err != nil {
		return nil, err
	}
	if v.Status == model.StatusReady {
		s.indexVideo(ctx, v)
	}
	return v, nil
}

func SeedVideos(ctx context.Context, repo repository.VideoRepository) error {
	count, err := repo.Count(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	return repo.InsertMany(ctx, DefaultSeedVideos())
}

func BackfillTags(ctx context.Context, repo repository.VideoRepository, search *search.Client) error {
	list, err := repo.ListAll(ctx, 500)
	if err != nil {
		return err
	}
	for i := range list {
		if len(list[i].Tags) > 0 {
			continue
		}
		tags := tagging.Extract(list[i].Title, list[i].Description)
		if err := repo.UpdateTags(ctx, list[i].ID, tags); err != nil {
			continue
		}
		list[i].Tags = tags
		if search != nil {
			_ = search.Index(ctx, &list[i])
		}
	}
	return nil
}

func DefaultSeedVideos() []model.Video {
	seed := []model.Video{
		{
			ID: "v1", UserID: "u1", Title: "Go 并发入门", Description: "goroutine 与 channel 基础讲解",
			CoverURL: "https://picsum.photos/seed/v1/640/360", Status: model.StatusReady, Duration: 320, CreatedAt: 1717200000,
			PlayURLs: map[string]string{},
		},
		{
			ID: "v2", UserID: "u1", Title: "Gin 框架快速上手", Description: "10 分钟搭建 REST API",
			CoverURL: "https://picsum.photos/seed/v2/640/360", Status: model.StatusReady, Duration: 610, CreatedAt: 1717100000,
			PlayURLs: map[string]string{},
		},
		{
			ID: "v3", UserID: "u2", Title: "gRPC 微服务实践", Description: "Proto 定义与服务拆分",
			CoverURL: "https://picsum.photos/seed/v3/640/360", Status: model.StatusReady, Duration: 480, CreatedAt: 1717000000,
			PlayURLs: map[string]string{},
		},
	}
	for i := range seed {
		seed[i].Tags = tagging.Extract(seed[i].Title, seed[i].Description)
	}
	return seed
}

// indexVideoToAI sends the video to ai-service for Chroma embedding (RAG on Feed).
// Best-effort: errors are logged but do not block publishing.
func (s *VideoService) indexVideoToAI(ctx context.Context, v *model.Video) error {
	base := strings.TrimRight(os.Getenv("AI_SERVICE_URL"), "/")
	if base == "" {
		base = "http://ai-service:8090"
	}

	payload := map[string]any{
		"video_id":    v.ID,
		"title":       v.Title,
		"description": v.Description,
		"tags":        v.Tags,
	}
	body, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/videos/index", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("ai index failed: status %d", resp.StatusCode)
	}
	return nil
}
