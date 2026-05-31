package repository

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"short-video-platform/video-service/internal/model"
)

type InteractionRepository struct {
	likes     *mongo.Collection
	favorites *mongo.Collection
	comments  *mongo.Collection
	stats     *mongo.Collection
}

func NewInteractionRepository(db *mongo.Database) *InteractionRepository {
	return &InteractionRepository{
		likes:     db.Collection("likes"),
		favorites: db.Collection("favorites"),
		comments:  db.Collection("comments"),
		stats:     db.Collection("video_stats"),
	}
}

func (r *InteractionRepository) EnsureIndexes(ctx context.Context) error {
	models := []mongo.IndexModel{
		{Keys: bson.D{{Key: "video_id", Value: 1}, {Key: "user_id", Value: 1}}, Options: options.Index().SetUnique(true)},
	}
	if _, err := r.likes.Indexes().CreateOne(ctx, models[0]); err != nil {
		return err
	}
	if _, err := r.favorites.Indexes().CreateOne(ctx, models[0]); err != nil {
		return err
	}
	_, err := r.comments.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "video_id", Value: 1}, {Key: "created_at", Value: -1}},
	})
	if err != nil {
		return err
	}
	_, err = r.likes.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "created_at", Value: -1}},
	})
	if err != nil {
		return err
	}
	_, err = r.favorites.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "created_at", Value: -1}},
	})
	return err
}

func (r *InteractionRepository) ensureStats(ctx context.Context, videoID string) error {
	_, err := r.stats.UpdateOne(ctx,
		bson.M{"video_id": videoID},
		bson.M{"$setOnInsert": bson.M{
			"video_id":       videoID,
			"like_count":     int64(0),
			"comment_count":  int64(0),
			"favorite_count": int64(0),
		}},
		options.Update().SetUpsert(true),
	)
	return err
}

func (r *InteractionRepository) GetStats(ctx context.Context, videoID string) (*model.VideoStats, error) {
	if err := r.ensureStats(ctx, videoID); err != nil {
		return nil, err
	}
	var doc model.VideoStatsDoc
	err := r.stats.FindOne(ctx, bson.M{"video_id": videoID}).Decode(&doc)
	if err != nil {
		return nil, err
	}
	return &model.VideoStats{
		VideoID:       doc.VideoID,
		LikeCount:     doc.LikeCount,
		CommentCount:  doc.CommentCount,
		FavoriteCount: doc.FavoriteCount,
	}, nil
}

func (r *InteractionRepository) HasLike(ctx context.Context, videoID, userID string) (bool, error) {
	if userID == "" {
		return false, nil
	}
	err := r.likes.FindOne(ctx, bson.M{"video_id": videoID, "user_id": userID}).Err()
	if err == mongo.ErrNoDocuments {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *InteractionRepository) HasFavorite(ctx context.Context, videoID, userID string) (bool, error) {
	if userID == "" {
		return false, nil
	}
	err := r.favorites.FindOne(ctx, bson.M{"video_id": videoID, "user_id": userID}).Err()
	if err == mongo.ErrNoDocuments {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *InteractionRepository) ToggleLike(ctx context.Context, videoID, userID string) (bool, int64, error) {
	if err := r.ensureStats(ctx, videoID); err != nil {
		return false, 0, err
	}
	filter := bson.M{"video_id": videoID, "user_id": userID}
	err := r.likes.FindOne(ctx, filter).Err()
	if err == mongo.ErrNoDocuments {
		_, insErr := r.likes.InsertOne(ctx, model.LikeDoc{
			VideoID: videoID, UserID: userID, CreatedAt: time.Now().Unix(),
		})
		if insErr != nil {
			return false, 0, insErr
		}
		var doc model.VideoStatsDoc
		if err := r.stats.FindOneAndUpdate(ctx, bson.M{"video_id": videoID}, bson.M{"$inc": bson.M{"like_count": 1}},
			options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&doc); err != nil {
			return false, 0, err
		}
		return true, doc.LikeCount, nil
	}
	if err != nil {
		return false, 0, err
	}
	if _, err := r.likes.DeleteOne(ctx, filter); err != nil {
		return false, 0, err
	}
	var doc model.VideoStatsDoc
	if err := r.stats.FindOneAndUpdate(ctx, bson.M{"video_id": videoID, "like_count": bson.M{"$gt": 0}},
		bson.M{"$inc": bson.M{"like_count": -1}},
		options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&doc); err != nil {
		if err == mongo.ErrNoDocuments {
			_, _ = r.stats.UpdateOne(ctx, bson.M{"video_id": videoID}, bson.M{"$set": bson.M{"like_count": int64(0)}})
			return false, 0, nil
		}
		return false, 0, err
	}
	return false, doc.LikeCount, nil
}

func (r *InteractionRepository) ToggleFavorite(ctx context.Context, videoID, userID string) (bool, int64, error) {
	if err := r.ensureStats(ctx, videoID); err != nil {
		return false, 0, err
	}
	filter := bson.M{"video_id": videoID, "user_id": userID}
	err := r.favorites.FindOne(ctx, filter).Err()
	if err == mongo.ErrNoDocuments {
		_, insErr := r.favorites.InsertOne(ctx, model.FavoriteDoc{
			VideoID: videoID, UserID: userID, CreatedAt: time.Now().Unix(),
		})
		if insErr != nil {
			return false, 0, insErr
		}
		var doc model.VideoStatsDoc
		if err := r.stats.FindOneAndUpdate(ctx, bson.M{"video_id": videoID}, bson.M{"$inc": bson.M{"favorite_count": 1}},
			options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&doc); err != nil {
			return false, 0, err
		}
		return true, doc.FavoriteCount, nil
	}
	if err != nil {
		return false, 0, err
	}
	if _, err := r.favorites.DeleteOne(ctx, filter); err != nil {
		return false, 0, err
	}
	var doc model.VideoStatsDoc
	if err := r.stats.FindOneAndUpdate(ctx, bson.M{"video_id": videoID, "favorite_count": bson.M{"$gt": 0}},
		bson.M{"$inc": bson.M{"favorite_count": -1}},
		options.FindOneAndUpdate().SetReturnDocument(options.After)).Decode(&doc); err != nil {
		if err == mongo.ErrNoDocuments {
			_, _ = r.stats.UpdateOne(ctx, bson.M{"video_id": videoID}, bson.M{"$set": bson.M{"favorite_count": int64(0)}})
			return false, 0, nil
		}
		return false, 0, err
	}
	return false, doc.FavoriteCount, nil
}

func (r *InteractionRepository) ListComments(ctx context.Context, videoID string, page, pageSize int) ([]model.Comment, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 20
	}
	filter := bson.M{"video_id": videoID}
	total64, err := r.comments.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetSkip(int64((page - 1) * pageSize)).
		SetLimit(int64(pageSize))
	cur, err := r.comments.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cur.Close(ctx)
	list := make([]model.Comment, 0)
	for cur.Next(ctx) {
		var doc model.CommentDoc
		if err := cur.Decode(&doc); err != nil {
			return nil, 0, err
		}
		list = append(list, *model.DocToComment(&doc))
	}
	return list, int(total64), cur.Err()
}

func (r *InteractionRepository) CreateComment(ctx context.Context, c *model.Comment) error {
	if err := r.ensureStats(ctx, c.VideoID); err != nil {
		return err
	}
	count, err := r.comments.CountDocuments(ctx, bson.M{"video_id": c.VideoID})
	if err != nil {
		return err
	}
	c.ID = fmt.Sprintf("%s-c%d", c.VideoID, count+1)
	if c.CreatedAt == 0 {
		c.CreatedAt = time.Now().Unix()
	}
	if _, err := r.comments.InsertOne(ctx, model.ToCommentDoc(*c)); err != nil {
		return err
	}
	_, err = r.stats.UpdateOne(ctx, bson.M{"video_id": c.VideoID}, bson.M{"$inc": bson.M{"comment_count": 1}})
	return err
}

func (r *InteractionRepository) listVideoIDsByUser(
	ctx context.Context,
	col *mongo.Collection,
	userID string,
	page, pageSize int,
) ([]string, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 50 {
		pageSize = 12
	}
	filter := bson.M{"user_id": userID}
	total64, err := col.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetSkip(int64((page - 1) * pageSize)).
		SetLimit(int64(pageSize))
	cur, err := col.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cur.Close(ctx)
	ids := make([]string, 0)
	for cur.Next(ctx) {
		var doc struct {
			VideoID string `bson:"video_id"`
		}
		if err := cur.Decode(&doc); err != nil {
			return nil, 0, err
		}
		if doc.VideoID != "" {
			ids = append(ids, doc.VideoID)
		}
	}
	return ids, int(total64), cur.Err()
}

func (r *InteractionRepository) ListLikedVideoIDs(ctx context.Context, userID string, page, pageSize int) ([]string, int, error) {
	return r.listVideoIDsByUser(ctx, r.likes, userID, page, pageSize)
}

func (r *InteractionRepository) ListFavoriteVideoIDs(ctx context.Context, userID string, page, pageSize int) ([]string, int, error) {
	return r.listVideoIDsByUser(ctx, r.favorites, userID, page, pageSize)
}

func (r *InteractionRepository) ClearAllForVideo(ctx context.Context, videoID string) (*model.VideoStats, error) {
	if _, err := r.likes.DeleteMany(ctx, bson.M{"video_id": videoID}); err != nil {
		return nil, err
	}
	if _, err := r.favorites.DeleteMany(ctx, bson.M{"video_id": videoID}); err != nil {
		return nil, err
	}
	if _, err := r.comments.DeleteMany(ctx, bson.M{"video_id": videoID}); err != nil {
		return nil, err
	}
	var stats model.VideoStatsDoc
	err := r.stats.FindOne(ctx, bson.M{"video_id": videoID}).Decode(&stats)
	if err == mongo.ErrNoDocuments {
		return &model.VideoStats{VideoID: videoID}, nil
	}
	if err != nil {
		return nil, err
	}
	_, err = r.stats.UpdateOne(ctx, bson.M{"video_id": videoID}, bson.M{"$set": bson.M{
		"like_count":     int64(0),
		"comment_count":  int64(0),
		"favorite_count": int64(0),
	}})
	if err != nil {
		return nil, err
	}
	return &model.VideoStats{
		VideoID:       videoID,
		LikeCount:     stats.LikeCount,
		CommentCount:  stats.CommentCount,
		FavoriteCount: stats.FavoriteCount,
	}, nil
}
