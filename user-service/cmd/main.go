package main

import (
	"context"
	"log"
	"net"
	"os"

	"google.golang.org/grpc"

	"short-video-platform/gen/userpb"
	"short-video-platform/pkg/auth"
	"short-video-platform/user-service/internal/config"
	"short-video-platform/user-service/internal/database"
	"short-video-platform/user-service/internal/grpcclient"
	"short-video-platform/user-service/internal/handler"
	"short-video-platform/user-service/internal/repository"
	"short-video-platform/user-service/internal/service"
)

func main() {
	config.LoadEnv()

	db, err := database.OpenMySQL()
	if err != nil {
		log.Fatalf("mysql: %v", err)
	}
	if err := database.AutoMigrate(db); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	repo := repository.NewMySQLRepository(db)
	if err := service.SeedUsers(context.Background(), repo); err != nil {
		log.Fatalf("seed users: %v", err)
	}

	videoAddr := getenv("VIDEO_GRPC_ADDR", "127.0.0.1:50052")
	videoClient, videoConn, err := grpcclient.DialVideo(videoAddr)
	if err != nil {
		log.Fatalf("video client: %v", err)
	}
	defer videoConn.Close()

	addr := getenv("USER_GRPC_ADDR", ":50051")
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}

	svc := service.NewUserService(repo, videoClient)
	srv := grpc.NewServer(grpc.UnaryInterceptor(auth.UserUnaryServerInterceptor(auth.DefaultUserAuthRules())))
	userpb.RegisterUserServiceServer(srv, handler.NewUserGRPCServer(svc))

	log.Printf("user-service listening on %s (mysql + video %s)", addr, videoAddr)
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
