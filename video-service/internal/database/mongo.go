package database

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func OpenMongo(ctx context.Context) (*mongo.Database, error) {
	uri := getenv("MONGODB_URI", "mongodb://127.0.0.1:27017")
	dbName := getenv("MONGODB_DATABASE", "short_video")
	log.Printf("connecting mongodb: %s db=%s", uri, dbName)

	var lastErr error
	for i := 1; i <= 30; i++ {
		pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		client, err := mongo.Connect(pingCtx, options.Client().ApplyURI(uri))
		if err != nil {
			lastErr = err
			cancel()
			log.Printf("mongo attempt %d/30 connect: %v", i, err)
			time.Sleep(2 * time.Second)
			continue
		}
		if err = client.Ping(pingCtx, nil); err != nil {
			lastErr = err
			_ = client.Disconnect(context.Background())
			cancel()
			log.Printf("mongo attempt %d/30 ping: %v", i, err)
			time.Sleep(2 * time.Second)
			continue
		}
		cancel()

		db := client.Database(dbName)
		if err := ensureIndexes(ctx, db); err != nil {
			return nil, err
		}
		return db, nil
	}
	return nil, fmt.Errorf("mongo connect failed: %w", lastErr)
}

func ensureIndexes(ctx context.Context, db *mongo.Database) error {
	videos := db.Collection("videos")
	_, err := videos.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "video_id", Value: 1}}, Options: options.Index().SetUnique(true)},
		{Keys: bson.D{{Key: "user_id", Value: 1}, {Key: "created_at", Value: -1}}},
	})
	if err != nil {
		return fmt.Errorf("videos indexes: %w", err)
	}

	barrages := db.Collection("barrages")
	_, err = barrages.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "video_id", Value: 1}, {Key: "time_offset", Value: 1}},
	})
	if err != nil {
		return fmt.Errorf("barrages indexes: %w", err)
	}
	return nil
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
