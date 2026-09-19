export type TankStatus = 'active' | 'calibration_due' | 'inactive'

export interface CapacityCurve {
  coefficients: number[]
}

export interface StorageTank {
  id: number
  tank_code: string
  name: string
  nominal_capacity_m3: number
  min_level_m: number
  max_level_m: number
  reference_density_kgm3: number
  reference_temperature_c: number
  thermal_expansion_per_c: number
  capacity_curve_json: CapacityCurve
  coefficient_version: string
  tank_status: TankStatus
  version: number
  created_at: string
  updated_at: string
}

export interface TankInput {
  tank_code: string
  name: string
  nominal_capacity_m3: number
  min_level_m: number
  max_level_m: number
  reference_density_kgm3: number
  reference_temperature_c: number
  thermal_expansion_per_c: number
  capacity_curve: number[]
  coefficient_version: string
  tank_status: TankStatus
}

export interface TankMeasurementQuality {
  tank_id: number
  total: number
  good: number
  suspect: number
  invalid: number
  latest_snapshot?: unknown
  calculation_ready: boolean
}
