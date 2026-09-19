package dto

import "time"

type CreateMeasurementRequest struct {
	TankID                    uint       `json:"tank_id" binding:"required"`
	MeasuredAt                *time.Time `json:"measured_at" binding:"required"`
	LiquidLevelM              float64    `json:"liquid_level_m" binding:"gte=0,lte=100"`
	LiquidTempC               float64    `json:"liquid_temp_c" binding:"gte=-200,lte=-100"`
	VaporPressureKPA          float64    `json:"vapor_pressure_kpa" binding:"required,gte=0,lte=2000"`
	DensityKGM3               float64    `json:"density_kgm3" binding:"required,gte=350,lte=550"`
	MeasurementUncertaintyPct float64    `json:"measurement_uncertainty_pct" binding:"required,gt=0,lte=10"`
	QualityFlag               string     `json:"quality_flag" binding:"required,oneof=good suspect invalid"`
	SourceNote                string     `json:"source_note" binding:"required,min=3,max=500"`
}

type MeasurementValidation struct {
	Valid                  bool     `json:"valid"`
	Issues                 []string `json:"issues"`
	CalculatedVolumeM3     float64  `json:"calculated_volume_m3"`
	TemperatureDensityKGM3 float64  `json:"temperature_density_kgm3"`
	CalculatedMassKG       float64  `json:"calculated_mass_kg"`
}
