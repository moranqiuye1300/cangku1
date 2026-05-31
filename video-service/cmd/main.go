package main

import (
	"context"
	"log"
	"net"
	"os"

	"google.golang.org/grpc"

	"short-video-platform/gen/videopb"
	"short-video-platform/pkg/auth"
	"short-video-platform/video-service/internal/config"
	"short-video-platform/video-service/internal/database"
	"short-video-platform/video-service/internal/handler"
	"short-video-platform/video-service/internal/kafka"
	"short-video-platform/video-service/internal/repository"
	"short-video-platform/video-service/internal/search"
	"short-video-platform/video-service/internal/service"
)

func main() {
	config.LoadEnv()

	ctx := context.Background()
	db, err := database.OpenMongo(ctx)
	if err != nil {
		log.Fatalf("mongodb: %v", err)
	}

	repo := repository.NewMongoRepository(db)
	barrageRepo := repository.NewBarrageRepository(db)
	interactRepo := repository.NewInteractionRepository(db)
	auditRepo := repository.NewAuditRepository(db)
	archiveRepo := repository.NewArchiveRepository(db)
	prefRepo := repository.NewPreferenceRepository(db)
	_ = barrageRepo.EnsureIndexes(ctx)
	_ = interactRepo.EnsureIndexes(ctx)
	if err := service.SeedVideos(ctx, repo); err != nil {
		log.Fatalf("seed videos: %v", err)
	}

	producer, err := kafka.NewProducer()
	if err != nil {
		log.Fatalf("kafka producer: %v", err)
	}
	defer producer.Close()

	searchClient, err := search.NewFromEnv()
	if err != nil {
		log.Fatalf("elasticsearch: %v", err)
	}
	if searchClient != nil {
		if err := searchClient.Ping(ctx); err != nil {
			log.Fatalf("elasticsearch ping: %v", err)
		}
		log.Printf("elasticsearch connected")
	} else {
		log.Printf("elasticsearch disabled (ELASTICSEARCH_URL empty)")
	}
	if err := service.BackfillTags(ctx, repo, searchClient); err != nil {
		log.Printf("backfill tags warning: %v", err)
	}

	addr := getenv("VIDEO_GRPC_ADDR", ":50052")
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}

	svc := service.NewVideoService(repo, barrageRepo, interactRepo, auditRepo, archiveRepo, prefRepo, producer, searchClient)
	if searchClient != nil {
		if err := svc.ReindexSearch(ctx); err != nil {
			log.Printf("elasticsearch reindex warning: %v", err)
		} else {
			log.Printf("elasticsearch reindex done")
		}
	}

	srv := grpc.NewServer(grpc.UnaryInterceptor(auth.UnaryServerInterceptor(auth.DefaultVideoAuthRules())))
	videopb.RegisterVideoServiceServer(srv, handler.NewVideoGRPCServer(svc))

	log.Printf("video-service listening on %s (mongodb + kafka + search + grpc-auth)", addr)
	if err := srv.Serve(lis); err != nil {
		log.Fatalf("serve: %v", err)
	}
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
