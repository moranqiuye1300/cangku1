FROM golang:1.22-alpine AS builder

WORKDIR /src
ENV GOPROXY=https://goproxy.cn,direct
ENV GOSUMDB=sum.golang.google.cn

COPY gen/go.mod gen/go.sum ./gen/
COPY pkg/auth/go.mod ./pkg/auth/
COPY user-service/go.mod user-service/go.sum ./user-service/

WORKDIR /src/user-service
RUN go mod download

COPY gen /src/gen
COPY pkg/auth /src/pkg/auth
COPY user-service /src/user-service
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /user-service ./cmd

FROM alpine:3.20
WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /user-service /app/user-service
ENV SVP_SKIP_DOTENV=1
EXPOSE 50051
CMD ["/app/user-service"]
