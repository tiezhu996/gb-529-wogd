package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"lng-boiloff-gas-balance/backend/internal/balance"
	"lng-boiloff-gas-balance/backend/internal/constants"
	"lng-boiloff-gas-balance/backend/internal/dto"
	"lng-boiloff-gas-balance/backend/internal/model"
	"lng-boiloff-gas-balance/backend/internal/repository"
	"lng-boiloff-gas-balance/backend/pkg/api"
)

type MeasurementService struct {
	repo     *repository.MeasurementRepository
	tankRepo *repository.TankRepository
}

func NewMeasurementService(repo *repository.MeasurementRepository, tankRepo *repository.TankRepository) *MeasurementService {
	return &MeasurementService{repo: repo, tankRepo: tankRepo}
}

func (s *MeasurementService) List(ctx context.Context, filter repository.MeasurementFilter) ([]model.MeasurementSnapshot, int64, error) {
	if filter.Quality != "" && !constants.ValidQualityFlag(constants.QualityFlag(filter.Quality)) {
		return nil, 0, api.NewError(400, "INVALID_QUALITY_FLAG", "计量质量筛选值无效")
	}
	if filter.From != nil && filter.To != nil && !filter.To.After(*filter.From) {
		return nil, 0, api.NewError(400, "INVALID_TIME_RANGE", "结束时间必须晚于开始时间")
	}
	return s.repo.List(ctx, filter)
}

func (s *MeasurementService) Get(ctx context.Context, id uint) (model.MeasurementSnapshot, error) {
	return s.repo.Get(ctx, id)
}

func (s *MeasurementService) Create(ctx context.Context, request dto.CreateMeasurementRequest, actor repository.Actor) (model.MeasurementSnapshot, error) {
	if !constants.CanAnalyze(actor.Role) {
		return model.MeasurementSnapshot{}, api.ErrForbidden
	}
	tank, err := s.tankRepo.Get(ctx, request.TankID)
	if err != nil {
		return model.MeasurementSnapshot{}, err
	}
	if tank.TankStatus == "inactive" {
		return model.MeasurementSnapshot{}, api.NewError(409, "TANK_INACTIVE", "停用储罐不能新增计量快照")
	}
	if request.MeasuredAt == nil {
		return model.MeasurementSnapshot{}, api.NewError(400, "MEASURED_AT_REQUIRED", "必须提供计量时间")
	}
	measuredAt := request.MeasuredAt.UTC()
	if measuredAt.After(time.Now().UTC().Add(5 * time.Minute)) {
		return model.MeasurementSnapshot{}, api.NewError(422, "MEASUREMENT_TIME_IN_FUTURE", "计量时间不能晚于当前时间")
	}
	calculated, err := calculateMeasurement(tank, request)
	if err != nil {
		return model.MeasurementSnapshot{}, err
	}
	snapshot := model.MeasurementSnapshot{
		TankID:                    request.TankID,
		MeasuredAt:                measuredAt,
		LiquidLevelM:              request.LiquidLevelM,
		LiquidTempC:               request.LiquidTempC,
		VaporPressureKPA:          request.VaporPressureKPA,
		DensityKGM3:               request.DensityKGM3,
		CalculatedVolumeM3:        calculated.VolumeM3,
		TemperatureDensityKGM3:    calculated.CorrectedDensityKGM3,
		CalculatedLiquidMassKG:    calculated.LiquidMassKG,
		MeasurementUncertaintyPct: request.MeasurementUncertaintyPct,
		QualityFlag:               constants.QualityFlag(request.QualityFlag),
		SourceNote:                strings.TrimSpace(request.SourceNote),
		CreatedBy:                 actor.UserID,
	}
	if err := s.repo.Create(ctx, &snapshot, actor); err != nil {
		return model.MeasurementSnapshot{}, err
	}
	snapshot.Tank = &tank
	return snapshot, nil
}

func calculateMeasurement(tank model.StorageTank, request dto.CreateMeasurementRequest) (balance.SnapshotMassResult, error) {
	if err := balance.ValidateLevel(request.LiquidLevelM, tank.MinLevelM, tank.MaxLevelM); err != nil {
		return balance.SnapshotMassResult{}, api.WithDetails(api.NewError(422, "LEVEL_OUT_OF_RANGE", "液位超出储罐计算边界"), map[string]any{"reason": err.Error()})
	}
	if err := balance.ValidateUncertainty(request.MeasurementUncertaintyPct); err != nil {
		return balance.SnapshotMassResult{}, api.WithDetails(api.NewError(422, "INVALID_UNCERTAINTY", "计量不确定度无效"), map[string]any{"reason": err.Error()})
	}
	if !constants.ValidQualityFlag(constants.QualityFlag(request.QualityFlag)) {
		return balance.SnapshotMassResult{}, api.NewError(422, "INVALID_QUALITY_FLAG", "计量质量标记无效")
	}
	curve, err := balance.ParseCapacityCurve(tank.CapacityCurveJSON)
	if err != nil {
		return balance.SnapshotMassResult{}, fmt.Errorf("load tank capacity coefficients: %w", err)
	}
	result, err := balance.CalculateSnapshotMass(balance.SnapshotMassInput{
		LevelM:                request.LiquidLevelM,
		MinimumLevelM:         tank.MinLevelM,
		MaximumLevelM:         tank.MaxLevelM,
		NominalCapacityM3:     tank.NominalCapacityM3,
		DensityKGM3:           request.DensityKGM3,
		TemperatureC:          request.LiquidTempC,
		ReferenceTemperatureC: tank.ReferenceTemperatureC,
		ThermalExpansionPerC:  tank.ThermalExpansionPerC,
		Curve:                 curve,
	})
	if err != nil {
		return balance.SnapshotMassResult{}, api.WithDetails(api.NewError(422, "MEASUREMENT_CALCULATION_FAILED", "计量快照无法形成有效液相质量"), map[string]any{"reason": err.Error()})
	}
	return result, nil
}
