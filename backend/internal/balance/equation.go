package balance

import (
	"fmt"
	"math"
)

type SnapshotMassInput struct {
	LevelM                float64
	MinimumLevelM         float64
	MaximumLevelM         float64
	NominalCapacityM3     float64
	DensityKGM3           float64
	TemperatureC          float64
	ReferenceTemperatureC float64
	ThermalExpansionPerC  float64
	Curve                 CapacityCurve
}

type SnapshotMassResult struct {
	VolumeM3             float64 `json:"volume_m3"`
	CorrectedDensityKGM3 float64 `json:"corrected_density_kgm3"`
	LiquidMassKG         float64 `json:"liquid_mass_kg"`
}

func CorrectDensity(density, temperature, referenceTemperature, expansion float64) (float64, error) {
	if err := ValidateDensity(density); err != nil {
		return 0, err
	}
	if err := ValidateTemperature(temperature); err != nil {
		return 0, err
	}
	if expansion < 0 || expansion > 0.01 || !finite(expansion) {
		return 0, fmt.Errorf("thermal expansion coefficient %.8f is outside [0, 0.01]", expansion)
	}
	denominator := 1 + expansion*(temperature-referenceTemperature)
	if denominator <= 0.5 || denominator >= 1.5 {
		return 0, fmt.Errorf("temperature correction denominator %.6f is not physically usable", denominator)
	}
	corrected := density / denominator
	if !finite(corrected) || corrected <= 0 {
		return 0, fmt.Errorf("temperature correction produced an invalid density")
	}
	return Round(corrected, 6), nil
}

func CalculateSnapshotMass(input SnapshotMassInput) (SnapshotMassResult, error) {
	volume, err := input.Curve.VolumeAt(input.LevelM, input.MinimumLevelM, input.MaximumLevelM, input.NominalCapacityM3)
	if err != nil {
		return SnapshotMassResult{}, fmt.Errorf("calculate calibrated volume: %w", err)
	}
	density, err := CorrectDensity(input.DensityKGM3, input.TemperatureC, input.ReferenceTemperatureC, input.ThermalExpansionPerC)
	if err != nil {
		return SnapshotMassResult{}, fmt.Errorf("calculate temperature-corrected density: %w", err)
	}
	mass := volume * density
	if !finite(mass) || mass < 0 {
		return SnapshotMassResult{}, fmt.Errorf("calculated liquid mass is invalid")
	}
	return SnapshotMassResult{VolumeM3: volume, CorrectedDensityKGM3: density, LiquidMassKG: Round(mass, 3)}, nil
}

func NetTransfer(inflows, outflows []float64) (float64, error) {
	total := 0.0
	for _, value := range inflows {
		if value < 0 || !finite(value) {
			return 0, fmt.Errorf("inflow mass must be finite and non-negative")
		}
		total += value
	}
	for _, value := range outflows {
		if value < 0 || !finite(value) {
			return 0, fmt.Errorf("outflow mass must be finite and non-negative")
		}
		total -= value
	}
	return Round(total, 3), nil
}

func PhysicalBalance(openingMass, netTransfer, closingMass float64) (float64, error) {
	if openingMass < 0 || closingMass < 0 || !finite(openingMass) || !finite(closingMass) || !finite(netTransfer) {
		return 0, fmt.Errorf("mass balance inputs must be finite and boundary masses non-negative")
	}
	return Round(openingMass+netTransfer-closingMass, 3), nil
}

func DeviationPercent(deviation, openingMass float64) float64 {
	if math.Abs(openingMass) < 1e-9 {
		return 0
	}
	return Round(deviation/openingMass*100, 6)
}
