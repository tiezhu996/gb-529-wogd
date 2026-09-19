export type UserRole = 'process_analyst' | 'reviewer' | 'admin'

export interface User {
  id: number
  email: string
  display_name: string
  role: UserRole
  active: boolean
}

export interface LoginResult {
  token: string
  user: User
}
