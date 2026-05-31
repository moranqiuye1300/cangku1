package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"short-video-platform/transcode-worker/internal/aitag"
	"short-video-platform/transcode-worker/internal/config"
	"short-video-platform/transcode-worker/internal/ffmpeg"
	"short-video-platform/transcode-worker/internal/grpcclient"
	"short-video-platform/transcode-worker/internal/worker"
)

func main() {
	config.LoadEnv()

	videoAddr := getenv("VIDEO_GRPC_ADDR", "127.0.0.1:50052")
	videoClient, conn, err := grpcclient.DialVideo(videoAddr)
	if err != nil {
		log.Fatalf("video client: %v", err)
	}
	defer conn.Close()

	mediaRoot := config.MediaRoot()
	publicURL := getenv("MEDIA_PUBLIC_URL", "/media")
	timeout := worker.ParseDurationEnv("TRANSCODE_TIMEOUT", 10*time.Minute)
	limit := parseInt(getenv("TRANSCODE_CONCURRENCY", "2"))

	transcoder := ffmpeg.New(mediaRoot, publicURL, timeout)
	tagClient := aitag.NewFromEnv()
	w := worker.New(videoClient, transcoder, tagClient, limit)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	log.Printf("transcode-worker started (media=%s concurrency=%d)", mediaRoot, limit)
	if err := worker.Run(ctx, w); err != nil && err != context.Canceled {
		log.Fatalf("worker: %v", err)
	}
	log.Println("transcode-worker stopped")
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func parseInt(s string) int {
	n := 0
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return 2
		}
		n = n*10 + int(ch-'0')
	}
	if n < 1 {
		return 2
	}
	return n
}
