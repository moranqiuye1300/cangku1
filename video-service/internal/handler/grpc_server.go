package handler

import (
	"context"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"short-video-platform/gen/videopb"
	"short-video-platform/pkg/auth"
	"short-video-platform/video-service/internal/model"
	"short-video-platform/video-service/internal/repository"
	"short-video-platform/video-service/internal/service"
)

type VideoGRPCServer struct {
	videopb.UnimplementedVideoServiceServer
	svc *service.VideoService
}

func NewVideoGRPCServer(svc *service.VideoService) *VideoGRPCServer {
	return &VideoGRPCServer{svc: svc}
}

func (s *VideoGRPCServer) GetVideoList(ctx context.Context, req *videopb.GetVideoListRequest) (*videopb.GetVideoListResponse, error) {
	list, total, err := s.svc.List(ctx, int(req.GetPage()), int(req.GetPageSize()))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &videopb.GetVideoListResponse{Videos: toProtoList(list), Total: int32(total)}, nil
}

func (s *VideoGRPCServer) GetRecommendedFeed(ctx context.Context, req *videopb.GetRecommendedFeedRequest) (*videopb.GetRecommendedFeedResponse, error) {
	list, total, personalized, err := s.svc.RecommendFeed(ctx, req.GetViewerUserId(), int(req.GetPage()), int(req.GetPageSize()))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &videopb.GetRecommendedFeedResponse{
		Videos:        toProtoList(list),
		Total:         int32(total),
		Personalized:  personalized,
	}, nil
}

func (s *VideoGRPCServer) GetVideoInfo(ctx context.Context, req *videopb.GetVideoInfoRequest) (*videopb.GetVideoInfoResponse, error) {
	v, err := s.svc.GetPublicByID(ctx, req.GetVideoId(), auth.UserIDFromContext(ctx))
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, status.Error(codes.NotFound, "video not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &videopb.GetVideoInfoResponse{Video: toProto(v)}, nil
}

func (s *VideoGRPCServer) ListVideosByUser(ctx context.Context, req *videopb.ListVideosByUserRequest) (*videopb.ListVideosByUserResponse, error) {
	list, total, err := s.svc.ListByUser(ctx, req.GetUserId(), auth.UserIDFromContext(ctx), int(req.GetPage()), int(req.GetPageSize()))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &videopb.ListVideosByUserResponse{Videos: toProtoList(list), Total: int32(total)}, nil
}

func (s *VideoGRPCServer) CreateVideo(ctx context.Context, req *videopb.CreateVideoRequest) (*videopb.CreateVideoResponse, error) {
	userID := auth.UserIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.Unauthenticated, "login required")
	}
	if req.GetUserId() != "" && req.GetUserId() != userID {
		return nil, status.Error(codes.PermissionDenied, "user_id mismatch")
	}
	if req.GetTitle() == "" || req.GetSourcePath() == "" {
		return nil, status.Error(codes.InvalidArgument, "title and source_path required")
	}
	v, err := s.svc.Create(ctx, userID, req.GetTitle(), req.GetDescription(), req.GetSourcePath())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &videopb.CreateVideoResponse{Video: toProto(v)}, nil
}

func (s *VideoGRPCServer) SearchVideos(ctx context.Context, req *videopb.SearchVideosRequest) (*videopb.SearchVideosResponse, error) {
	keyword := strings.TrimSpace(req.GetKeyword())
	if keyword == "" {
		return nil, status.Error(codes.InvalidArgument, "keyword required")
	}
	list, total, err := s.svc.Search(ctx, keyword, int(req.GetPage()), int(req.GetPageSize()))
	if err != nil {
		if strings.Contains(err.Error(), "unavailable") {
			return nil, status.Error(codes.Unavailable, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &videopb.SearchVideosResponse{Videos: toProtoList(list), Total: int32(total)}, nil
}

func (s *VideoGRPCServer) ListBarrages(ctx context.Context, req *videopb.ListBarragesRequest) (*videopb.ListBarragesResponse, error) {
	if req.GetVideoId() == "" {
		return nil, status.Error(codes.InvalidArgument, "video_id required")
	}
	list, total, err := s.svc.ListBarrages(ctx, req.GetVideoId(), int(req.GetPage()), int(req.GetPageSize()))
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, status.Error(codes.NotFound, "video not found")
		}
		if strings.Contains(err.Error(), "not published") {
			return nil, status.Error(codes.FailedPrecondition, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &videopb.ListBarragesResponse{Barrages: toBarrageProtoList(list), Total: int32(total)}, nil
}

func (s *VideoGRPCServer) PostBarrage(ctx context.Context, req *videopb.PostBarrageRequest) (*videopb.PostBarrageResponse, error) {
	userID := auth.UserIDFromContext(ctx)
	username := auth.UsernameFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.Unauthenticated, "login required")
	}
	if req.GetVideoId() == "" || strings.TrimSpace(req.GetContent()) == "" {
		return nil, status.Error(codes.InvalidArgument, "video_id and content required")
	}
	if username == "" {
		username = req.GetUsername()
	}
	b, err := s.svc.PostBarrage(ctx, req.GetVideoId(), userID, username, req.GetContent(), req.GetTimeMs())
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, status.Error(codes.NotFound, "video not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &videopb.PostBarrageResponse{Barrage: toBarrageProto(b)}, nil
}

func (s *VideoGRPCServer) GetVideoEngagement(ctx context.Context, req *videopb.GetVideoEngagementRequest) (*videopb.GetVideoEngagementResponse, error) {
	if req.GetVideoId() == "" {
		return nil, status.Error(codes.InvalidArgument, "video_id required")
	}
	e, err := s.svc.GetEngagement(ctx, req.GetVideoId(), req.GetViewerUserId())
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, status.Error(codes.NotFound, "video not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &videopb.GetVideoEngagementResponse{Engagement: toEngagementProto(e)}, nil
}

func (s *VideoGRPCServer) ToggleLike(ctx context.Context, req *videopb.ToggleLikeRequest) (*videopb.ToggleLikeResponse, error) {
	userID := auth.UserIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.Unauthenticated, "login required")
	}
	if req.GetVideoId() == "" {
		return nil, status.Error(codes.InvalidArgument, "video_id required")
	}
	liked, count, err := s.svc.ToggleLike(ctx, req.GetVideoId(), userID)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, status.Error(codes.NotFound, "video not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &videopb.ToggleLikeResponse{Liked: liked, LikeCount: count}, nil
}

func (s *VideoGRPCServer) ToggleFavorite(ctx context.Context, req *videopb.ToggleFavoriteRequest) (*videopb.ToggleFavoriteResponse, error) {
	userID := auth.UserIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.Unauthenticated, "login required")
	}
	if req.GetVideoId() == "" {
		return nil, status.Error(codes.InvalidArgument, "video_id required")
	}
	favorited, count, err := s.svc.ToggleFavorite(ctx, req.GetVideoId(), userID)
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, status.Error(codes.NotFound, "video not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &videopb.ToggleFavoriteResponse{Favorited: favorited, FavoriteCount: count}, nil
}

func (s *VideoGRPCServer) ListComments(ctx context.Context, req *videopb.ListCommentsRequest) (*videopb.ListCommentsResponse, error) {
	if req.GetVideoId() == "" {
		return nil, status.Error(codes.InvalidArgument, "video_id required")
	}
	list, total, err := s.svc.ListComments(ctx, req.GetVideoId(), int(req.GetPage()), int(req.GetPageSize()))
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, status.Error(codes.NotFound, "video not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &videopb.ListCommentsResponse{Comments: toCommentProtoList(list), Total: int32(total)}, nil
}

func (s *VideoGRPCServer) PostComment(ctx context.Context, req *videopb.PostCommentRequest) (*videopb.PostCommentResponse, error) {
	userID := auth.UserIDFromContext(ctx)
	username := auth.UsernameFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.Unauthenticated, "login required")
	}
	if req.GetVideoId() == "" || strings.TrimSpace(req.GetContent()) == "" {
		return nil, status.Error(codes.InvalidArgument, "video_id and content required")
	}
	if username == "" {
		username = req.GetUsername()
	}
	c, err := s.svc.PostComment(ctx, req.GetVideoId(), userID, username, req.GetContent())
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, status.Error(codes.NotFound, "video not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &videopb.PostCommentResponse{Comment: toCommentProto(c)}, nil
}

func (s *VideoGRPCServer) ListUserLikedVideos(ctx context.Context, req *videopb.ListVideosByUserRequest) (*videopb.ListVideosByUserResponse, error) {
	userID := auth.UserIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.Unauthenticated, "login required")
	}
	if req.GetUserId() == "" || req.GetUserId() != userID {
		return nil, status.Error(codes.PermissionDenied, "can only view your own liked videos")
	}
	list, total, err := s.svc.ListLikedByUser(ctx, req.GetUserId(), int(req.GetPage()), int(req.GetPageSize()))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &videopb.ListVideosByUserResponse{Videos: toProtoList(list), Total: int32(total)}, nil
}

func (s *VideoGRPCServer) ListUserFavoriteVideos(ctx context.Context, req *videopb.ListVideosByUserRequest) (*videopb.ListVideosByUserResponse, error) {
	userID := auth.UserIDFromContext(ctx)
	if userID == "" {
		return nil, status.Error(codes.Unauthenticated, "login required")
	}
	if req.GetUserId() == "" || req.GetUserId() != userID {
		return nil, status.Error(codes.PermissionDenied, "can only view your own favorite videos")
	}
	list, total, err := s.svc.ListFavoritesByUser(ctx, req.GetUserId(), int(req.GetPage()), int(req.GetPageSize()))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &videopb.ListVideosByUserResponse{Videos: toProtoList(list), Total: int32(total)}, nil
}

func (s *VideoGRPCServer) UpdateTranscodeResult(ctx context.Context, req *videopb.UpdateTranscodeResultRequest) (*videopb.UpdateTranscodeResultResponse, error) {
	if req.GetVideoId() == "" || req.GetStatus() == "" {
		return nil, status.Error(codes.InvalidArgument, "video_id and status required")
	}
	v, err := s.svc.UpdateTranscodeResult(ctx, req.GetVideoId(), req.GetStatus(), req.GetDuration(), req.GetCoverUrl(), req.GetPlayUrls(), req.GetErrorMessage(), req.GetTags())
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, status.Error(codes.NotFound, "video not found")
		}
		if strings.Contains(err.Error(), "invalid status") || strings.Contains(err.Error(), "cannot publish") {
			return nil, status.Error(codes.FailedPrecondition, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &videopb.UpdateTranscodeResultResponse{Video: toProto(v)}, nil
}

func (s *VideoGRPCServer) AdminListVideos(ctx context.Context, req *videopb.AdminListVideosRequest) (*videopb.AdminListVideosResponse, error) {
	list, total, err := s.svc.AdminListVideos(ctx, int(req.GetPage()), int(req.GetPageSize()), req.GetIncludeDeleted())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &videopb.AdminListVideosResponse{Videos: toAdminProtoList(list), Total: int32(total)}, nil
}

func (s *VideoGRPCServer) AdminSoftDeleteVideo(ctx context.Context, req *videopb.AdminSoftDeleteVideoRequest) (*videopb.AdminSoftDeleteVideoResponse, error) {
	if req.GetVideoId() == "" {
		return nil, status.Error(codes.InvalidArgument, "video_id required")
	}
	v, err := s.svc.SoftDeleteVideo(ctx, adminOpFromContext(ctx, req.GetVideoId(), req.GetReason(), req.GetIp(), req.GetUserAgent()))
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, status.Error(codes.NotFound, "video not found")
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &videopb.AdminSoftDeleteVideoResponse{Video: toAdminProto(v)}, nil
}

func (s *VideoGRPCServer) AdminRestoreVideo(ctx context.Context, req *videopb.AdminRestoreVideoRequest) (*videopb.AdminRestoreVideoResponse, error) {
	if req.GetVideoId() == "" {
		return nil, status.Error(codes.InvalidArgument, "video_id required")
	}
	v, err := s.svc.RestoreVideo(ctx, adminOpFromContext(ctx, req.GetVideoId(), "", req.GetIp(), req.GetUserAgent()))
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, status.Error(codes.NotFound, "video not found")
		}
		if strings.Contains(err.Error(), "recycle bin") || strings.Contains(err.Error(), "not in recycle") {
			return nil, status.Error(codes.FailedPrecondition, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &videopb.AdminRestoreVideoResponse{Video: toProto(v)}, nil
}

func (s *VideoGRPCServer) AdminPermanentDeleteVideo(ctx context.Context, req *videopb.AdminPermanentDeleteVideoRequest) (*videopb.AdminPermanentDeleteVideoResponse, error) {
	if req.GetVideoId() == "" {
		return nil, status.Error(codes.InvalidArgument, "video_id required")
	}
	err := s.svc.PermanentDeleteVideo(ctx, adminOpFromContext(ctx, req.GetVideoId(), "", req.GetIp(), req.GetUserAgent()))
	if err != nil {
		if err == repository.ErrNotFound {
			return nil, status.Error(codes.NotFound, "video not found")
		}
		if strings.Contains(err.Error(), "soft-deleted") {
			return nil, status.Error(codes.FailedPrecondition, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &videopb.AdminPermanentDeleteVideoResponse{}, nil
}

func (s *VideoGRPCServer) ListRecycleBin(ctx context.Context, req *videopb.ListRecycleBinRequest) (*videopb.ListRecycleBinResponse, error) {
	list, total, err := s.svc.ListRecycleBin(ctx, int(req.GetPage()), int(req.GetPageSize()))
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &videopb.ListRecycleBinResponse{Videos: toAdminProtoList(list), Total: int32(total)}, nil
}

func (s *VideoGRPCServer) ListAuditLogs(ctx context.Context, req *videopb.ListAuditLogsRequest) (*videopb.ListAuditLogsResponse, error) {
	list, total, err := s.svc.ListAuditLogs(ctx, int(req.GetPage()), int(req.GetPageSize()), req.GetTargetType())
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &videopb.ListAuditLogsResponse{Logs: toAuditProtoList(list), Total: int32(total)}, nil
}

func (s *VideoGRPCServer) ReviewerListPending(ctx context.Context, req *videopb.ReviewerListPendingRequest) (*videopb.ReviewerListPendingResponse, error) {
	list, total, err := s.svc.ListPendingReview(ctx, req.GetStage(), int(req.GetPage()), int(req.GetPageSize()))
	if err != nil {
		if strings.Contains(err.Error(), "invalid stage") {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &videopb.ReviewerListPendingResponse{Videos: toAdminProtoList(list), Total: int32(total)}, nil
}

func (s *VideoGRPCServer) ReviewerApproveSource(ctx context.Context, req *videopb.ReviewerReviewActionRequest) (*videopb.ReviewerReviewActionResponse, error) {
	v, err := s.svc.ApproveSourceReview(ctx, reviewOpFromProto(ctx, req))
	if err != nil {
		return nil, reviewActionError(err)
	}
	return &videopb.ReviewerReviewActionResponse{Video: toAdminProto(v)}, nil
}

func (s *VideoGRPCServer) ReviewerRejectSource(ctx context.Context, req *videopb.ReviewerReviewActionRequest) (*videopb.ReviewerReviewActionResponse, error) {
	v, err := s.svc.RejectSourceReview(ctx, reviewOpFromProto(ctx, req))
	if err != nil {
		return nil, reviewActionError(err)
	}
	return &videopb.ReviewerReviewActionResponse{Video: toAdminProto(v)}, nil
}

func (s *VideoGRPCServer) ReviewerApprovePublish(ctx context.Context, req *videopb.ReviewerReviewActionRequest) (*videopb.ReviewerReviewActionResponse, error) {
	v, err := s.svc.ApprovePublishReview(ctx, reviewOpFromProto(ctx, req))
	if err != nil {
		return nil, reviewActionError(err)
	}
	return &videopb.ReviewerReviewActionResponse{Video: toAdminProto(v)}, nil
}

func (s *VideoGRPCServer) ReviewerRejectPublish(ctx context.Context, req *videopb.ReviewerReviewActionRequest) (*videopb.ReviewerReviewActionResponse, error) {
	v, err := s.svc.RejectPublishReview(ctx, reviewOpFromProto(ctx, req))
	if err != nil {
		return nil, reviewActionError(err)
	}
	return &videopb.ReviewerReviewActionResponse{Video: toAdminProto(v)}, nil
}

func adminOpFromContext(ctx context.Context, videoID, reason, ip, userAgent string) service.AdminOp {
	return service.AdminOp{
		VideoID:          videoID,
		OperatorID:       auth.UserIDFromContext(ctx),
		OperatorUsername: auth.UsernameFromContext(ctx),
		Reason:           reason,
		IP:               ip,
		UserAgent:        userAgent,
	}
}

func reviewOpFromProto(ctx context.Context, req *videopb.ReviewerReviewActionRequest) service.AdminOp {
	return adminOpFromContext(ctx, req.GetVideoId(), req.GetReason(), req.GetIp(), req.GetUserAgent())
}

func reviewActionError(err error) error {
	if err == repository.ErrNotFound {
		return status.Error(codes.NotFound, "video not found")
	}
	if err == repository.ErrStatusConflict || strings.Contains(err.Error(), "not awaiting") || strings.Contains(err.Error(), "invalid stage") || strings.Contains(err.Error(), "invalid status") || strings.Contains(err.Error(), "invalid source") {
		return status.Error(codes.FailedPrecondition, err.Error())
	}
	return status.Error(codes.Internal, err.Error())
}

func toProtoList(list []model.Video) []*videopb.Video {
	out := make([]*videopb.Video, 0, len(list))
	for i := range list {
		out = append(out, toProto(&list[i]))
	}
	return out
}

func toProto(v *model.Video) *videopb.Video {
	return &videopb.Video{
		Id:          v.ID,
		UserId:      v.UserID,
		Title:       v.Title,
		Description: v.Description,
		CoverUrl:    v.CoverURL,
		Status:      v.Status,
		Duration:    v.Duration,
		CreatedAt:   v.CreatedAt,
		PlayUrls:    v.PlayURLs,
		Tags:        v.Tags,
	}
}

func toBarrageProtoList(list []model.Barrage) []*videopb.Barrage {
	out := make([]*videopb.Barrage, 0, len(list))
	for i := range list {
		out = append(out, toBarrageProto(&list[i]))
	}
	return out
}

func toBarrageProto(b *model.Barrage) *videopb.Barrage {
	return &videopb.Barrage{
		Id:        b.ID,
		VideoId:   b.VideoID,
		UserId:    b.UserID,
		Username:  b.Username,
		Content:   b.Content,
		TimeMs:    b.TimeMs,
		CreatedAt: b.CreatedAt,
	}
}

func toEngagementProto(e *model.Engagement) *videopb.VideoEngagement {
	if e == nil {
		return nil
	}
	return &videopb.VideoEngagement{
		VideoId:       e.VideoID,
		LikeCount:     e.LikeCount,
		CommentCount:  e.CommentCount,
		FavoriteCount: e.FavoriteCount,
		Liked:         e.Liked,
		Favorited:     e.Favorited,
	}
}

func toCommentProtoList(list []model.Comment) []*videopb.Comment {
	out := make([]*videopb.Comment, 0, len(list))
	for i := range list {
		out = append(out, toCommentProto(&list[i]))
	}
	return out
}

func toCommentProto(c *model.Comment) *videopb.Comment {
	return &videopb.Comment{
		Id:        c.ID,
		VideoId:   c.VideoID,
		UserId:    c.UserID,
		Username:  c.Username,
		Content:   c.Content,
		CreatedAt: c.CreatedAt,
	}
}

func toAdminProto(v *model.Video) *videopb.AdminVideo {
	if v == nil {
		return nil
	}
	return &videopb.AdminVideo{
		Video:        toProto(v),
		DeletedAt:    v.DeletedAt,
		DeletedBy:    v.DeletedBy,
		DeleteReason: v.DeleteReason,
		PurgeAt:      v.PurgeAt,
		SourcePath:   v.SourcePath,
	}
}

func toAdminProtoList(list []model.Video) []*videopb.AdminVideo {
	out := make([]*videopb.AdminVideo, 0, len(list))
	for i := range list {
		out = append(out, toAdminProto(&list[i]))
	}
	return out
}

func toAuditProtoList(list []model.AuditLog) []*videopb.AuditLog {
	out := make([]*videopb.AuditLog, 0, len(list))
	for i := range list {
		out = append(out, &videopb.AuditLog{
			Id:            list[i].ID,
			Action:        list[i].Action,
			ActorId:       list[i].ActorID,
			ActorUsername: list[i].ActorUsername,
			TargetType:    list[i].TargetType,
			TargetId:      list[i].TargetID,
			Ip:            list[i].IP,
			UserAgent:     list[i].UserAgent,
			Detail:        list[i].Detail,
			CreatedAt:     list[i].CreatedAt,
		})
	}
	return out
}
