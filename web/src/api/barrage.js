import request from './request'

export function fetchBarrages(videoId) {
  return request.get(`/videos/${videoId}/barrages`)
}

export function postBarrage(videoId, data) {
  return request.post(`/videos/${videoId}/barrages`, data)
}
