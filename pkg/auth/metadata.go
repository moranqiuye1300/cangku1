package auth

import (
	"context"
	"strings"

	"google.golang.org/grpc/metadata"
)

const (
	HeaderAuthorization = "authorization"
	HeaderInternalKey   = "x-internal-key"
)

type ctxKey int

const bearerKey ctxKey = 1

func WithBearer(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, bearerKey, strings.TrimSpace(token))
}

func BearerFromContext(ctx context.Context) string {
	v, _ := ctx.Value(bearerKey).(string)
	return v
}

func OutgoingContext(ctx context.Context) context.Context {
	md := metadata.MD{}
	if token := BearerFromContext(ctx); token != "" {
		md.Set(HeaderAuthorization, "Bearer "+token)
	}
	if len(md) == 0 {
		return ctx
	}
	return metadata.NewOutgoingContext(ctx, md)
}

func OutgoingInternalContext(ctx context.Context) context.Context {
	md := metadata.Pairs(HeaderInternalKey, InternalKey())
	return metadata.NewOutgoingContext(ctx, md)
}

func TokenFromIncoming(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	vals := md.Get(HeaderAuthorization)
	if len(vals) == 0 {
		return ""
	}
	auth := vals[0]
	if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		return strings.TrimSpace(auth[7:])
	}
	return strings.TrimSpace(auth)
}

func InternalKeyFromIncoming(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	vals := md.Get(HeaderInternalKey)
	if len(vals) == 0 {
		return ""
	}
	return vals[0]
}
