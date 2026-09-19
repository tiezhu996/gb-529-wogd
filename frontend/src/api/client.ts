import type { ApiEnvelope, ApiFailure, PageResult } from '../types/api'

const tokenKey = 'lng_balance_token'

export class ApiError extends Error {
  constructor(
    message: string,
    public readonly status: number,
    public readonly code: string,
    public readonly requestId?: string,
    public readonly details?: Record<string, unknown>
  ) {
    super(message)
  }
}

export interface PeriodBoundaryViolation {
  transfer_id: number
  operation_type: string
  crossed_boundary: 'period_start' | 'period_end'
  boundary_at: string
  out_of_boundary_at: string
}

export function periodViolations(error: unknown): PeriodBoundaryViolation[] {
  if (!(error instanceof ApiError) || !error.details) return []
  const raw = error.details.violations
  return Array.isArray(raw) ? raw as PeriodBoundaryViolation[] : []
}

export async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
  const token = sessionStorage.getItem(tokenKey)
  const headers = new Headers(init.headers)
  if (init.body) headers.set('Content-Type', 'application/json')
  if (token) headers.set('Authorization', `Bearer ${token}`)
  headers.set('X-Request-ID', crypto.randomUUID())
  let response: Response
  try {
    response = await fetch(`/api/v1${path}`, { ...init, headers })
  } catch {
    const error = new ApiError('无法连接分析服务，请检查服务状态', 0, 'NETWORK_ERROR')
    window.dispatchEvent(new CustomEvent('api:error', { detail: error.message }))
    throw error
  }
  const payload = (await response.json().catch(() => ({}))) as ApiEnvelope<T> & ApiFailure
  if (!response.ok) {
    const error = new ApiError(
      payload.error?.message ?? `请求失败 (${response.status})`,
      response.status,
      payload.error?.code ?? 'REQUEST_FAILED',
      payload.request_id,
      payload.error?.details as Record<string, unknown> | undefined
    )
    if (response.status === 401 && path !== '/auth/login') {
      sessionStorage.removeItem(tokenKey)
      window.dispatchEvent(new Event('auth:expired'))
    }
    window.dispatchEvent(new CustomEvent('api:error', { detail: error.message + (error.requestId ? ' · ' + error.requestId : '') }))
    throw error
  }
  return payload.data
}

export async function requestPage<T>(path: string): Promise<PageResult<T>> {
  const token = sessionStorage.getItem(tokenKey)
  const headers = new Headers({ 'X-Request-ID': crypto.randomUUID() })
  if (token) headers.set('Authorization', `Bearer ${token}`)
  const response = await fetch(`/api/v1${path}`, { headers })
  const payload = (await response.json().catch(() => ({}))) as ApiEnvelope<T[]> & ApiFailure
  if (!response.ok) {
    const error = new ApiError(payload.error?.message ?? '列表请求失败', response.status, payload.error?.code ?? 'REQUEST_FAILED', payload.request_id)
    window.dispatchEvent(new CustomEvent('api:error', { detail: error.message }))
    throw error
  }
  return {
    items: payload.data,
    meta: payload.meta ?? { page: 1, page_size: payload.data.length, total: payload.data.length }
  }
}

export const authTokenKey = tokenKey
