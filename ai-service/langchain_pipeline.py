import os
from typing import List, Optional

import chromadb
import httpx

CHROMA_PATH = os.getenv("CHROMA_PERSIST_PATH", "/data/chroma")
client = chromadb.PersistentClient(path=CHROMA_PATH)
collection = client.get_or_create_collection(name="video_embeddings", metadata={"hnsw:space": "cosine"})


def _simple_split(text: str, chunk_size: int = 1000, overlap: int = 200) -> List[str]:
    if not text:
        return []
    chunks = []
    start = 0
    L = len(text)
    while start < L:
        end = min(start + chunk_size, L)
        chunks.append(text[start:end])
        start = end - overlap if end - overlap > start else end
        if start >= L:
            break
    return chunks


class AliyunLLM:
    def __init__(self):
        self.url = os.getenv("ALIYUN_API_URL", "").strip()

    def call(self, prompt: str) -> str:
        if not self.url:
            raise RuntimeError("ALIYUN_API_URL not configured")
        headers = {"Content-Type": "application/json"}
        key = os.getenv("ALIYUN_API_KEY", "").strip()
        key_header = os.getenv("ALIYUN_API_KEY_HEADER", "Authorization").strip()
        if key:
            if key_header.lower() == "authorization":
                headers["Authorization"] = f"Bearer {key}"
            else:
                headers[key_header] = key
        payload = {"prompt": prompt}
        r = httpx.post(self.url, json=payload, headers=headers, timeout=120.0)
        r.raise_for_status()
        ctype = r.headers.get("content-type", "")
        if "application/json" in ctype:
            data = r.json()
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
            return str(data)
        return r.text


def transcribe_audio_file(path: str) -> str:
    """No local ASR available in lightweight mode. Raise to indicate external ASR required."""
    raise RuntimeError("Local ASR not available in lightweight mode. Configure external ASR or enable faster-whisper and rebuild.")


def index_text(video_id: str, title: str, text: str, tags: Optional[List[str]] = None):
    docs = _simple_split(text)
    ids = [f"{video_id}-{i}" for i in range(len(docs))]
    metadatas = []
    for i, doc in enumerate(docs):
        md = {"video_id": video_id, "title": title}
        if tags:
            md["tags"] = ",".join(tags)
        metadatas.append(md)
    if docs:
        collection.upsert(ids=ids, documents=docs, metadatas=metadatas)
    return len(docs)


def rag_answer(question: str, top_k: int = 5) -> dict:
    results = collection.query(query_texts=[question], n_results=top_k, include=["documents", "metadatas", "distances"])
    sources = []
    context_snippets = []
    if results and results.get("ids"):
        for i, vid in enumerate(results["ids"][0]):
            meta = results.get("metadatas", [[]])[0][i] if results.get("metadatas") else {}
            doc = results.get("documents", [[]])[0][i] if results.get("documents") else ""
            dist = results.get("distances", [[]])[0][i] if results.get("distances") else 0.0
            sources.append({"id": vid, "title": meta.get("title", ""), "score": round(1 - dist, 4)})
            context_snippets.append(f"【{meta.get('title','')}】{doc[:400]}")

    local_answer = "未检索到相关内容"
    if context_snippets:
        local_answer = "基于检索到的片段：\n" + "\n".join(context_snippets[:5])

    if os.getenv("ALIYUN_API_URL", "").strip():
        llm = AliyunLLM()
        prompt = f"问题：{question}\n\n检索到的上下文：\n" + "\n".join(context_snippets[:6]) + "\n\n请基于以上内容回答，中文输出，简明扼要。"
        try:
            answer = llm.call(prompt)
        except Exception:
            answer = local_answer
    else:
        answer = local_answer

    return {"question": question, "answer": answer, "sources": sources}
