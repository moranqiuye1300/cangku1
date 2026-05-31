package repository

import (
	"context"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type PreferenceRepository struct {
	col *mongo.Collection
}

func NewPreferenceRepository(db *mongo.Database) *PreferenceRepository {
	return &PreferenceRepository{col: db.Collection("user_preferences")}
}

type preferenceDoc struct {
	UserID     string             `bson:"user_id"`
	TagWeights map[string]float64 `bson:"tag_weights"`
	UpdatedAt  int64              `bson:"updated_at"`
}

func (r *PreferenceRepository) AddTagWeights(ctx context.Context, userID string, tags []string, delta float64) error {
	if userID == "" || len(tags) == 0 || delta == 0 {
		return nil
	}
	inc := bson.M{}
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		inc["tag_weights."+tag] = delta
	}
	if len(inc) == 0 {
		return nil
	}
	_, err := r.col.UpdateOne(ctx,
		bson.M{"user_id": userID},
		bson.M{
			"$inc": inc,
			"$set": bson.M{"updated_at": time.Now().Unix()},
			"$setOnInsert": bson.M{
				"user_id": userID,
			},
		},
		options.Update().SetUpsert(true),
	)
	return err
}

func (r *PreferenceRepository) GetWeights(ctx context.Context, userID string) (map[string]float64, error) {
	if userID == "" {
		return map[string]float64{}, nil
	}
	var doc preferenceDoc
	err := r.col.FindOne(ctx, bson.M{"user_id": userID}).Decode(&doc)
	if err == mongo.ErrNoDocuments {
		return map[string]float64{}, nil
	}
	if err != nil {
		return nil, err
	}
	if doc.TagWeights == nil {
		return map[string]float64{}, nil
	}
	out := make(map[string]float64, len(doc.TagWeights))
	for k, v := range doc.TagWeights {
		if v > 0 {
			out[k] = v
		}
	}
	return out, nil
}
