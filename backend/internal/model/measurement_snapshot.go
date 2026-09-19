package model

import (
	"time"

	"lng-boiloff-gas-balance/backend/internal/constants"
)

type MeasurementSnapshot struct {
	ID                        uint                  `json:"id" gorm:"primaryKey"`
	TankID                    uint                  `json:"tank_id" gorm:"not null;index:idx_snapshot_boundary"`
	MeasuredAt                time.Time             `json:"measured_at" gorm:"not null;index:idx_snapshot_boundary"`
	LiquidLevelM              float64               `json:"liquid_level_m" gorm:"not null"`
	LiquidTempC               float64               `json:"liquid_temp_c" gorm:"not null"`
	VaporPressureKPA          float64               `json:"vapor_pressure_kpa" gorm:"not null"`
	DensityKGM3               float64               `json:"density_kgm3" gorm:"not null"`
	CalculatedVolumeM3        float64               `json:"calculated_volume_m3" gorm:"not null"`
	TemperatureDensityKGM3    float64               `json:"temperature_density_kgm3" gorm:"not null"`
	CalculatedLiquidMassKG    float64               `json:"calculated_liquid_mass_kg" gorm:"not null"`
	MeasurementUncertaintyPct float64               `json:"measurement_uncertainty_pct" gorm:"not null"`
	QualityFlag               constants.QualityFlag `json:"quality_flag" gorm:"type:varchar(16);not null;check:quality_flag IN ('good','suspect','invalid')"`
	SourceNote                string                `json:"source_note" gorm:"size:500;not null"`
	CreatedBy                 uint                  `json:"created_by" gorm:"not null"`
	CreatedAt                 time.Time             `json:"created_at"`
	Tank                      *StorageTank          `json:"tank,omitempty" gorm:"foreignKey:TankID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
}

func (MeasurementSnapshot) TableName() string { return "measurement_snapshots" }
