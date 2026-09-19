import { request, requestPage } from './client'
import type { BalanceRun, BalanceRunInput, BalanceStatus, UncertaintyBreakdown } from '../types/balance'

export const listBalances = (tankId?: number) =>
  requestPage<BalanceRun>(`/balances?page=1&page_size=100${tankId ? '&tank_id=' + tankId : ''}`)
export const getBalance = (id: number) => request<BalanceRun>(`/balances/${id}`)
export const runBalance = (input: BalanceRunInput) =>
  request<BalanceRun>('/balances/run', { method: 'POST', body: JSON.stringify(input) })
export const submitBalance = (id: number, version: number) =>
  request<BalanceRun>(`/balances/${id}/submit`, { method: 'POST', body: JSON.stringify({ version }) })
export const reviewBalance = (id: number, version: number, targetStatus: Extract<BalanceStatus, 'accepted' | 'rejected'>, reviewNote: string) =>
  request<BalanceRun>(`/balances/${id}/review`, {
    method: 'POST',
    body: JSON.stringify({ version, target_status: targetStatus, review_note: reviewNote })
  })
export const invalidateBalance = (id: number, version: number, reason: string) =>
  request<BalanceRun>(`/balances/${id}/invalidate`, { method: 'POST', body: JSON.stringify({ version, reason }) })
export const getUncertainty = (id: number) => request<UncertaintyBreakdown>(`/balances/${id}/uncertainty`)
