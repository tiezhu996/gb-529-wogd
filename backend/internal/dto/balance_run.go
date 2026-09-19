package dto

import (
	"time"

	"lng-boiloff-gas-balance/backend/internal/constants"
)

type RunBalanceRequest struct {
	TankID      uint       `json:"tank_id" binding:"required"`
	PeriodStart *time.Time `json:"period_start" binding:"required"`
	PeriodEnd   *time.Time `json:"period_end" binding:"required"`
}

type SubmitBalanceRequest struct {
	Version uint `json:"version" binding:"required"`
}

type ReviewBalanceRequest struct {
	TargetStatus constants.BalanceStatus `json:"target_status" binding:"required"`
	Version      uint                    `json:"version" binding:"required"`
	ReviewNote   string                  `json:"review_note" binding:"required,min=6,max=1000"`
}

type InvalidateBalanceRequest struct {
	Version uint   `json:"version" binding:"required"`
	Reason  string `json:"reason" binding:"required,min=6,max=1000"`
}

type UncertaintyComponent struct {
	Source         string  `json:"source"`
	EntityID       uint    `json:"entity_id"`
	MassKG         float64 `json:"mass_kg"`
	UncertaintyPct float64 `json:"uncertainty_pct"`
	AbsoluteKG     float64 `json:"absolute_kg"`
}

type UncertaintyBreakdown struct {
	BalanceRunID uint                     `json:"balance_run_id"`
	CombinedKG   float64                  `json:"combined_kg"`
	LowerKG      float64                  `json:"lower_kg"`
	UpperKG      float64                  `json:"upper_kg"`
	Relationship constants.DeviationLevel `json:"relationship"`
	Components   []UncertaintyComponent   `json:"components"`
}
