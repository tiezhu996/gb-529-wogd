package service

import (
	"testing"

	"lng-boiloff-gas-balance/backend/internal/balance"
)

func TestCapacityCurveRejectsPhysicalBoundaryViolation(t *testing.T) {
	curve, err := balance.NewCapacityCurve([]float64{0, 20000})
	if err != nil {
		t.Fatalf("new curve: %v", err)
	}
	if _, err := curve.VolumeAt(12, 0, 12, 180000); err == nil {
		t.Fatal("expected volume above nominal capacity to fail")
	}
}

func TestMeasurementInputErrorsRemainExplicit(t *testing.T) {
	curve, _ := balance.NewCapacityCurve([]float64{0, 15000})
	_, err := balance.CalculateSnapshotMass(balance.SnapshotMassInput{
		LevelM: 13, MinimumLevelM: 0, MaximumLevelM: 12, NominalCapacityM3: 180000,
		DensityKGM3: 451, TemperatureC: -160, ReferenceTemperatureC: -160,
		ThermalExpansionPerC: 0.0035, Curve: curve,
	})
	if err == nil {
		t.Fatal("expected out-of-range liquid level to fail")
	}
	if _, err := balance.PropagateUncertainty([]balance.UncertaintyInput{{MassKG: 1, UncertaintyPct: 0.2}}); err == nil {
		t.Fatal("expected incomplete uncertainty boundary to fail")
	}
}
