package main

import (
	"log"
	"os"

	"short-video-platform/api-gateway/internal/config"
	"short-video-platform/api-gateway/internal/grpcclient"
	"short-video-platform/api-gateway/internal/router"

	"github.com/gin-gonic/gin"
)

func main() {
	config.LoadEnv()

	if os.Getenv("GIN_MODE") == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	userAddr := getenv("USER_GRPC_ADDR", "127.0.0.1:50051")
	videoAddr := getenv("VIDEO_GRPC_ADDR", "127.0.0.1:50052")

	userClient, userConn, err := grpcclient.DialUser(userAddr)
	if err != nil {
		log.Fatalf("user client: %v", err)
	}
	defer userConn.Close()

	videoClient, videoConn, err := grpcclient.DialVideo(videoAddr)
	if err != nil {
		log.Fatalf("video client: %v", err)
	}
	defer videoConn.Close()

	addr := getenv("HTTP_ADDR", ":8080")
	r := router.Setup(router.Options{
		UserClient:  userClient,
		VideoClient: videoClient,
	})

	log.Printf("api-gateway listening on %s (user %s, video %s)", addr, userAddr, videoAddr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("http: %v", err)
	}
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
