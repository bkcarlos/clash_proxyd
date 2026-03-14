import request from './request'

export interface LoginRequest {
  username: string
  password: string
}

export interface LoginResponse {
  token: string
  expires_at: number
}

export const login = (data: LoginRequest): Promise<LoginResponse> => {
  return request({
    url: '/auth/login',
    method: 'POST',
    data
  })
}

export const logout = (): Promise<void> => {
  return request({
    url: '/auth/logout',
    method: 'POST'
  })
}

export const refreshToken = (): Promise<LoginResponse> => {
  return request({
    url: '/auth/refresh',
    method: 'POST'
  })
}

export const getProfile = (): Promise<any> => {
  return request({
    url: '/auth/profile',
    method: 'GET'
  })
}

export const updatePassword = (oldPassword: string, newPassword: string): Promise<void> => {
  return request({
    url: '/auth/password',
    method: 'PUT',
    data: {
      old_password: oldPassword,
      new_password: newPassword
    }
  })
}
