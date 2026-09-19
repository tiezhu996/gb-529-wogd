import { useCallback, useEffect } from 'react'
import { useAuthStore } from '../stores/authStore'
import type { UserRole } from '../types/auth'

export function useAuth() {
  const state = useAuthStore()
  useEffect(() => {
    void state.bootstrap()
    const expire = () => state.logout()
    window.addEventListener('auth:expired', expire)
    return () => window.removeEventListener('auth:expired', expire)
  }, [state.bootstrap, state.logout])
  const can = useCallback((...roles: UserRole[]) => Boolean(state.user && roles.includes(state.user.role)), [state.user])
  return { ...state, can }
}
