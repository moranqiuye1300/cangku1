FROM golang:1.22-alpine AS builder

WORKDIR /src
ENV GOPROXY=https://goproxy.cn,direct
ENV GOSUMDB=sum.golang.google.cn

COPY gen/go.mod gen/go.sum ./gen/
COPY pkg/auth/go.mod ./pkg/auth/
COPY pkg/events/go.mod ./pkg/events/
COPY video-service/go.mod video-service/go.sum ./video-service/

WORKDIR /src/video-service
RUN go mod download

COPY gen /src/gen
COPY pkg/auth /src/pkg/auth
COPY pkg/events /src/pkg/events
COPY video-service /src/video-service
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /video-service ./cmd

FROM alpine:3.20
WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /video-service /app/video-service
ENV SVP_SKIP_DOTENV=1
EXPOSE 50052
CMD ["/app/video-service"]
