import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { LoginRequest } from '@/api/auth'
import * as authApi from '@/api/auth'

export const useUserStore = defineStore('user', () => {
  const token = ref<string>(localStorage.getItem('token') || '')
  const username = ref<string>(localStorage.getItem('username') || '')
  const expiresAt = ref<number>(parseInt(localStorage.getItem('expiresAt') || '0'))

  const isLoggedIn = computed(() => {
    return token.value && expiresAt.value > Date.now() / 1000
  })

  const login = async (credentials: LoginRequest): Promise<void> => {
    try {
      const response = await authApi.login(credentials)
      token.value = response.token
      expiresAt.value = response.expires_at
      username.value = credentials.username

      localStorage.setItem('token', response.token)
      localStorage.setItem('expiresAt', response.expires_at.toString())
      localStorage.setItem('username', credentials.username)
    } catch (error) {
      throw error
    }
  }

  const logout = async (): Promise<void> => {
    try {
      await authApi.logout()
    } finally {
      token.value = ''
      username.value = ''
      expiresAt.value = 0
      localStorage.removeItem('token')
      localStorage.removeItem('expiresAt')
      localStorage.removeItem('username')
    }
  }

  const refreshToken = async (): Promise<void> => {
    try {
      const response = await authApi.refreshToken()
      token.value = response.token
      expiresAt.value = response.expires_at
      localStorage.setItem('token', response.token)
      localStorage.setItem('expiresAt', response.expires_at.toString())
    } catch (error) {
      await logout()
      throw error
    }
  }

  return {
    token,
    username,
    expiresAt,
    isLoggedIn,
    login,
    logout,
    refreshToken
  }
})
