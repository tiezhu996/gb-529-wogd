package repository

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"lng-boiloff-gas-balance/backend/internal/constants"
	"lng-boiloff-gas-balance/backend/internal/model"
	"lng-boiloff-gas-balance/backend/pkg/api"
)

type TankRepository struct {
	db *gorm.DB
}

func NewTankRepository(db *gorm.DB) *TankRepository { return &TankRepository{db: db} }

func (r *TankRepository) List(ctx context.Context, page, pageSize int, status string) ([]model.StorageTank, int64, error) {
	page, pageSize = normalizePage(page, pageSize)
	query := r.db.WithContext(ctx).Model(&model.StorageTank{})
	if status != "" {
		query = query.Where("tank_status = ?", status)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count storage tanks: %w", err)
	}
	var tanks []model.StorageTank
	if err := query.Order("tank_code ASC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&tanks).Error; err != nil {
		return nil, 0, fmt.Errorf("list storage tanks: %w", err)
	}
	return tanks, total, nil
}

func (r *TankRepository) Get(ctx context.Context, id uint) (model.StorageTank, error) {
	var tank model.StorageTank
	if err := r.db.WithContext(ctx).First(&tank, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return model.StorageTank{}, api.NewError(404, "TANK_NOT_FOUND", "储罐不存在")
		}
		return model.StorageTank{}, fmt.Errorf("get storage tank: %w", err)
	}
	return tank, nil
}

func (r *TankRepository) Create(ctx context.Context, tank *model.StorageTank, actor Actor) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(tank).Error; err != nil {
			if err == gorm.ErrDuplicatedKey {
				return api.NewError(409, "TANK_CODE_EXISTS", "储罐编号已存在")
			}
			return fmt.Errorf("create storage tank: %w", err)
		}
		audit := NewAudit(actor, "storage_tank.created", "storage_tank", tank.ID, nil, tank)
		if err := tx.Create(&audit).Error; err != nil {
			return fmt.Errorf("audit storage tank creation: %w", err)
		}
		return nil
	})
}

func (r *TankRepository) Update(ctx context.Context, updated, before model.StorageTank, expectedVersion uint, actor Actor) (model.StorageTank, error) {
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		updates := map[string]any{
			"name":                    updated.Name,
			"nominal_capacity_m3":     updated.NominalCapacityM3,
			"min_level_m":             updated.MinLevelM,
			"max_level_m":             updated.MaxLevelM,
			"reference_density_kgm3":  updated.ReferenceDensityKGM3,
			"reference_temperature_c": updated.ReferenceTemperatureC,
			"thermal_expansion_per_c": updated.ThermalExpansionPerC,
			"capacity_curve_json":     updated.CapacityCurveJSON,
			"coefficient_version":     updated.CoefficientVersion,
			"tank_status":             updated.TankStatus,
			"version":                 gorm.Expr("version + 1"),
		}
		result := tx.Model(&model.StorageTank{}).Where("id = ? AND version = ?", updated.ID, expectedVersion).Updates(updates)
		if result.Error != nil {
			return fmt.Errorf("update storage tank: %w", result.Error)
		}
		if result.RowsAffected != 1 {
			return api.NewError(409, "TANK_VERSION_CONFLICT", "储罐参数已被其他请求更新，请刷新后重试")
		}
		if err := tx.First(&updated, updated.ID).Error; err != nil {
			return fmt.Errorf("reload storage tank: %w", err)
		}
		audit := NewAudit(actor, "storage_tank.coefficients_updated", "storage_tank", updated.ID, before, updated)
		if err := tx.Create(&audit).Error; err != nil {
			return fmt.Errorf("audit storage tank update: %w", err)
		}
		return nil
	})
	return updated, err
}

func (r *TankRepository) MeasurementQuality(ctx context.Context, tankID uint) (total, good, suspect, invalid int64, latest *model.MeasurementSnapshot, err error) {
	query := r.db.WithContext(ctx).Model(&model.MeasurementSnapshot{}).Where("tank_id = ?", tankID)
	if err = query.Count(&total).Error; err != nil {
		return 0, 0, 0, 0, nil, fmt.Errorf("count tank snapshots: %w", err)
	}
	count := func(flag constants.QualityFlag, destination *int64) error {
		return query.Session(&gorm.Session{}).Where("quality_flag = ?", flag).Count(destination).Error
	}
	if err = count(constants.QualityGood, &good); err != nil {
		return 0, 0, 0, 0, nil, fmt.Errorf("count good snapshots: %w", err)
	}
	if err = count(constants.QualitySuspect, &suspect); err != nil {
		return 0, 0, 0, 0, nil, fmt.Errorf("count suspect snapshots: %w", err)
	}
	if err = count(constants.QualityInvalid, &invalid); err != nil {
		return 0, 0, 0, 0, nil, fmt.Errorf("count invalid snapshots: %w", err)
	}
	var item model.MeasurementSnapshot
	find := r.db.WithContext(ctx).Where("tank_id = ?", tankID).Order("measured_at DESC, id DESC").First(&item)
	if find.Error == nil {
		latest = &item
	} else if find.Error != gorm.ErrRecordNotFound {
		return 0, 0, 0, 0, nil, fmt.Errorf("load latest snapshot: %w", find.Error)
	}
	return total, good, suspect, invalid, latest, nil
}
