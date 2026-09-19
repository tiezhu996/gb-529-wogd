package balance

import (
	"math"
	"testing"

	"lng-boiloff-gas-balance/backend/internal/constants"
)

func TestCapacityCurveAndSnapshotMass(t *testing.T) {
	curve, err := NewCapacityCurve([]float64{0, 15000})
	if err != nil {
		t.Fatalf("new curve: %v", err)
	}
	result, err := CalculateSnapshotMass(SnapshotMassInput{
		LevelM: 8.2, MinimumLevelM: 0, MaximumLevelM: 12, NominalCapacityM3: 180000,
		DensityKGM3: 451.8, TemperatureC: -160.4, ReferenceTemperatureC: -160,
		ThermalExpansionPerC: 0.0035, Curve: curve,
	})
	if err != nil {
		t.Fatalf("calculate snapshot mass: %v", err)
	}
	if result.VolumeM3 != 123000 {
		t.Fatalf("unexpected calibrated volume: %.6f", result.VolumeM3)
	}
	if math.Abs(result.CorrectedDensityKGM3-452.433) > 0.01 {
		t.Fatalf("unexpected corrected density: %.6f", result.CorrectedDensityKGM3)
	}
	if result.LiquidMassKG <= 55_000_000 {
		t.Fatalf("unexpected liquid mass: %.3f", result.LiquidMassKG)
	}
}

func TestPhysicalBalanceAndUncertainty(t *testing.T) {
	net, err := NetTransfer([]float64{425000}, []float64{96500})
	if err != nil {
		t.Fatalf("net transfer: %v", err)
	}
	if net != 328500 {
		t.Fatalf("unexpected net transfer %.3f", net)
	}
	deviation, err := PhysicalBalance(55_000_000, net, 55_250_000)
	if err != nil {
		t.Fatalf("physical balance: %v", err)
	}
	if deviation != 78500 {
		t.Fatalf("unexpected deviation %.3f", deviation)
	}
	propagated, err := PropagateUncertainty([]UncertaintyInput{
		{MassKG: 55_000_000, UncertaintyPct: 0.28},
		{MassKG: 55_250_000, UncertaintyPct: 0.30},
		{MassKG: 425000, UncertaintyPct: 0.22},
		{MassKG: 96500, UncertaintyPct: 0.25},
	})
	if err != nil {
		t.Fatalf("uncertainty: %v", err)
	}
	if propagated.CombinedKG <= 200000 {
		t.Fatalf("unexpected combined uncertainty %.3f", propagated.CombinedKG)
	}
	if level := ClassifyDeviation(deviation, propagated.CombinedKG, true); level != constants.DeviationWithinUncertainty {
		t.Fatalf("expected within uncertainty, got %s", level)
	}
}

func TestDeviationClassificationBoundaries(t *testing.T) {
	tests := []struct {
		deviation   float64
		uncertainty float64
		valid       bool
		want        constants.DeviationLevel
	}{
		{50, 100, true, constants.DeviationWithinUncertainty},
		{150, 100, true, constants.DeviationWatch},
		{250, 100, true, constants.DeviationInvestigate},
		{50, 100, false, constants.DeviationInvalid},
	}
	for _, test := range tests {
		if got := ClassifyDeviation(test.deviation, test.uncertainty, test.valid); got != test.want {
			t.Fatalf("classify %.2f/%.2f valid=%v: got %s want %s", test.deviation, test.uncertainty, test.valid, got, test.want)
		}
	}
}
