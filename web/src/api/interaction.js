import request from './request'

export function fetchEngagement(videoId) {
  return request.get(`/videos/${videoId}/engagement`)
}

export function toggleLike(videoId) {
  return request.post(`/videos/${videoId}/like`)
}

export function toggleFavorite(videoId) {
  return request.post(`/videos/${videoId}/favorite`)
}

export function fetchComments(videoId, params) {
  return request.get(`/videos/${videoId}/comments`, { params })
}

export function postComment(videoId, data) {
  return request.post(`/videos/${videoId}/comments`, data)
}
