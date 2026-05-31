import request from './request'

export function fetchUser(id) {
  return request.get(`/users/${id}`)
}

export function fetchUserVideos(id, params) {
  return request.get(`/users/${id}/videos`, { params })
}

export function fetchUserLikedVideos(id, params) {
  return request.get(`/users/${id}/likes`, { params })
}

export function fetchUserFavoriteVideos(id, params) {
  return request.get(`/users/${id}/favorites`, { params })
}

export function uploadAvatar(file) {
  const form = new FormData()
  form.append('file', file)
  return request.post('/users/me/avatar', form, {
    headers: { 'Content-Type': 'multipart/form-data' },
    timeout: 30000
  })
}
