import os
from typing import Any

import httpx
from fastapi import FastAPI
from pydantic import BaseModel, Field

app = FastAPI(title="svp-ai-service", version="0.1.0")

GATEWAY = os.getenv("AI_GATEWAY_URL", "http://api-gateway:8080").rstrip("/")


class AskRequest(BaseModel):
    question: str = Field(min_length=2, max_length=500)


class TagRequest(BaseModel):
    title: str = Field(default="", max_length=200)
    description: str = Field(default="", max_length=2000)


KEYWORD_TAGS: dict[str, list[str]] = {
    "go": ["Go", "编程", "后端"],
    "gin": ["Gin", "Web", "后端"],
    "grpc": ["gRPC", "微服务", "后端"],
    "并发": ["Go", "并发", "编程"],
    "goroutine": ["Go", "并发", "编程"],
    "channel": ["Go", "并发", "编程"],
    "微服务": ["微服务", "后端", "架构"],
    "api": ["Web", "后端"],
    "rest": ["Web", "REST", "后端"],
    "docker": ["Docker", "DevOps", "部署"],
    "kafka": ["Kafka", "消息队列", "后端"],
    "redis": ["Redis", "缓存", "后端"],
    "mysql": ["MySQL", "数据库", "后端"],
    "mongo": ["MongoDB", "数据库", "后端"],
    "vue": ["Vue", "前端", "Web"],
    "python": ["Python", "编程"],
    "教程": ["教程", "入门"],
    "入门": ["教程", "入门"],
    "实战": ["实战", "项目"],
    "框架": ["框架", "Web"],
}


def _extract_tags(title: str, description: str) -> list[str]:
    text = f"{title} {description}".lower()
    seen: set[str] = set()
    tags: list[str] = []
    for kw, labels in KEYWORD_TAGS.items():
        if kw in text:
            for label in labels:
                if label not in seen:
                    seen.add(label)
                    tags.append(label)
    if not tags:
        tags.append("短视频")
    return tags[:8]


@app.get("/health")
def health() -> dict[str, Any]:
    return {"status": "ok", "service": "ai-service"}


@app.post("/tags/generate")
def generate_tags(body: TagRequest) -> dict[str, Any]:
    tags = _extract_tags(body.title.strip(), body.description.strip())
    return {"tags": tags, "engine": "keyword-tagger-v1"}


@app.post("/rag/ask")
async def rag_ask(body: AskRequest) -> dict[str, Any]:
    q = body.question.strip()
    sources: list[dict[str, Any]] = []
    try:
        async with httpx.AsyncClient(timeout=10.0) as client:
            resp = await client.get(
                f"{GATEWAY}/api/v1/videos/search",
                params={"q": q, "page": 1, "page_size": 5},
            )
            if resp.status_code == 200:
                data = resp.json().get("data") or {}
                for v in data.get("videos") or []:
                    sources.append(
                        {
                            "id": v.get("id"),
                            "title": v.get("title"),
                            "description": v.get("description"),
                        }
                    )
    except Exception:
        pass

    if sources:
        titles = "、".join(s["title"] for s in sources if s.get("title"))
        answer = f"根据平台视频检索，与「{q}」相关的内容有：{titles}。你可以点开详情页观看。"
    else:
        answer = f"未检索到与「{q}」强相关的视频，建议换个关键词或上传新视频后再问。"

    return {
        "question": q,
        "answer": answer,
        "sources": sources,
        "engine": "langchain-style-rag-mvp",
    }
