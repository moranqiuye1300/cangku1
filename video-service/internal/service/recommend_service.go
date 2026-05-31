package service

import (
	"context"
	"math/rand"
	"sort"
	"time"

	"short-video-platform/video-service/internal/model"
)

type scoredVideo struct {
	video model.Video
	score float64
}

func (s *VideoService) RecommendFeed(ctx context.Context, viewerUserID string, page, pageSize int) ([]model.Video, int, bool, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 10
	}

	personalized := false
	candidates, err := s.repo.ListReadyActive(ctx, 500)
	if err != nil {
		return nil, 0, false, err
	}
	if len(candidates) == 0 {
		list, total, err := s.List(ctx, page, pageSize)
		return list, total, false, err
	}

	var ranked []model.Video
	if viewerUserID != "" && s.prefRepo != nil {
		weights, err := s.prefRepo.GetWeights(ctx, viewerUserID)
		if err != nil {
			return nil, 0, false, err
		}
		if len(weights) > 0 {
			personalized = true
			ranked = rankByPreference(candidates, weights)
		}
	}
	if !personalized {
		ranked = candidates
	}

	total := len(ranked)
	start := (page - 1) * pageSize
	if start >= total {
		return []model.Video{}, total, personalized, nil
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	out := make([]model.Video, end-start)
	copy(out, ranked[start:end])
	return out, total, personalized, nil
}

func rankByPreference(videos []model.Video, weights map[string]float64) []model.Video {
	now := time.Now().Unix()
	items := make([]scoredVideo, 0, len(videos))
	for _, v := range videos {
		score := rand.Float64() * 0.3
		for _, tag := range v.Tags {
			score += weights[tag]
		}
		ageDays := float64(now-v.CreatedAt) / 86400
		if ageDays < 7 {
			score += 1.5 - ageDays*0.1
		}
		items = append(items, scoredVideo{video: v, score: score})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].score == items[j].score {
			return items[i].video.CreatedAt > items[j].video.CreatedAt
		}
		return items[i].score > items[j].score
	})
	out := make([]model.Video, 0, len(items))
	for _, it := range items {
		out = append(out, it.video)
	}
	return out
}

func (s *VideoService) trackPreference(ctx context.Context, userID string, tags []string, delta float64) {
	if s.prefRepo == nil || userID == "" || len(tags) == 0 {
		return
	}
	_ = s.prefRepo.AddTagWeights(ctx, userID, tags, delta)
}
