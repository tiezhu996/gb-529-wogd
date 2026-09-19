import { request } from './client'
import type { LoginResult, User } from '../types/auth'

export const login = (email: string, password: string) =>
  request<LoginResult>('/auth/login', { method: 'POST', body: JSON.stringify({ email, password }) })

export const me = () => request<User>('/auth/me')
