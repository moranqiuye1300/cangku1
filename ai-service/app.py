import os
from typing import Any, List, Optional

import chromadb
from fastapi import FastAPI
from pydantic import BaseModel, Field
import httpx
import json
from fastapi import UploadFile, File
from langchain_pipeline import transcribe_audio_file, index_text, rag_answer

app = FastAPI(title="svp-ai-service", version="0.3.0-light")

# =====================
# ChromaDB (lightweight, no torch)
# =====================
CHROMA_PATH = os.getenv("CHROMA_PERSIST_PATH", "/data/chroma")
os.makedirs(CHROMA_PATH, exist_ok=True)

# Use Chroma's built-in lightweight embedding (no external model download)
client = chromadb.PersistentClient(path=CHROMA_PATH)
collection = client.get_or_create_collection(
    name="video_embeddings",
    metadata={"hnsw:space": "cosine"}
)

GATEWAY = os.getenv("AI_GATEWAY_URL", "http://api-gateway:8080").rstrip("/")


# =====================
# Pydantic Models
# =====================
class IndexVideoRequest(BaseModel):
    video_id: str = Field(..., min_length=1)
    title: str = Field(..., min_length=1)
    description: str = Field(default="")
    tags: List[str] = Field(default_factory=list)


class AskRequest(BaseModel):
    question: str = Field(min_length=2, max_length=500)
    context_video_ids: Optional[List[str]] = Field(
        default=None,
        description="Limit retrieval to these video IDs (from Feed)"
    )


class TagRequest(BaseModel):
    title: str = Field(default="", max_length=200)
    description: str = Field(default="", max_length=2000)


# =====================
# Helpers
# =====================
def _build_document_text(title: str, description: str, tags: List[str]) -> str:
    tag_str = " ".join(tags) if tags else ""
    return f"{title}. {description}. {tag_str}".strip()


# =====================
# Endpoints
# =====================
@app.get("/health")
def health() -> dict[str, Any]:
    return {
        "status": "ok",
        "service": "ai-service",
        "chroma_collection": collection.name,
        "engine": "chroma-lightweight",
        "document_count": collection.count()
    }


@app.post("/videos/index")
def index_video(req: IndexVideoRequest) -> dict[str, Any]:
    """Index a video into Chroma (called when video becomes 'ready')."""
    doc_text = _build_document_text(req.title, req.description, req.tags)
    metadata = {
        "video_id": req.video_id,
        "title": req.title,
        "tags": ",".join(req.tags) if req.tags else ""
    }
    # Use the shared indexing pipeline to split long text into chunks
    # and store them as separate documents with per-chunk metadata.
    try:
        chunks = index_text(video_id=req.video_id, title=req.title, text=doc_text, tags=req.tags)
    except Exception:
        # Fallback to single-document upsert if pipeline fails
        collection.upsert(ids=[req.video_id], documents=[doc_text], metadatas=[metadata])
        chunks = 1

    return {
        "indexed": True,
        "video_id": req.video_id,
        "chunks": chunks,
        "engine": "chroma-lightweight"
    }


@app.post("/videos/index_from_audio")
def index_from_audio(video_id: str, title: str, tags: Optional[List[str]] = None, file: UploadFile = File(...)) -> dict[str, Any]:
    """Upload audio, ASR -> split -> index into Chroma using LangChain pipeline."""
    # save uploaded file to temp
    tmp_path = f"/tmp/{video_id}-{file.filename}"
    with open(tmp_path, "wb") as f:
        f.write(file.file.read())
    # transcribe
    model_name = os.getenv("HF_MODEL_WHISPER", "small")
    transcript = transcribe_audio_file(tmp_path, model_name=model_name)
    count = index_text(video_id=video_id, title=title, text=transcript, tags=tags)
    return {"indexed": True, "video_id": video_id, "chunks": count}


@app.post("/rag/langchain")
def rag_langchain(req: AskRequest) -> dict[str, Any]:
    """LangChain-based RAG answer using Chroma retrieval + Aliyun LLM (if configured)."""
    q = req.question.strip()
    res = rag_answer(q)
    return res


@app.post("/rag/ask")
def rag_ask(req: AskRequest) -> dict[str, Any]:
    """
    Semantic RAG over indexed videos using Chroma.
    If context_video_ids is provided (from Feed), retrieval is limited to those videos.
    """
    q = req.question.strip()

    where_filter = None
    if req.context_video_ids:
        where_filter = {"video_id": {"$in": req.context_video_ids}}

    results = collection.query(
        query_texts=[q],
        n_results=5,
        where=where_filter,
        include=["documents", "metadatas", "distances"]
    )

    sources = []
    context_snippets = []

    if results and results.get("ids"):
        for i, vid in enumerate(results["ids"][0]):
            meta = results["metadatas"][0][i] if results["metadatas"] else {}
            doc = results["documents"][0][i] if results["documents"] else ""
            dist = results["distances"][0][i] if results["distances"] else 0.0

            sources.append({
                "id": vid,
                "title": meta.get("title", ""),
                "description": doc[:200],
                "score": round(1 - dist, 4)
            })
            context_snippets.append(f"【{meta.get('title','')}】{doc[:300]}")

    # Build a simple local answer summary from retrieved contexts
    if sources:
        titles = "、".join(s["title"] for s in sources if s.get("title"))
        local_answer = f"根据您推荐列表中的视频，我检索到与「{q}」最相关的内容有：{titles}。\n\n"
        local_answer += "相关片段：\n" + "\n".join(context_snippets[:3])
    else:
        local_answer = f"未在当前推荐视频中找到与「{q}」强相关的内容，建议换个问题或刷新推荐。"

    # If ALIYUN model endpoint is configured, call it to generate a richer answer.
    aliyun_url = os.getenv("ALIYUN_API_URL", "").strip()
    if aliyun_url:
        # Construct prompt combining question and context snippets
        prompt_parts = [f"问题：{q}", "\n检索到的上下文片段："]
        if context_snippets:
            prompt_parts.append("\n".join(context_snippets[:6]))
        prompt_parts.append("\n请基于上述内容给出简洁、有帮助的回答，中文输出。\n")
        prompt = "\n".join(prompt_parts)

        def call_aliyun_model(prompt_text: str) -> Optional[str]:
            headers = {"Content-Type": "application/json"}
            key = os.getenv("ALIYUN_API_KEY", "").strip()
            key_header = os.getenv("ALIYUN_API_KEY_HEADER", "Authorization").strip()
            if key:
                if key_header.lower() == "authorization":
                    headers["Authorization"] = f"Bearer {key}"
                else:
                    headers[key_header] = key
            payload = {"prompt": prompt_text}
            try:
                r = httpx.post(aliyun_url, json=payload, headers=headers, timeout=120.0)
                r.raise_for_status()
                ctype = r.headers.get("content-type", "")
                if "application/json" in ctype:
                    data = r.json()
                    # Try common fields used by various model endpoints
                    for k in ("answer", "output", "text", "result"):
                        if k in data and isinstance(data[k], str):
                            return data[k]
                    if "choices" in data and isinstance(data["choices"], list) and data["choices"]:
                        first = data["choices"][0]
                        if isinstance(first, dict):
                            for fld in ("text", "message", "content"):
                                if fld in first and isinstance(first[fld], str):
                                    return first[fld]
                        elif isinstance(first, str):
                            return first
                    # fallback to raw json string
                    return json.dumps(data, ensure_ascii=False)
                return r.text
            except Exception:
                return None

        model_resp = call_aliyun_model(prompt)
        if model_resp:
            answer = model_resp
        else:
            answer = local_answer
    else:
        answer = local_answer

    return {
        "question": q,
        "answer": answer,
        "sources": sources,
        "engine": "chroma-lightweight-rag",
        "context_limited_to_feed": bool(req.context_video_ids)
    }


# =====================
# Legacy keyword tagger (kept for backward compatibility)
# =====================
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


@app.post("/tags/generate")
def generate_tags(body: TagRequest) -> dict[str, Any]:
    tags = _extract_tags(body.title.strip(), body.description.strip())
    return {"tags": tags, "engine": "keyword-tagger-v1"}