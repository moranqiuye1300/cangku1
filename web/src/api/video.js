import request from './request'

export function fetchVideos(params) {
  return request.get('/videos', { params })
}

export function fetchFeed(params) {
  return request.get('/videos/feed', { params })
}

export function searchVideos(params) {
  return request.get('/videos/search', { params })
}

export function fetchVideo(id) {
  return request.get(`/videos/${id}`)
}

export function uploadVideo(formData) {
  return request.post('/videos/upload', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
    timeout: 120000
  })
}

export function initChunkUpload(payload) {
  return request.post('/videos/upload/init', payload, {
    timeout: 30000
  })
}

export function uploadChunk(formData) {
  return request.post('/videos/upload/chunk', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
    timeout: 300000
  })
}

export function completeChunkUpload(payload) {
  return request.post('/videos/upload/complete', payload, {
    timeout: 120000
  })
}

export function askAI(question, contextVideoIDs = []) {
  return request.post('/ai/ask', {
    question,
    context_video_ids: contextVideoIDs
  }, {
    timeout: 60000   // AI 查询允许最多 60 秒
  })
}
