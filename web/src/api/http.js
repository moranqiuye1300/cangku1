import request from './request'
import { useAuthStore } from '../stores/auth'

export function setupHttp() {
  request.interceptors.request.use((config) => {
    const auth = useAuthStore()
    if (auth.token) {
      config.headers.Authorization = `Bearer ${auth.token}`
    }
    return config
  })

  request.interceptors.response.use(
    (response) => {
      const body = response.data
      if (body?.code !== 0) {
        return Promise.reject(new Error(body?.message || '请求失败'))
      }
      return body
    },
    (error) => {
      const message = error.response?.data?.message || error.message || '网络错误'
      if (error.response?.status === 401) {
        useAuthStore().logout()
      }
      return Promise.reject(new Error(message))
    }
  )
}
