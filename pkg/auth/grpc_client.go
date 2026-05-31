package auth

import (
	"context"

	"google.golang.org/grpc"
)

func UnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		ctx = OutgoingContext(ctx)
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

func UnaryInternalClientInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		ctx = OutgoingInternalContext(ctx)
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}
