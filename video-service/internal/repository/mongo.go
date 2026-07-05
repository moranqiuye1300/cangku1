package repository

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"short-video-platform/video-service/internal/model"
)

type MongoRepository struct {
	col *mongo.Collection
}

func NewMongoRepository(db *mongo.Database) *MongoRepository {
	return &MongoRepository{col: db.Collection("videos")}
}

func activeFilter(extra bson.M) bson.M {
	filter := bson.M{
		"$or": []bson.M{
			{"deleted_at": bson.M{"$exists": false}},
			{"deleted_at": 0},
			{"deleted_at": nil},
		},
	}
	for k, v := range extra {
		filter[k] = v
	}
	return filter
}

func deletedFilter() bson.M {
	return bson.M{"deleted_at": bson.M{"$gt": 0}}
}

func (r *MongoRepository) List(ctx context.Context, page, pageSize int) ([]model.Video, int, error) {
	return r.ListPublished(ctx, page, pageSize)
}

func (r *MongoRepository) ListPublished(ctx context.Context, page, pageSize int) ([]model.Video, int, error) {
	filter := activeFilter(bson.M{"status": model.StatusReady})
	total64, err := r.col.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	total := int(total64)

	skip := int64((page - 1) * pageSize)
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetSkip(skip).
		SetLimit(int64(pageSize))

	cur, err := r.col.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cur.Close(ctx)

	return r.cursorToVideos(ctx, cur, total)
}

func (r *MongoRepository) ListByStatus(ctx context.Context, status string, page, pageSize int) ([]model.Video, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	filter := activeFilter(bson.M{"status": status})
	total64, err := r.col.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	total := int(total64)

	skip := int64((page - 1) * pageSize)
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetSkip(skip).
		SetLimit(int64(pageSize))

	cur, err := r.col.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cur.Close(ctx)

	return r.cursorToVideos(ctx, cur, total)
}

func (r *MongoRepository) ListByUser(ctx context.Context, userID string, page, pageSize int) ([]model.Video, int, error) {
	filter := activeFilter(bson.M{"user_id": userID})
	total64, err := r.col.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	total := int(total64)

	skip := int64((page - 1) * pageSize)
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetSkip(skip).
		SetLimit(int64(pageSize))

	cur, err := r.col.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cur.Close(ctx)

	return r.cursorToVideos(ctx, cur, total)
}

func (r *MongoRepository) ListByUserPublished(ctx context.Context, userID string, page, pageSize int) ([]model.Video, int, error) {
	filter := activeFilter(bson.M{"user_id": userID, "status": model.StatusReady})
	total64, err := r.col.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	total := int(total64)

	skip := int64((page - 1) * pageSize)
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetSkip(skip).
		SetLimit(int64(pageSize))

	cur, err := r.col.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cur.Close(ctx)

	return r.cursorToVideos(ctx, cur, total)
}

func (r *MongoRepository) GetByID(ctx context.Context, id string) (*model.Video, error) {
	var doc model.VideoDoc
	err := r.col.FindOne(ctx, activeFilter(bson.M{"video_id": id})).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return model.DocToVideo(&doc), nil
}

func (r *MongoRepository) GetByIDIncludingDeleted(ctx context.Context, id string) (*model.Video, error) {
	var doc model.VideoDoc
	err := r.col.FindOne(ctx, bson.M{"video_id": id}).Decode(&doc)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return model.DocToVideo(&doc), nil
}

func (r *MongoRepository) Count(ctx context.Context) (int64, error) {
	return r.col.CountDocuments(ctx, activeFilter(bson.M{}))
}

func parseVideoNum(id string) int {
	n, err := strconv.Atoi(strings.TrimPrefix(id, "v"))
	if err != nil || n < 0 {
		return 0
	}
	return n
}

// NextVideoID returns the next unused vN id based on all documents (including soft-deleted).
func (r *MongoRepository) NextVideoID(ctx context.Context) (string, error) {
	cur, err := r.col.Find(ctx, bson.M{}, options.Find().SetProjection(bson.M{"video_id": 1}))
	if err != nil {
		return "", err
	}
	defer cur.Close(ctx)

	maxNum := 0
	for cur.Next(ctx) {
		var doc struct {
			VideoID string `bson:"video_id"`
		}
		if err := cur.Decode(&doc); err != nil {
			return "", err
		}
		if n := parseVideoNum(doc.VideoID); n > maxNum {
			maxNum = n
		}
	}
	if err := cur.Err(); err != nil {
		return "", err
	}
	return fmt.Sprintf("v%d", maxNum+1), nil
}

func (r *MongoRepository) InsertMany(ctx context.Context, videos []model.Video) error {
	if len(videos) == 0 {
		return nil
	}
	docs := make([]any, 0, len(videos))
	for i := range videos {
		docs = append(docs, model.ToDoc(videos[i]))
	}
	_, err := r.col.InsertMany(ctx, docs)
	return err
}

func (r *MongoRepository) Create(ctx context.Context, video *model.Video) error {
	_, err := r.col.InsertOne(ctx, model.ToDoc(*video))
	return err
}

func (r *MongoRepository) UpdateStatus(ctx context.Context, videoID, status string) error {
	res, err := r.col.UpdateOne(ctx,
		bson.M{"video_id": videoID},
		bson.M{"$set": bson.M{"status": status}},
	)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *MongoRepository) UpdateStatusIf(ctx context.Context, videoID, fromStatus, toStatus string) error {
	res, err := r.col.UpdateOne(ctx,
		activeFilter(bson.M{"video_id": videoID, "status": fromStatus}),
		bson.M{"$set": bson.M{"status": toStatus}},
	)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return ErrStatusConflict
	}
	return nil
}

func (r *MongoRepository) UpdateTranscodeResult(ctx context.Context, videoID, status string, duration int32, coverURL string, playURLs map[string]string, errMsg string, tags []string) (*model.Video, error) {
	set := bson.M{
		"status": status,
	}
	if duration > 0 {
		set["duration"] = duration
	}
	if coverURL != "" {
		set["cover_url"] = coverURL
	}
	if len(playURLs) > 0 {
		set["play_urls"] = playURLs
	}
	if errMsg != "" {
		set["transcode_error"] = errMsg
	}
	if len(tags) > 0 {
		set["tags"] = tags
	}
	res, err := r.col.UpdateOne(ctx, bson.M{"video_id": videoID}, bson.M{"$set": set})
	if err != nil {
		return nil, err
	}
	if res.MatchedCount == 0 {
		return nil, ErrNotFound
	}
	return r.GetByID(ctx, videoID)
}

func (r *MongoRepository) ListAll(ctx context.Context, limit int) ([]model.Video, error) {
	if limit <= 0 {
		limit = 500
	}
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetLimit(int64(limit))
	cur, err := r.col.Find(ctx, activeFilter(bson.M{}), opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	list := make([]model.Video, 0)
	for cur.Next(ctx) {
		var doc model.VideoDoc
		if err := cur.Decode(&doc); err != nil {
			return nil, err
		}
		list = append(list, *model.DocToVideo(&doc))
	}
	return list, cur.Err()
}

func (r *MongoRepository) ListReadyActive(ctx context.Context, limit int) ([]model.Video, error) {
	if limit <= 0 {
		limit = 500
	}
	filter := activeFilter(bson.M{"status": model.StatusReady})
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetLimit(int64(limit))
	cur, err := r.col.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	list := make([]model.Video, 0)
	for cur.Next(ctx) {
		var doc model.VideoDoc
		if err := cur.Decode(&doc); err != nil {
			return nil, err
		}
		list = append(list, *model.DocToVideo(&doc))
	}
	return list, cur.Err()
}

func (r *MongoRepository) cursorToVideos(ctx context.Context, cur *mongo.Cursor, total int) ([]model.Video, int, error) {
	list := make([]model.Video, 0)
	for cur.Next(ctx) {
		var doc model.VideoDoc
		if err := cur.Decode(&doc); err != nil {
			return nil, 0, err
		}
		list = append(list, *model.DocToVideo(&doc))
	}
	if err := cur.Err(); err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (r *MongoRepository) AdminList(ctx context.Context, page, pageSize int, includeDeleted bool) ([]model.Video, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	filter := activeFilter(bson.M{})
	if includeDeleted {
		filter = bson.M{}
	}
	total64, err := r.col.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetSkip(int64((page - 1) * pageSize)).
		SetLimit(int64(pageSize))
	cur, err := r.col.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cur.Close(ctx)
	return r.cursorToVideos(ctx, cur, int(total64))
}

func (r *MongoRepository) ListDeleted(ctx context.Context, page, pageSize int) ([]model.Video, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	filter := deletedFilter()
	total64, err := r.col.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	opts := options.Find().
		SetSort(bson.D{{Key: "deleted_at", Value: -1}}).
		SetSkip(int64((page - 1) * pageSize)).
		SetLimit(int64(pageSize))
	cur, err := r.col.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cur.Close(ctx)
	return r.cursorToVideos(ctx, cur, int(total64))
}

func (r *MongoRepository) SoftDelete(ctx context.Context, videoID, deletedBy, reason string, deletedAt, purgeAt int64) (*model.Video, error) {
	res, err := r.col.UpdateOne(ctx,
		activeFilter(bson.M{"video_id": videoID}),
		bson.M{"$set": bson.M{
			"deleted_at":    deletedAt,
			"deleted_by":    deletedBy,
			"delete_reason": reason,
			"purge_at":      purgeAt,
		}},
	)
	if err != nil {
		return nil, err
	}
	if res.MatchedCount == 0 {
		return nil, ErrNotFound
	}
	return r.GetByIDIncludingDeleted(ctx, videoID)
}

func (r *MongoRepository) Restore(ctx context.Context, videoID string) (*model.Video, error) {
	res, err := r.col.UpdateOne(ctx,
		bson.M{"video_id": videoID, "deleted_at": bson.M{"$gt": 0}},
		bson.M{"$set": bson.M{
			"deleted_at":    int64(0),
			"deleted_by":    "",
			"delete_reason": "",
			"purge_at":      int64(0),
		}},
	)
	if err != nil {
		return nil, err
	}
	if res.MatchedCount == 0 {
		return nil, ErrNotFound
	}
	return r.GetByID(ctx, videoID)
}

func (r *MongoRepository) PermanentDelete(ctx context.Context, videoID string) error {
	res, err := r.col.DeleteOne(ctx, bson.M{"video_id": videoID})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *MongoRepository) UpdateTags(ctx context.Context, videoID string, tags []string) error {
	res, err := r.col.UpdateOne(ctx,
		bson.M{"video_id": videoID},
		bson.M{"$set": bson.M{"tags": tags}},
	)
	if err != nil {
		return err
	}
	if res.MatchedCount == 0 {
		return ErrNotFound
	}
	return nil
}
