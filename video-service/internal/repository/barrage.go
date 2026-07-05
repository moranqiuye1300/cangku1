package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"short-video-platform/video-service/internal/model"
)

type BarrageRepository struct {
	col *mongo.Collection
}

func NewBarrageRepository(db *mongo.Database) *BarrageRepository {
	return &BarrageRepository{col: db.Collection("barrages")}
}

func (r *BarrageRepository) ListByVideo(ctx context.Context, videoID string, page, pageSize int) ([]model.Barrage, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 100
	}
	filter := bson.M{"video_id": videoID}
	total64, err := r.col.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	skip := int64((page - 1) * pageSize)
	opts := options.Find().
		SetSort(bson.D{{Key: "time_ms", Value: 1}}).
		SetSkip(skip).
		SetLimit(int64(pageSize))
	cur, err := r.col.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cur.Close(ctx)
	list := make([]model.Barrage, 0)
	for cur.Next(ctx) {
		var doc model.BarrageDoc
		if err := cur.Decode(&doc); err != nil {
			return nil, 0, err
		}
		list = append(list, *model.DocToBarrage(&doc))
	}
	return list, int(total64), cur.Err()
}

func (r *BarrageRepository) Create(ctx context.Context, b *model.Barrage) error {
	count, err := r.col.CountDocuments(ctx, bson.M{"video_id": b.VideoID})
	if err != nil {
		return err
	}
	b.ID = fmt.Sprintf("b%d", count+1)
	if b.CreatedAt == 0 {
		b.CreatedAt = time.Now().Unix()
	}
	_, err = r.col.InsertOne(ctx, model.ToBarrageDoc(*b))
	return err
}

func (r *BarrageRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.col.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "video_id", Value: 1}, {Key: "time_ms", Value: 1}},
	})
	return err
}

func (r *BarrageRepository) DeleteByVideo(ctx context.Context, videoID string) error {
	_, err := r.col.DeleteMany(ctx, bson.M{"video_id": videoID})
	return err
}

var ErrBarrageNotFound = errors.New("barrage not found")
