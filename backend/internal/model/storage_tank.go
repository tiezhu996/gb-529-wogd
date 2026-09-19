package model

import (
	"time"

	"gorm.io/datatypes"
)

type StorageTank struct {
	ID                    uint           `json:"id" gorm:"primaryKey"`
	TankCode              string         `json:"tank_code" gorm:"size:32;not null;uniqueIndex"`
	Name                  string         `json:"name" gorm:"size:120;not null"`
	NominalCapacityM3     float64        `json:"nominal_capacity_m3" gorm:"not null;check:nominal_capacity_m3 > 0"`
	MinLevelM             float64        `json:"min_level_m" gorm:"not null;check:min_level_m >= 0"`
	MaxLevelM             float64        `json:"max_level_m" gorm:"not null;check:max_level_m > min_level_m"`
	ReferenceDensityKGM3  float64        `json:"reference_density_kgm3" gorm:"not null;check:reference_density_kgm3 > 0"`
	ReferenceTemperatureC float64        `json:"reference_temperature_c" gorm:"not null"`
	ThermalExpansionPerC  float64        `json:"thermal_expansion_per_c" gorm:"not null;check:thermal_expansion_per_c >= 0"`
	CapacityCurveJSON     datatypes.JSON `json:"capacity_curve_json" gorm:"type:jsonb;not null"`
	CoefficientVersion    string         `json:"coefficient_version" gorm:"size:32;not null"`
	TankStatus            string         `json:"tank_status" gorm:"size:24;not null;check:tank_status IN ('active','calibration_due','inactive')"`
	Version               uint           `json:"version" gorm:"not null;default:1"`
	CreatedAt             time.Time      `json:"created_at"`
	UpdatedAt             time.Time      `json:"updated_at"`
}

func (StorageTank) TableName() string { return "storage_tanks" }
