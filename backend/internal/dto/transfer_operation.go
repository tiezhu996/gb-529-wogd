package dto

import "time"

type CreateTransferRequest struct {
	TankID                    uint       `json:"tank_id" binding:"required"`
	OperationType             string     `json:"operation_type" binding:"required,oneof=inflow outflow"`
	StartAt                   *time.Time `json:"start_at" binding:"required"`
	EndAt                     *time.Time `json:"end_at" binding:"required"`
	MeasuredMassKG            float64    `json:"measured_mass_kg" binding:"required,gt=0,lte=1000000000"`
	MeasurementUncertaintyPct float64    `json:"measurement_uncertainty_pct" binding:"required,gt=0,lte=10"`
	CounterpartyRef           string     `json:"counterparty_ref" binding:"required,min=3,max=120"`
	OperationStatus           string     `json:"operation_status" binding:"required,oneof=draft confirmed"`
}

type TransitionTransferRequest struct {
	TargetStatus string `json:"target_status" binding:"required,oneof=confirmed cancelled"`
	Version      uint   `json:"version" binding:"required"`
	Reason       string `json:"reason" binding:"max=500"`
}
