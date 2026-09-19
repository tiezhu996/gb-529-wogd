import { request, requestPage } from './client'
import type { MeasurementInput, MeasurementSnapshot } from '../types/measurement'

export const listMeasurements = (tankId?: number) =>
  requestPage<MeasurementSnapshot>(`/measurements?page=1&page_size=100${tankId ? '&tank_id=' + tankId : ''}`)
export const getMeasurement = (id: number) => request<MeasurementSnapshot>(`/measurements/${id}`)
export const createMeasurement = (input: MeasurementInput) =>
  request<MeasurementSnapshot>('/measurements', { method: 'POST', body: JSON.stringify(input) })
