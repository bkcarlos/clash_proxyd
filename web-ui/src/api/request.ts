import axios from 'axios'
import type { AxiosInstance, AxiosResponse } from 'axios'
import { ElMessage } from 'element-plus'

const baseURL = '/api/v1'

// Create axios instance
const request: AxiosInstance = axios.create({
  baseURL,
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json'
  }
})

// Request interceptor
request.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// Response interceptor
request.interceptors.response.use(
  (response: AxiosResponse) => {
    return response.data
  },
  (error) => {
    const message = error.response?.data?.error || error.message || 'Request failed'

    if (error.response?.status === 401) {
      localStorage.removeItem('token')
      localStorage.removeItem('expiresAt')
      localStorage.removeItem('username')
      window.location.href = '/login'
      return Promise.reject(error)
    }

    if (error.response?.status === 403) {
      ElMessage.error('Access denied')
    } else if (error.response?.status >= 500) {
      ElMessage.error('Server error')
    } else if (error.response?.status !== 401) {
      ElMessage.error(message)
    }

    return Promise.reject(error)
  }
)

export default request

export const downloadRequest = (url: string, filename: string) => {
  return axios
    .get(url, {
      baseURL,
      responseType: 'blob',
      headers: {
        Authorization: localStorage.getItem('token') ? `Bearer ${localStorage.getItem('token')}` : undefined
      }
    })
    .then((response) => {
      const blob = new Blob([response.data])
      const link = document.createElement('a')
      link.href = URL.createObjectURL(blob)
      link.download = filename
      link.click()
      URL.revokeObjectURL(link.href)
    })
}
