package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"lng-boiloff-gas-balance/backend/internal/constants"
	"lng-boiloff-gas-balance/backend/internal/model"
	"lng-boiloff-gas-balance/backend/pkg/api"
)

type MeasurementFilter struct {
	TankID   uint
	Quality  string
	From     *time.Time
	To       *time.Time
	Page     int
	PageSize int
}

type MeasurementRepository struct {
	db *gorm.DB
}

func NewMeasurementRepository(db *gorm.DB) *MeasurementRepository {
	return &MeasurementRepository{db: db}
}

func (r *MeasurementRepository) List(ctx context.Context, filter MeasurementFilter) ([]model.MeasurementSnapshot, int64, error) {
	filter.Page, filter.PageSize = normalizePage(filter.Page, filter.PageSize)
	query := r.db.WithContext(ctx).Model(&model.MeasurementSnapshot{})
	if filter.TankID > 0 {
		query = query.Where("tank_id = ?", filter.TankID)
	}
	if filter.Quality != "" {
		query = query.Where("quality_flag = ?", filter.Quality)
	}
	if filter.From != nil {
		query = query.Where("measured_at >= ?", filter.From.UTC())
	}
	if filter.To != nil {
		query = query.Where("measured_at <= ?", filter.To.UTC())
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count measurement snapshots: %w", err)
	}
	var snapshots []model.MeasurementSnapshot
	if err := query.Preload("Tank").Order("measured_at DESC, id DESC").
		Offset((filter.Page - 1) * filter.PageSize).Limit(filter.PageSize).Find(&snapshots).Error; err != nil {
		return nil, 0, fmt.Errorf("list measurement snapshots: %w", err)
	}
	return snapshots, total, nil
}

func (r *MeasurementRepository) Get(ctx context.Context, id uint) (model.MeasurementSnapshot, error) {
	var snapshot model.MeasurementSnapshot
	if err := r.db.WithContext(ctx).Preload("Tank").First(&snapshot, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return model.MeasurementSnapshot{}, api.NewError(404, "MEASUREMENT_NOT_FOUND", "计量快照不存在")
		}
		return model.MeasurementSnapshot{}, fmt.Errorf("get measurement snapshot: %w", err)
	}
	return snapshot, nil
}

func (r *MeasurementRepository) Create(ctx context.Context, snapshot *model.MeasurementSnapshot, actor Actor) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing int64
		if err := tx.Model(&model.MeasurementSnapshot{}).
			Where("tank_id = ? AND measured_at = ?", snapshot.TankID, snapshot.MeasuredAt).
			Count(&existing).Error; err != nil {
			return fmt.Errorf("check duplicate measurement timestamp: %w", err)
		}
		if existing > 0 {
			return api.NewError(409, "MEASUREMENT_DUPLICATE", "该储罐在相同时间已存在计量快照")
		}
		if err := tx.Create(snapshot).Error; err != nil {
			return fmt.Errorf("create immutable measurement snapshot: %w", err)
		}
		audit := NewAudit(actor, "measurement_snapshot.created", "measurement_snapshot", snapshot.ID, nil, snapshot)
		if err := tx.Create(&audit).Error; err != nil {
			return fmt.Errorf("audit measurement snapshot: %w", err)
		}
		return nil
	})
}

func (r *MeasurementRepository) BoundarySnapshots(ctx context.Context, tankID uint, periodStart, periodEnd time.Time) (model.MeasurementSnapshot, model.MeasurementSnapshot, error) {
	var opening model.MeasurementSnapshot
	openingQuery := r.db.WithContext(ctx).
		Where("tank_id = ? AND measured_at <= ? AND quality_flag <> ?", tankID, periodStart.UTC(), constants.QualityInvalid).
		Order("measured_at DESC, id DESC").First(&opening)
	if openingQuery.Error != nil {
		if openingQuery.Error == gorm.ErrRecordNotFound {
			return model.MeasurementSnapshot{}, model.MeasurementSnapshot{}, api.NewError(422, "OPENING_SNAPSHOT_MISSING", "期间开始前缺少有效期初计量快照")
		}
		return model.MeasurementSnapshot{}, model.MeasurementSnapshot{}, fmt.Errorf("load opening snapshot: %w", openingQuery.Error)
	}
	var closing model.MeasurementSnapshot
	closingQuery := r.db.WithContext(ctx).
		Where("tank_id = ? AND measured_at >= ? AND measured_at <= ? AND quality_flag <> ?", tankID, periodStart.UTC(), periodEnd.UTC(), constants.QualityInvalid).
		Order("measured_at DESC, id DESC").First(&closing)
	if closingQuery.Error != nil {
		if closingQuery.Error == gorm.ErrRecordNotFound {
			return model.MeasurementSnapshot{}, model.MeasurementSnapshot{}, api.NewError(422, "CLOSING_SNAPSHOT_MISSING", "期间结束前缺少有效期末计量快照")
		}
		return model.MeasurementSnapshot{}, model.MeasurementSnapshot{}, fmt.Errorf("load closing snapshot: %w", closingQuery.Error)
	}
	if closing.ID == opening.ID || !closing.MeasuredAt.After(opening.MeasuredAt) {
		return model.MeasurementSnapshot{}, model.MeasurementSnapshot{}, api.NewError(422, "BOUNDARY_SNAPSHOTS_INVALID", "期初和期末快照必须是时间递增的两条独立记录")
	}
	return opening, closing, nil
}

func (r *MeasurementRepository) ListForTank(ctx context.Context, tankID uint) ([]model.MeasurementSnapshot, error) {
	var snapshots []model.MeasurementSnapshot
	if err := r.db.WithContext(ctx).Where("tank_id = ?", tankID).Order("measured_at ASC").Find(&snapshots).Error; err != nil {
		return nil, fmt.Errorf("list tank measurement history: %w", err)
	}
	return snapshots, nil
}
