import { request, requestPage } from './client'
import type { StorageTank, TankInput, TankMeasurementQuality } from '../types/tank'

export const listTanks = () => requestPage<StorageTank>('/tanks?page=1&page_size=100')
export const getTank = (id: number) => request<StorageTank>(`/tanks/${id}`)
export const createTank = (input: TankInput) => request<StorageTank>('/tanks', { method: 'POST', body: JSON.stringify(input) })
export const updateTank = (id: number, input: TankInput & { version: number }) =>
  request<StorageTank>(`/tanks/${id}`, { method: 'PUT', body: JSON.stringify(input) })
export const getTankQuality = (id: number) => request<TankMeasurementQuality>(`/tanks/${id}/measurement-quality`)
