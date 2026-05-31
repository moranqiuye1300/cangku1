package grpcclient

import (
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"short-video-platform/gen/videopb"
	"short-video-platform/pkg/auth"
)

func DialVideo(addr string) (videopb.VideoServiceClient, *grpc.ClientConn, error) {
	conn, err := grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(auth.UnaryInternalClientInterceptor()),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("video grpc dial: %w", err)
	}
	return videopb.NewVideoServiceClient(conn), conn, nil
}
