package constants

type DeviationLevel string

const (
	DeviationWithinUncertainty DeviationLevel = "within_uncertainty"
	DeviationWatch             DeviationLevel = "watch"
	DeviationInvestigate       DeviationLevel = "investigate"
	DeviationInvalid           DeviationLevel = "invalid"
)

var DeviationLevels = []DeviationLevel{
	DeviationWithinUncertainty, DeviationWatch, DeviationInvestigate, DeviationInvalid,
}

func ValidDeviationLevel(value DeviationLevel) bool {
	for _, level := range DeviationLevels {
		if level == value {
			return true
		}
	}
	return false
}

type QualityFlag string

const (
	QualityGood    QualityFlag = "good"
	QualitySuspect QualityFlag = "suspect"
	QualityInvalid QualityFlag = "invalid"
)

func ValidQualityFlag(value QualityFlag) bool {
	return value == QualityGood || value == QualitySuspect || value == QualityInvalid
}
