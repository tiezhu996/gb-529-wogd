package model

import "time"

type TransferOperation struct {
	ID                        uint         `json:"id" gorm:"primaryKey"`
	TankID                    uint         `json:"tank_id" gorm:"not null;index:idx_transfer_period"`
	OperationType             string       `json:"operation_type" gorm:"size:12;not null;check:operation_type IN ('inflow','outflow')"`
	StartAt                   time.Time    `json:"start_at" gorm:"not null;index:idx_transfer_period"`
	EndAt                     time.Time    `json:"end_at" gorm:"not null;index:idx_transfer_period"`
	MeasuredMassKG            float64      `json:"measured_mass_kg" gorm:"not null;check:measured_mass_kg > 0"`
	MeasurementUncertaintyPct float64      `json:"measurement_uncertainty_pct" gorm:"not null"`
	CounterpartyRef           string       `json:"counterparty_ref" gorm:"size:120;not null"`
	OperationStatus           string       `json:"operation_status" gorm:"size:16;not null;check:operation_status IN ('draft','confirmed','cancelled')"`
	Version                   uint         `json:"version" gorm:"not null;default:1"`
	CreatedBy                 uint         `json:"created_by" gorm:"not null"`
	CreatedAt                 time.Time    `json:"created_at"`
	UpdatedAt                 time.Time    `json:"updated_at"`
	Tank                      *StorageTank `json:"tank,omitempty" gorm:"foreignKey:TankID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
}

func (TransferOperation) TableName() string { return "transfer_operations" }
