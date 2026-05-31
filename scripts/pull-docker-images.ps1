# 预拉取构建所需基础镜像，避免 compose build 时 metadata 拉取超时
$images = @(
    "node:20-alpine",
    "nginx:alpine",
    "golang:1.22-alpine",
    "alpine:3.20",
    "mysql:8.0",
    "mongo:7",
    "bitnami/kafka:3.7",
    "elasticsearch:8.11.3",
    "redis:7-alpine",
    "minio/minio:RELEASE.2024-12-18T13-15-44Z"
)

foreach ($img in $images) {
    Write-Host "Pulling $img ..."
    docker pull $img
    if ($LASTEXITCODE -ne 0) {
        Write-Host "Failed: $img" -ForegroundColor Red
        exit 1
    }
}
Write-Host "All base images pulled." -ForegroundColor Green
