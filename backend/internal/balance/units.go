package balance

import (
	"fmt"
	"math"
)

func ValidateLevel(level, minimum, maximum float64) error {
	if !finite(level) || !finite(minimum) || !finite(maximum) {
		return fmt.Errorf("level boundary contains a non-finite value")
	}
	if maximum <= minimum {
		return fmt.Errorf("maximum level %.6f must exceed minimum level %.6f", maximum, minimum)
	}
	if level < minimum || level > maximum {
		return fmt.Errorf("liquid level %.6f m is outside [%.6f, %.6f] m", level, minimum, maximum)
	}
	return nil
}

func ValidateTemperature(value float64) error {
	if !finite(value) || value < -200 || value > -100 {
		return fmt.Errorf("liquid temperature %.6f C is outside LNG engineering range", value)
	}
	return nil
}

func ValidateDensity(value float64) error {
	if !finite(value) || value < 350 || value > 550 {
		return fmt.Errorf("density %.6f kg/m3 is outside configured LNG range", value)
	}
	return nil
}

func ValidateUncertainty(percent float64) error {
	if !finite(percent) || percent <= 0 || percent > 10 {
		return fmt.Errorf("measurement uncertainty %.6f%% must be within (0, 10]", percent)
	}
	return nil
}

func PercentFraction(percent float64) float64 { return percent / 100 }

func Round(value float64, places int) float64 {
	scale := math.Pow10(places)
	return math.Round(value*scale) / scale
}

func finite(value float64) bool { return !math.IsNaN(value) && !math.IsInf(value, 0) }
