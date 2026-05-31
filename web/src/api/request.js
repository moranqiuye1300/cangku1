import axios from 'axios'

const request = axios.create({
  baseURL: '/api/v1',
  timeout: 15000
})

export default request
