package util

import "math"

// JSONSafeNumber preserves non-finite diagnostics in JSON error details.
func JSONSafeNumber(value float64) any {
	switch {
	case math.IsNaN(value):
		return "NaN"
	case math.IsInf(value, 1):
		return "+Infinity"
	case math.IsInf(value, -1):
		return "-Infinity"
	default:
		return value
	}
}
