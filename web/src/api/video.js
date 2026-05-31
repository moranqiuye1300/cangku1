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
