import type { StorageTank } from './tank'

export type QualityFlag = 'good' | 'suspect' | 'invalid'

export interface MeasurementSnapshot {
  id: number
  tank_id: number
  measured_at: string
  liquid_level_m: number
  liquid_temp_c: number
  vapor_pressure_kpa: number
  density_kgm3: number
  calculated_volume_m3: number
  temperature_density_kgm3: number
  calculated_liquid_mass_kg: number
  measurement_uncertainty_pct: number
  quality_flag: QualityFlag
  source_note: string
  created_by: number
  created_at: string
  tank?: StorageTank
}

export interface MeasurementInput {
  tank_id: number
  measured_at: string
  liquid_level_m: number
  liquid_temp_c: number
  vapor_pressure_kpa: number
  density_kgm3: number
  measurement_uncertainty_pct: number
  quality_flag: QualityFlag
  source_note: string
}
