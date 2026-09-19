package balance

import (
	"encoding/json"
	"fmt"
	"math"
)

type CapacityCurve struct {
	Coefficients []float64 `json:"coefficients"`
}

func ParseCapacityCurve(raw []byte) (CapacityCurve, error) {
	var curve CapacityCurve
	if err := json.Unmarshal(raw, &curve); err != nil {
		return CapacityCurve{}, fmt.Errorf("decode capacity curve: %w", err)
	}
	if err := curve.Validate(); err != nil {
		return CapacityCurve{}, err
	}
	return curve, nil
}

func NewCapacityCurve(coefficients []float64) (CapacityCurve, error) {
	curve := CapacityCurve{Coefficients: append([]float64(nil), coefficients...)}
	if err := curve.Validate(); err != nil {
		return CapacityCurve{}, err
	}
	return curve, nil
}

func (c CapacityCurve) Validate() error {
	if len(c.Coefficients) < 2 || len(c.Coefficients) > 6 {
		return fmt.Errorf("capacity curve requires 2 to 6 polynomial coefficients")
	}
	for index, coefficient := range c.Coefficients {
		if !finite(coefficient) {
			return fmt.Errorf("capacity coefficient %d is not finite", index)
		}
	}
	return nil
}

func (c CapacityCurve) Marshal() ([]byte, error) {
	data, err := json.Marshal(c)
	if err != nil {
		return nil, fmt.Errorf("encode capacity curve: %w", err)
	}
	return data, nil
}

func (c CapacityCurve) VolumeAt(level, minimum, maximum, nominal float64) (float64, error) {
	if err := ValidateLevel(level, minimum, maximum); err != nil {
		return 0, err
	}
	if nominal <= 0 || !finite(nominal) {
		return 0, fmt.Errorf("nominal capacity must be positive")
	}
	volume := 0.0
	for index := len(c.Coefficients) - 1; index >= 0; index-- {
		volume = volume*level + c.Coefficients[index]
	}
	if !finite(volume) || volume < 0 || volume > nominal*1.001 {
		return 0, fmt.Errorf("capacity curve produced %.6f m3 outside physical boundary [0, %.6f]", volume, nominal)
	}
	return Round(math.Min(volume, nominal), 6), nil
}
