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

type AuditRepository struct {
	col *mongo.Collection
}

func NewAuditRepository(db *mongo.Database) *AuditRepository {
	return &AuditRepository{col: db.Collection("audit_logs")}
}

func (r *AuditRepository) Create(ctx context.Context, log *model.AuditLog) error {
	count, err := r.col.CountDocuments(ctx, bson.M{})
	if err != nil {
		return err
	}
	if log.ID == "" {
		log.ID = fmt.Sprintf("log%d", count+1)
	}
	if log.CreatedAt == 0 {
		log.CreatedAt = time.Now().Unix()
	}
	_, err = r.col.InsertOne(ctx, model.AuditLogDoc{
		LogID:         log.ID,
		Action:        log.Action,
		ActorID:       log.ActorID,
		ActorUsername: log.ActorUsername,
		TargetType:    log.TargetType,
		TargetID:      log.TargetID,
		IP:            log.IP,
		UserAgent:     log.UserAgent,
		Detail:        log.Detail,
		CreatedAt:     log.CreatedAt,
	})
	return err
}

func (r *AuditRepository) List(ctx context.Context, page, pageSize int, targetType string) ([]model.AuditLog, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	filter := bson.M{}
	if targetType != "" {
		filter["target_type"] = targetType
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
	list := make([]model.AuditLog, 0)
	for cur.Next(ctx) {
		var doc model.AuditLogDoc
		if err := cur.Decode(&doc); err != nil {
			return nil, 0, err
		}
		list = append(list, model.AuditLog{
			ID:            doc.LogID,
			Action:        doc.Action,
			ActorID:       doc.ActorID,
			ActorUsername: doc.ActorUsername,
			TargetType:    doc.TargetType,
			TargetID:      doc.TargetID,
			IP:            doc.IP,
			UserAgent:     doc.UserAgent,
			Detail:        doc.Detail,
			CreatedAt:     doc.CreatedAt,
		})
	}
	return list, int(total64), cur.Err()
}

type ArchiveRepository struct {
	col *mongo.Collection
}

func NewArchiveRepository(db *mongo.Database) *ArchiveRepository {
	return &ArchiveRepository{col: db.Collection("video_archives")}
}

func (r *ArchiveRepository) Save(ctx context.Context, archive *model.VideoArchive) error {
	_, err := r.col.UpdateOne(ctx,
		bson.M{"video_id": archive.VideoID},
		bson.M{"$set": model.VideoArchiveDoc{
			VideoID:        archive.VideoID,
			VideoSnapshot:  model.ToDoc(archive.VideoSnapshot),
			LikeCount:      archive.LikeCount,
			CommentCount:   archive.CommentCount,
			FavoriteCount:  archive.FavoriteCount,
			DeletedAt:      archive.DeletedAt,
			DeletedBy:      archive.DeletedBy,
			DeleteReason:   archive.DeleteReason,
			RetentionUntil: archive.RetentionUntil,
		}},
		options.Update().SetUpsert(true),
	)
	return err
}
