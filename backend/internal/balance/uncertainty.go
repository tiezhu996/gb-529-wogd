package balance

import (
	"fmt"
	"math"

	"lng-boiloff-gas-balance/backend/internal/constants"
)

type UncertaintyInput struct {
	Source         string
	EntityID       uint
	MassKG         float64
	UncertaintyPct float64
}

type UncertaintyResult struct {
	CombinedKG float64
	Components []float64
}

func PropagateUncertainty(inputs []UncertaintyInput) (UncertaintyResult, error) {
	if len(inputs) < 2 {
		return UncertaintyResult{}, fmt.Errorf("at least opening and closing uncertainty inputs are required")
	}
	sumSquares := 0.0
	components := make([]float64, 0, len(inputs))
	for index, input := range inputs {
		if input.MassKG < 0 || !finite(input.MassKG) {
			return UncertaintyResult{}, fmt.Errorf("uncertainty input %d mass is invalid", index)
		}
		if err := ValidateUncertainty(input.UncertaintyPct); err != nil {
			return UncertaintyResult{}, fmt.Errorf("uncertainty input %d: %w", index, err)
		}
		absolute := input.MassKG * PercentFraction(input.UncertaintyPct)
		components = append(components, Round(absolute, 3))
		sumSquares += absolute * absolute
	}
	return UncertaintyResult{CombinedKG: Round(math.Sqrt(sumSquares), 3), Components: components}, nil
}

func ClassifyDeviation(deviation, uncertainty float64, valid bool) constants.DeviationLevel {
	if !valid || uncertainty <= 0 || !finite(deviation) || !finite(uncertainty) {
		return constants.DeviationInvalid
	}
	ratio := math.Abs(deviation) / uncertainty
	switch {
	case ratio <= 1:
		return constants.DeviationWithinUncertainty
	case ratio <= 2:
		return constants.DeviationWatch
	default:
		return constants.DeviationInvestigate
	}
}

func ConfidenceInterval(deviation, uncertainty float64) (float64, float64) {
	return Round(deviation-uncertainty, 3), Round(deviation+uncertainty, 3)
}
