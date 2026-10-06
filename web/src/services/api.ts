import axios, { type AxiosInstance, type AxiosError, type InternalAxiosRequestConfig } from 'axios'

export const api: AxiosInstance = axios.create({
  baseURL: '/api',
  timeout: 15000,
  headers: {
    'Content-Type': 'application/json',
  },
})

api.interceptors.request.use((config) => {
  const token = localStorage.getItem('marketplace.access_token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  const locale = localStorage.getItem('marketplace.locale') || 'en'
  config.headers['Accept-Language'] = locale
  return config
})

export interface ApiError {
  status: number
  error: string
  message: string
}

// --- Silent refresh on 401 ---

interface RetryConfig extends InternalAxiosRequestConfig {
  _retry?: boolean
}

let refreshPromise: Promise<string> | null = null

async function performRefresh(): Promise<string> {
  const refreshToken = localStorage.getItem('marketplace.refresh_token')
  if (!refreshToken) throw new Error('no refresh token')

  const { data } = await axios.post<{ access_token: string }>(
    '/api/v1/auth/refresh',
    { refresh_token: refreshToken },
    { headers: { 'Content-Type': 'application/json' } },
  )
  localStorage.setItem('marketplace.access_token', data.access_token)
  return data.access_token
}

api.interceptors.response.use(
  (resp) => resp,
  async (err: AxiosError<{ error?: string; message?: string }>) => {
    const original = err.config as RetryConfig | undefined
    const status = err.response?.status ?? 0

    // Attempt silent refresh once per request.
    if (status === 401 && original && !original._retry) {
      original._retry = true
      try {
        if (!refreshPromise) {
          refreshPromise = performRefresh().finally(() => {
            refreshPromise = null
          })
        }
        const newToken = await refreshPromise
        original.headers = original.headers || {}
        original.headers.Authorization = `Bearer ${newToken}`
        return api.request(original)
      } catch {
        localStorage.removeItem('marketplace.access_token')
        localStorage.removeItem('marketplace.refresh_token')
        // Fall through to normal error
      }
    }

    const normalized: ApiError = {
      status,
      error: err.response?.data?.error || 'unknown_error',
      message: err.response?.data?.message || err.message || 'Unknown error',
    }
    return Promise.reject(normalized)
  },
)
