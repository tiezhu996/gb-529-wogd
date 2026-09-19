import { create } from 'zustand'
import * as authApi from '../api/auth'
import { authTokenKey } from '../api/client'
import type { User } from '../types/auth'

interface AuthState {
  user: User | null
  token: string | null
  loading: boolean
  initialized: boolean
  login: (email: string, password: string) => Promise<void>
  bootstrap: () => Promise<void>
  logout: () => void
}

export const useAuthStore = create<AuthState>((set, get) => ({
  user: null,
  token: sessionStorage.getItem(authTokenKey),
  loading: false,
  initialized: false,
  login: async (email, password) => {
    set({ loading: true })
    try {
      const result = await authApi.login(email, password)
      sessionStorage.setItem(authTokenKey, result.token)
      set({ token: result.token, user: result.user, initialized: true })
    } finally {
      set({ loading: false })
    }
  },
  bootstrap: async () => {
    if (get().initialized) return
    const token = sessionStorage.getItem(authTokenKey)
    if (!token) {
      set({ initialized: true, token: null, user: null })
      return
    }
    set({ loading: true })
    try {
      const user = await authApi.me()
      set({ user, token, initialized: true })
    } catch {
      sessionStorage.removeItem(authTokenKey)
      set({ user: null, token: null, initialized: true })
    } finally {
      set({ loading: false })
    }
  },
  logout: () => {
    sessionStorage.removeItem(authTokenKey)
    set({ user: null, token: null, initialized: true })
  }
}))
