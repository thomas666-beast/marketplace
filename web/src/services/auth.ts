import { api } from './api'

export interface User {
  id: string
  email: string
  display_name: string
  role: 'buyer' | 'seller' | 'admin'
  preferred_locale: 'en' | 'ru' | 'es'
  email_verified: boolean
  created_at: string
}

export interface AuthResponse {
  user: User
  access_token: string
  refresh_token: string
  expires_in: number
}

export interface RegisterPayload {
  email: string
  password: string
  display_name: string
  role: 'buyer' | 'seller'
  preferred_locale: 'en' | 'ru' | 'es'
}

export interface LoginPayload {
  email: string
  password: string
}

export async function register(payload: RegisterPayload): Promise<AuthResponse> {
  const { data } = await api.post<AuthResponse>('/v1/auth/register', payload)
  return data
}

export async function login(payload: LoginPayload): Promise<AuthResponse> {
  const { data } = await api.post<AuthResponse>('/v1/auth/login', payload)
  return data
}

export async function refresh(refreshToken: string): Promise<{ access_token: string; expires_in: number }> {
  const { data } = await api.post<{ access_token: string; expires_in: number }>(
    '/v1/auth/refresh',
    { refresh_token: refreshToken },
  )
  return data
}

export async function me(): Promise<User> {
  const { data } = await api.get<User>('/v1/users/me')
  return data
}
