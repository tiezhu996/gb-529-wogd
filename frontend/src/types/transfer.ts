import type { StorageTank } from './tank'

export type OperationType = 'inflow' | 'outflow'
export type OperationStatus = 'draft' | 'confirmed' | 'cancelled'

export interface TransferOperation {
  id: number
  tank_id: number
  operation_type: OperationType
  start_at: string
  end_at: string
  measured_mass_kg: number
  measurement_uncertainty_pct: number
  counterparty_ref: string
  operation_status: OperationStatus
  version: number
  created_by: number
  created_at: string
  updated_at: string
  tank?: StorageTank
}

export interface TransferInput {
  tank_id: number
  operation_type: OperationType
  start_at: string
  end_at: string
  measured_mass_kg: number
  measurement_uncertainty_pct: number
  counterparty_ref: string
  operation_status: Extract<OperationStatus, 'draft' | 'confirmed'>
}
