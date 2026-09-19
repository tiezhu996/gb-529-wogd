import { request, requestPage } from './client'
import type { OperationStatus, TransferInput, TransferOperation } from '../types/transfer'

export const listTransfers = (tankId?: number) =>
  requestPage<TransferOperation>(`/transfers?page=1&page_size=100${tankId ? '&tank_id=' + tankId : ''}`)
export const getTransfer = (id: number) => request<TransferOperation>(`/transfers/${id}`)
export const createTransfer = (input: TransferInput) =>
  request<TransferOperation>('/transfers', { method: 'POST', body: JSON.stringify(input) })
export const transitionTransfer = (id: number, version: number, targetStatus: Extract<OperationStatus, 'confirmed' | 'cancelled'>, reason = '') =>
  request<TransferOperation>(`/transfers/${id}/status`, {
    method: 'POST',
    body: JSON.stringify({ version, target_status: targetStatus, reason })
  })
