import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import * as authApi from '@/services/auth'
import type { User, RegisterPayload, LoginPayload } from '@/services/auth'

const ACCESS_KEY = 'marketplace.access_token'
const REFRESH_KEY = 'marketplace.refresh_token'

export const useAuthStore = defineStore('auth', () => {
  const accessToken = ref<string | null>(localStorage.getItem(ACCESS_KEY))
  const refreshToken = ref<string | null>(localStorage.getItem(REFRESH_KEY))
  const user = ref<User | null>(null)
  const loading = ref(false)

  const isAuthenticated = computed(() => !!accessToken.value && !!user.value)
  const isSeller = computed(() => user.value?.role === 'seller' || user.value?.role === 'admin')
  const isAdmin = computed(() => user.value?.role === 'admin')

  function setTokens(access: string, refreshTokenValue: string) {
    accessToken.value = access
    refreshToken.value = refreshTokenValue
    localStorage.setItem(ACCESS_KEY, access)
    localStorage.setItem(REFRESH_KEY, refreshTokenValue)
  }

  function clearTokens() {
    accessToken.value = null
    refreshToken.value = null
    user.value = null
    localStorage.removeItem(ACCESS_KEY)
    localStorage.removeItem(REFRESH_KEY)
  }

  async function registerAction(payload: RegisterPayload) {
    loading.value = true
    try {
      const res = await authApi.register(payload)
      setTokens(res.access_token, res.refresh_token)
      user.value = res.user
      return res.user
    } finally {
      loading.value = false
    }
  }

  async function loginAction(payload: LoginPayload) {
    loading.value = true
    try {
      const res = await authApi.login(payload)
      setTokens(res.access_token, res.refresh_token)
      user.value = res.user
      return res.user
    } finally {
      loading.value = false
    }
  }

  async function loadCurrentUser() {
    if (!accessToken.value) return null
    try {
      user.value = await authApi.me()
      return user.value
    } catch {
      clearTokens()
      return null
    }
  }

  async function refreshAccessToken(): Promise<boolean> {
    if (!refreshToken.value) return false
    try {
      const res = await authApi.refresh(refreshToken.value)
      accessToken.value = res.access_token
      localStorage.setItem(ACCESS_KEY, res.access_token)
      return true
    } catch {
      clearTokens()
      return false
    }
  }

  function logout() {
    clearTokens()
  }

  return {
    accessToken,
    refreshToken,
    user,
    loading,
    isAuthenticated,
    isSeller,
    isAdmin,
    register: registerAction,
    login: loginAction,
    loadCurrentUser,
    refreshAccessToken,
    logout,
  }
})
