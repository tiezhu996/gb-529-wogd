import type { DeviationLevel } from './deviation'
import type { StorageTank } from './tank'

export type BalanceStatus = 'queued' | 'calculating' | 'pending_review' | 'accepted' | 'rejected' | 'invalidated'

export interface BalanceRun {
  id: number
  tank_id: number
  period_start: string
  period_end: string
  balance_status: BalanceStatus
  input_snapshot_json: Record<string, unknown>
  opening_mass_kg: number
  closing_mass_kg: number
  net_transfer_kg: number
  estimated_bog_kg: number
  uncertainty_kg: number
  interval_lower_kg: number
  interval_upper_kg: number
  deviation_pct: number
  deviation_level: DeviationLevel
  evidence_json: BalanceEvidence
  coefficient_version: string
  version: number
  created_by: number
  reviewed_by?: number
  review_note: string
  reviewed_at?: string
  created_at: string
  updated_at: string
  tank?: StorageTank
}

export interface UncertaintyComponent {
  source: string
  entity_id: number
  mass_kg: number
  uncertainty_pct: number
  absolute_kg: number
}

export interface UncertaintyBreakdown {
  balance_run_id?: number
  combined_kg: number
  lower_kg: number
  upper_kg: number
  relationship: DeviationLevel
  components: UncertaintyComponent[]
}

export interface BalanceEvidence {
  algorithm_version?: string
  equation?: Record<string, number>
  uncertainty?: UncertaintyBreakdown
  safety_boundary?: string
}

export interface BalanceRunInput {
  tank_id: number
  period_start: string
  period_end: string
}
