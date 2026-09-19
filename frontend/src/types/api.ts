export interface ApiEnvelope<T> {
  data: T
  request_id: string
  meta?: PageMeta
}

export interface PageMeta {
  page: number
  page_size: number
  total: number
}

export interface PageResult<T> {
  items: T[]
  meta: PageMeta
}

export interface ApiFailure {
  error?: {
    code?: string
    message?: string
    details?: Record<string, unknown>
  }
  request_id?: string
}
