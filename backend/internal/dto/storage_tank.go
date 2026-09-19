package dto

type CreateTankRequest struct {
	TankCode              string    `json:"tank_code" binding:"required,min=2,max=32"`
	Name                  string    `json:"name" binding:"required,min=2,max=120"`
	NominalCapacityM3     float64   `json:"nominal_capacity_m3" binding:"required,gt=0,lte=500000"`
	MinLevelM             float64   `json:"min_level_m" binding:"gte=0,lte=100"`
	MaxLevelM             float64   `json:"max_level_m" binding:"required,gt=0,lte=100"`
	ReferenceDensityKGM3  float64   `json:"reference_density_kgm3" binding:"required,gte=350,lte=550"`
	ReferenceTemperatureC float64   `json:"reference_temperature_c" binding:"gte=-200,lte=-100"`
	ThermalExpansionPerC  float64   `json:"thermal_expansion_per_c" binding:"required,gte=0.0001,lte=0.01"`
	CapacityCurve         []float64 `json:"capacity_curve" binding:"required,min=2,max=6,dive,gte=-1000000,lte=1000000"`
	CoefficientVersion    string    `json:"coefficient_version" binding:"required,min=2,max=32"`
	TankStatus            string    `json:"tank_status" binding:"required,oneof=active calibration_due inactive"`
}

type UpdateTankRequest struct {
	Name                  string    `json:"name" binding:"required,min=2,max=120"`
	NominalCapacityM3     float64   `json:"nominal_capacity_m3" binding:"required,gt=0,lte=500000"`
	MinLevelM             float64   `json:"min_level_m" binding:"gte=0,lte=100"`
	MaxLevelM             float64   `json:"max_level_m" binding:"required,gt=0,lte=100"`
	ReferenceDensityKGM3  float64   `json:"reference_density_kgm3" binding:"required,gte=350,lte=550"`
	ReferenceTemperatureC float64   `json:"reference_temperature_c" binding:"gte=-200,lte=-100"`
	ThermalExpansionPerC  float64   `json:"thermal_expansion_per_c" binding:"required,gte=0.0001,lte=0.01"`
	CapacityCurve         []float64 `json:"capacity_curve" binding:"required,min=2,max=6,dive,gte=-1000000,lte=1000000"`
	CoefficientVersion    string    `json:"coefficient_version" binding:"required,min=2,max=32"`
	TankStatus            string    `json:"tank_status" binding:"required,oneof=active calibration_due inactive"`
	Version               uint      `json:"version" binding:"required"`
}

type TankMeasurementQuality struct {
	TankID          uint  `json:"tank_id"`
	Total           int64 `json:"total"`
	Good            int64 `json:"good"`
	Suspect         int64 `json:"suspect"`
	Invalid         int64 `json:"invalid"`
	LatestSnapshot  any   `json:"latest_snapshot,omitempty"`
	CalculationRead bool  `json:"calculation_ready"`
}
