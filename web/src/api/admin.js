import request from './request'

export function adminLogin(data) {
  return request.post('/admin/login', data)
}

export function listUsers(params) {
  return request.get('/admin/users', { params })
}

export function setUserRole(userId, role) {
  return request.patch(`/admin/users/${userId}/role`, { role })
}

export function adminListVideos(params) {
  return request.get('/admin/videos', { params })
}

export function adminDeleteVideo(id, reason = '') {
  return request.delete(`/admin/videos/${id}`, { data: { reason } })
}

export function adminRestoreVideo(id) {
  return request.post(`/admin/videos/${id}/restore`)
}

export function adminPermanentDeleteVideo(id) {
  return request.delete(`/admin/videos/${id}/permanent`)
}

export function listRecycleBin(params) {
  return request.get('/admin/recycle-bin', { params })
}

export function listAuditLogs(params) {
  return request.get('/admin/audit-logs', { params })
}

export function reviewerListVideos(params) {
  return request.get('/reviewer/videos', { params })
}

export function reviewerListPending(stage, params = {}) {
  return request.get('/reviewer/videos', { params: { stage, ...params } })
}

export function approveSource(id) {
  return request.post(`/reviewer/videos/${id}/approve-source`)
}

export function rejectSource(id, reason) {
  return request.post(`/reviewer/videos/${id}/reject-source`, { reason })
}

export function approvePublish(id) {
  return request.post(`/reviewer/videos/${id}/approve-publish`)
}

export function rejectPublish(id, reason) {
  return request.post(`/reviewer/videos/${id}/reject-publish`, { reason })
}

export function reviewerRejectVideo(id, reason) {
  return request.post(`/reviewer/videos/${id}/reject`, { reason })
}
