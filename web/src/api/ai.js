import request from './request'

export function askAI(question) {
  return request.post('/ai/ask', { question })
}
