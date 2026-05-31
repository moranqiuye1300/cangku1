# 短视频播放平台 — 阶段 4

> OAuth2 · Redis 分布式限流 · gRPC JWT · 弹幕 · AI 问答 · MinIO(OSS)

## 阶段能力总览

| 阶段 | 能力 |
|------|------|
| 0 | Nginx + Vue + Gin，`/api/health` |
| 1 | 注册登录、视频列表/详情 |
| 2 | 上传 + 本地/OSS 存储元数据 |
| 3 | Kafka + FFmpeg 异步多码率 HLS |
| 4 | ES 搜索 + **本阶段增强** |

## 阶段 4 新增

| 模块 | 说明 |
|------|------|
| **gRPC JWT 拦截器** | `pkg/auth`，`CreateVideo`/`PostBarrage` 需 JWT；`UpdateTranscodeResult` 需内部密钥 |
| **Redis 限流** | `pkg/redislimit`，网关 `/api/v1/*` 分布式计数 + 本地令牌桶双保险 |
| **OAuth2** | GitHub OAuth（可配置）；`/api/v1/auth/oauth/mock` Mock 登录 |
| **OSS/MinIO** | `pkg/objectstore`，`OBJECT_STORE=minio` 时对接 S3 兼容存储 |
| **弹幕** | MongoDB `barrages` 集合，列表/发送 API |
| **LangChain MVP** | `ai-service`（FastAPI）检索视频后生成回答 |

## 架构

```
浏览器 → Nginx → api-gateway (Redis 限流 + JWT)
                    ├─ gRPC(+metadata) → user-service (MySQL, OAuth)
                    ├─ gRPC(+metadata) → video-service (Mongo, ES, 弹幕, gRPC 鉴权)
                    └─ HTTP → ai-service (RAG MVP)

transcode-worker ──(内部密钥 gRPC)──► video-service.UpdateTranscodeResult
```

## Docker 启动

```powershell
docker compose up -d --build --remove-orphans
```

新增容器：`redis`、`minio`、`ai-service`、`elasticsearch`（阶段 3）

## 关键 API

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/v1/auth/oauth/mock?oauth_id=demo` | Mock OAuth 登录 |
| GET | `/api/v1/auth/oauth/github/url` | GitHub 授权地址 |
| GET | `/api/v1/videos/search?q=` | ES 搜索 |
| GET | `/api/v1/videos/:id/barrages` | 弹幕列表 |
| POST | `/api/v1/videos/:id/barrages` | 发弹幕（JWT） |
| POST | `/api/v1/ai/ask` | AI 问答 `{question}` |

## 环境变量（节选）

```env
REDIS_ADDR=127.0.0.1:6379
INTERNAL_GRPC_KEY=svp-internal-dev-key
OBJECT_STORE=local          # 或 minio
MINIO_ENDPOINT=127.0.0.1:9000
CDN_PUBLIC_URL=/media
GITHUB_CLIENT_ID=           # 可选
AI_SERVICE_URL=http://127.0.0.1:8090
```

## 验收建议

1. `GET /api/v1/health` → `stage: "4"`
2. 登录页 **GitHub OAuth（Mock）** → 获得 token
3. 视频详情页发弹幕、AI 提问
4. 搜索 `Go` 仍有结果
5. 快速刷接口 → **429**（Redis 限流）

## Proto 重新生成

```powershell
.\scripts\regenerate_proto.ps1
```
