package model

import (
	"time"

	"gorm.io/datatypes"

	"lng-boiloff-gas-balance/backend/internal/constants"
)

type BalanceRun struct {
	ID                 uint                     `json:"id" gorm:"primaryKey"`
	TankID             uint                     `json:"tank_id" gorm:"not null;index"`
	PeriodStart        time.Time                `json:"period_start" gorm:"not null;index"`
	PeriodEnd          time.Time                `json:"period_end" gorm:"not null;index"`
	BalanceStatus      constants.BalanceStatus  `json:"balance_status" gorm:"type:varchar(24);not null;check:balance_status IN ('queued','calculating','pending_review','accepted','rejected','invalidated')"`
	InputSnapshotJSON  datatypes.JSON           `json:"input_snapshot_json" gorm:"type:jsonb;not null"`
	OpeningMassKG      float64                  `json:"opening_mass_kg" gorm:"not null"`
	ClosingMassKG      float64                  `json:"closing_mass_kg" gorm:"not null"`
	NetTransferKG      float64                  `json:"net_transfer_kg" gorm:"not null"`
	EstimatedBOGKG     float64                  `json:"estimated_bog_kg" gorm:"column:estimated_bog_kg;not null"`
	UncertaintyKG      float64                  `json:"uncertainty_kg" gorm:"not null"`
	IntervalLowerKG    float64                  `json:"interval_lower_kg" gorm:"not null"`
	IntervalUpperKG    float64                  `json:"interval_upper_kg" gorm:"not null"`
	DeviationPct       float64                  `json:"deviation_pct" gorm:"not null"`
	DeviationLevel     constants.DeviationLevel `json:"deviation_level" gorm:"type:varchar(24);not null"`
	EvidenceJSON       datatypes.JSON           `json:"evidence_json" gorm:"type:jsonb;not null"`
	CoefficientVersion string                   `json:"coefficient_version" gorm:"size:32;not null"`
	Version            uint                     `json:"version" gorm:"not null;default:1"`
	CreatedBy          uint                     `json:"created_by" gorm:"not null"`
	ReviewedBy         *uint                    `json:"reviewed_by"`
	ReviewNote         string                   `json:"review_note" gorm:"size:1000"`
	ReviewedAt         *time.Time               `json:"reviewed_at"`
	CreatedAt          time.Time                `json:"created_at"`
	UpdatedAt          time.Time                `json:"updated_at"`
	Tank               *StorageTank             `json:"tank,omitempty" gorm:"foreignKey:TankID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
}

func (BalanceRun) TableName() string { return "balance_runs" }
