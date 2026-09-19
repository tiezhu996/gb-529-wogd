package service

import (
	"testing"
	"time"

	"gorm.io/datatypes"

	"lng-boiloff-gas-balance/backend/internal/balance"
	"lng-boiloff-gas-balance/backend/internal/constants"
	"lng-boiloff-gas-balance/backend/internal/model"
)

func TestCalculateBalanceRunProducesReplayEvidence(t *testing.T) {
	curve, _ := balance.NewCapacityCurve([]float64{0, 15000})
	raw, _ := curve.Marshal()
	tank := model.StorageTank{
		ID: 1, TankCode: "TK-TEST", NominalCapacityM3: 180000, MinLevelM: 0, MaxLevelM: 12,
		ReferenceDensityKGM3: 452, ReferenceTemperatureC: -160, ThermalExpansionPerC: 0.0035,
		CapacityCurveJSON: datatypes.JSON(raw), CoefficientVersion: "CV-T1", TankStatus: "active",
	}
	start := time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)
	opening := model.MeasurementSnapshot{
		ID: 10, TankID: 1, MeasuredAt: start.Add(-time.Hour), CalculatedLiquidMassKG: 55_000_000,
		MeasurementUncertaintyPct: 0.30, QualityFlag: constants.QualityGood,
	}
	closing := model.MeasurementSnapshot{
		ID: 11, TankID: 1, MeasuredAt: end.Add(-time.Hour), CalculatedLiquidMassKG: 55_100_000,
		MeasurementUncertaintyPct: 0.32, QualityFlag: constants.QualityGood,
	}
	transfers := []model.TransferOperation{
		{ID: 20, OperationType: "inflow", MeasuredMassKG: 250000, MeasurementUncertaintyPct: 0.2},
		{ID: 21, OperationType: "outflow", MeasuredMassKG: 90000, MeasurementUncertaintyPct: 0.25},
	}
	calculated, snapshot, evidence, err := calculateBalanceRun(tank, opening, closing, transfers, start, end)
	if err != nil {
		t.Fatalf("calculate run: %v", err)
	}
	if calculated.NetTransferKG != 160000 || calculated.EstimatedBOGKG != 60000 {
		t.Fatalf("unexpected equation result: %+v", calculated)
	}
	if len(snapshot) < 200 || len(evidence) < 200 {
		t.Fatalf("expected replay snapshot and evidence, got %d/%d bytes", len(snapshot), len(evidence))
	}
	if calculated.DeviationLevel != constants.DeviationWithinUncertainty {
		t.Fatalf("unexpected deviation level: %s", calculated.DeviationLevel)
	}
}

func TestBalanceStateMachine(t *testing.T) {
	valid := []struct {
		from constants.BalanceStatus
		to   constants.BalanceStatus
	}{
		{constants.BalanceQueued, constants.BalanceCalculating},
		{constants.BalanceCalculating, constants.BalancePendingReview},
		{constants.BalancePendingReview, constants.BalanceAccepted},
		{constants.BalancePendingReview, constants.BalanceRejected},
	}
	for _, transition := range valid {
		if !constants.CanTransitionBalance(transition.from, transition.to) {
			t.Fatalf("expected valid transition %s -> %s", transition.from, transition.to)
		}
	}
	if constants.CanTransitionBalance(constants.BalanceAccepted, constants.BalanceCalculating) {
		t.Fatal("accepted result must remain immutable")
	}
}
