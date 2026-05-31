FROM golang:1.22-alpine AS builder

WORKDIR /src
ENV GOPROXY=https://goproxy.cn,direct
ENV GOSUMDB=sum.golang.google.cn

COPY gen/go.mod gen/go.sum ./gen/
COPY pkg/auth/go.mod ./pkg/auth/
COPY pkg/events/go.mod ./pkg/events/
COPY transcode-worker/go.mod transcode-worker/go.sum ./transcode-worker/

WORKDIR /src/transcode-worker
RUN go mod download

COPY gen /src/gen
COPY pkg/auth /src/pkg/auth
COPY pkg/events /src/pkg/events
COPY transcode-worker /src/transcode-worker
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /transcode-worker ./cmd

FROM alpine:3.20
RUN apk add --no-cache ffmpeg ca-certificates tzdata
WORKDIR /app
COPY --from=builder /transcode-worker /app/transcode-worker
ENV SVP_SKIP_DOTENV=1
CMD ["/app/transcode-worker"]
