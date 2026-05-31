FROM golang:1.22-alpine AS builder

WORKDIR /src
ENV GOPROXY=https://goproxy.cn,direct
ENV GOSUMDB=sum.golang.google.cn

COPY gen/go.mod gen/go.sum ./gen/
COPY pkg/auth/go.mod ./pkg/auth/
COPY pkg/limiter/go.mod ./pkg/limiter/
COPY pkg/redislimit/go.mod ./pkg/redislimit/
COPY pkg/objectstore/go.mod ./pkg/objectstore/
COPY api-gateway/go.mod api-gateway/go.sum ./api-gateway/

WORKDIR /src/api-gateway
RUN go mod download

COPY gen /src/gen
COPY pkg/auth /src/pkg/auth
COPY pkg/limiter /src/pkg/limiter
COPY pkg/redislimit /src/pkg/redislimit
COPY pkg/objectstore /src/pkg/objectstore
COPY api-gateway /src/api-gateway
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /api-gateway ./cmd

FROM alpine:3.20
WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /api-gateway /app/api-gateway
ENV SVP_SKIP_DOTENV=1
EXPOSE 8080
CMD ["/app/api-gateway"]
