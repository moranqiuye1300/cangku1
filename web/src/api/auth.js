import request from './request'

export function login(data) {
  return request.post('/auth/login', data)
}

export function oauthMock(params) {
  return request.get('/auth/oauth/mock', { params })
}

export function register(data) {
  return request.post('/auth/register', data)
}
