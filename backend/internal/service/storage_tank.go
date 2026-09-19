package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"gorm.io/datatypes"

	"lng-boiloff-gas-balance/backend/internal/balance"
	"lng-boiloff-gas-balance/backend/internal/constants"
	"lng-boiloff-gas-balance/backend/internal/dto"
	"lng-boiloff-gas-balance/backend/internal/model"
	"lng-boiloff-gas-balance/backend/internal/repository"
	"lng-boiloff-gas-balance/backend/pkg/api"
)

type TankService struct {
	repo *repository.TankRepository
}

func NewTankService(repo *repository.TankRepository) *TankService { return &TankService{repo: repo} }

func (s *TankService) List(ctx context.Context, page, pageSize int, status string) ([]model.StorageTank, int64, error) {
	if status != "" && status != "active" && status != "calibration_due" && status != "inactive" {
		return nil, 0, api.NewError(400, "INVALID_TANK_STATUS", "储罐状态筛选值无效")
	}
	return s.repo.List(ctx, page, pageSize, status)
}

func (s *TankService) Get(ctx context.Context, id uint) (model.StorageTank, error) {
	return s.repo.Get(ctx, id)
}

func (s *TankService) Create(ctx context.Context, request dto.CreateTankRequest, actor repository.Actor) (model.StorageTank, error) {
	if !constants.CanAnalyze(actor.Role) {
		return model.StorageTank{}, api.ErrForbidden
	}
	curveJSON, err := validateAndMarshalTank(request.MinLevelM, request.MaxLevelM, request.NominalCapacityM3, request.CapacityCurve)
	if err != nil {
		return model.StorageTank{}, err
	}
	tank := model.StorageTank{
		TankCode:              strings.ToUpper(strings.TrimSpace(request.TankCode)),
		Name:                  strings.TrimSpace(request.Name),
		NominalCapacityM3:     request.NominalCapacityM3,
		MinLevelM:             request.MinLevelM,
		MaxLevelM:             request.MaxLevelM,
		ReferenceDensityKGM3:  request.ReferenceDensityKGM3,
		ReferenceTemperatureC: request.ReferenceTemperatureC,
		ThermalExpansionPerC:  request.ThermalExpansionPerC,
		CapacityCurveJSON:     datatypes.JSON(curveJSON),
		CoefficientVersion:    strings.TrimSpace(request.CoefficientVersion),
		TankStatus:            request.TankStatus,
		Version:               1,
	}
	if err := s.repo.Create(ctx, &tank, actor); err != nil {
		return model.StorageTank{}, err
	}
	return tank, nil
}

func (s *TankService) Update(ctx context.Context, id uint, request dto.UpdateTankRequest, actor repository.Actor) (model.StorageTank, error) {
	if !constants.CanAnalyze(actor.Role) {
		return model.StorageTank{}, api.ErrForbidden
	}
	before, err := s.repo.Get(ctx, id)
	if err != nil {
		return model.StorageTank{}, err
	}
	if before.Version != request.Version {
		return model.StorageTank{}, api.NewError(409, "TANK_VERSION_CONFLICT", "储罐参数版本已变化，请刷新后重试")
	}
	curveJSON, err := validateAndMarshalTank(request.MinLevelM, request.MaxLevelM, request.NominalCapacityM3, request.CapacityCurve)
	if err != nil {
		return model.StorageTank{}, err
	}
	updated := before
	updated.Name = strings.TrimSpace(request.Name)
	updated.NominalCapacityM3 = request.NominalCapacityM3
	updated.MinLevelM = request.MinLevelM
	updated.MaxLevelM = request.MaxLevelM
	updated.ReferenceDensityKGM3 = request.ReferenceDensityKGM3
	updated.ReferenceTemperatureC = request.ReferenceTemperatureC
	updated.ThermalExpansionPerC = request.ThermalExpansionPerC
	updated.CapacityCurveJSON = datatypes.JSON(curveJSON)
	updated.CoefficientVersion = strings.TrimSpace(request.CoefficientVersion)
	updated.TankStatus = request.TankStatus
	return s.repo.Update(ctx, updated, before, request.Version, actor)
}

func (s *TankService) MeasurementQuality(ctx context.Context, id uint) (dto.TankMeasurementQuality, error) {
	tank, err := s.repo.Get(ctx, id)
	if err != nil {
		return dto.TankMeasurementQuality{}, err
	}
	total, good, suspect, invalid, latest, err := s.repo.MeasurementQuality(ctx, id)
	if err != nil {
		return dto.TankMeasurementQuality{}, err
	}
	result := dto.TankMeasurementQuality{
		TankID: id, Total: total, Good: good, Suspect: suspect, Invalid: invalid,
		CalculationRead: tank.TankStatus == "active" && good+suspect >= 2,
	}
	if latest != nil {
		result.LatestSnapshot = latest
	}
	return result, nil
}

func validateAndMarshalTank(minimum, maximum, nominal float64, coefficients []float64) ([]byte, error) {
	if maximum <= minimum {
		return nil, api.NewError(422, "INVALID_LEVEL_BOUNDARY", "最高液位必须大于最低液位")
	}
	curve, err := balance.NewCapacityCurve(coefficients)
	if err != nil {
		return nil, api.WithDetails(api.NewError(422, "INVALID_CAPACITY_CURVE", "罐容曲线系数无效"), map[string]any{"reason": err.Error()})
	}
	previous := -1.0
	for index := 0; index <= 20; index++ {
		level := minimum + (maximum-minimum)*float64(index)/20
		volume, volumeErr := curve.VolumeAt(level, minimum, maximum, nominal)
		if volumeErr != nil {
			return nil, api.WithDetails(api.NewError(422, "INVALID_CAPACITY_CURVE", "罐容曲线超出物理边界"), map[string]any{"level_m": level, "reason": volumeErr.Error()})
		}
		if volume+1e-6 < previous {
			return nil, api.NewError(422, "NON_MONOTONIC_CAPACITY_CURVE", "罐容曲线必须随液位单调不减")
		}
		previous = volume
	}
	raw, err := json.Marshal(curve)
	if err != nil {
		return nil, fmt.Errorf("marshal validated capacity curve: %w", err)
	}
	return raw, nil
}
