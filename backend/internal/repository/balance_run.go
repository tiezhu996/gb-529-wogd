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

type BalanceFilter struct {
	TankID   uint
	Status   string
	Page     int
	PageSize int
}

type BalanceRepository struct {
	db *gorm.DB
}

func NewBalanceRepository(db *gorm.DB) *BalanceRepository {
	return &BalanceRepository{db: db}
}

func (r *BalanceRepository) List(ctx context.Context, filter BalanceFilter) ([]model.BalanceRun, int64, error) {
	filter.Page, filter.PageSize = normalizePage(filter.Page, filter.PageSize)
	query := r.db.WithContext(ctx).Model(&model.BalanceRun{})
	if filter.TankID > 0 {
		query = query.Where("tank_id = ?", filter.TankID)
	}
	if filter.Status != "" {
		query = query.Where("balance_status = ?", filter.Status)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count balance runs: %w", err)
	}
	var runs []model.BalanceRun
	if err := query.Preload("Tank").Order("created_at DESC, id DESC").
		Offset((filter.Page - 1) * filter.PageSize).Limit(filter.PageSize).Find(&runs).Error; err != nil {
		return nil, 0, fmt.Errorf("list balance runs: %w", err)
	}
	return runs, total, nil
}

func (r *BalanceRepository) Get(ctx context.Context, id uint) (model.BalanceRun, error) {
	var run model.BalanceRun
	if err := r.db.WithContext(ctx).Preload("Tank").First(&run, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return model.BalanceRun{}, api.NewError(404, "BALANCE_NOT_FOUND", "质量平衡运行不存在")
		}
		return model.BalanceRun{}, fmt.Errorf("get balance run: %w", err)
	}
	return run, nil
}

func (r *BalanceRepository) CreateCalculated(ctx context.Context, run *model.BalanceRun, actor Actor) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(run).Error; err != nil {
			return fmt.Errorf("create calculated balance run: %w", err)
		}
		queuedAudit := NewAudit(actor, "balance_run.queued", "balance_run", run.ID, nil, map[string]any{
			"tank_id": run.TankID, "period_start": run.PeriodStart, "period_end": run.PeriodEnd,
		})
		if err := tx.Create(&queuedAudit).Error; err != nil {
			return fmt.Errorf("audit queued balance run: %w", err)
		}
		calculatedAudit := NewAudit(actor, "balance_run.calculated", "balance_run", run.ID, map[string]any{"status": constants.BalanceQueued}, map[string]any{
			"status": run.BalanceStatus, "estimated_bog_kg": run.EstimatedBOGKG,
			"uncertainty_kg": run.UncertaintyKG, "deviation_level": run.DeviationLevel,
			"coefficient_version": run.CoefficientVersion,
		})
		if err := tx.Create(&calculatedAudit).Error; err != nil {
			return fmt.Errorf("audit balance calculation: %w", err)
		}
		return nil
	})
}

func (r *BalanceRepository) Transition(ctx context.Context, id, version uint, target constants.BalanceStatus, note string, reviewerID *uint, actor Actor) (model.BalanceRun, error) {
	var updated model.BalanceRun
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var before model.BalanceRun
		if err := tx.First(&before, id).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return api.NewError(404, "BALANCE_NOT_FOUND", "质量平衡运行不存在")
			}
			return fmt.Errorf("load balance run for transition: %w", err)
		}
		if before.Version != version {
			return api.NewError(409, "BALANCE_VERSION_CONFLICT", "平衡运行版本已变化，请刷新后重试")
		}
		if !constants.CanTransitionBalance(before.BalanceStatus, target) {
			return api.WithDetails(api.NewError(409, "INVALID_BALANCE_TRANSITION", "当前平衡状态不允许目标迁移"), map[string]any{
				"current": before.BalanceStatus, "target": target,
			})
		}
		updates := map[string]any{
			"balance_status": target,
			"version":        gorm.Expr("version + 1"),
			"review_note":    note,
		}
		if reviewerID != nil {
			now := time.Now().UTC()
			updates["reviewed_by"] = *reviewerID
			updates["reviewed_at"] = now
		}
		result := tx.Model(&model.BalanceRun{}).
			Where("id = ? AND version = ? AND balance_status = ?", id, version, before.BalanceStatus).
			Updates(updates)
		if result.Error != nil {
			return fmt.Errorf("transition balance run: %w", result.Error)
		}
		if result.RowsAffected != 1 {
			return api.NewError(409, "BALANCE_VERSION_CONFLICT", "平衡运行被其他请求更新")
		}
		if err := tx.First(&updated, id).Error; err != nil {
			return fmt.Errorf("reload balance run: %w", err)
		}
		audit := NewAudit(actor, "balance_run."+string(target), "balance_run", id, before, updated)
		if err := tx.Create(&audit).Error; err != nil {
			return fmt.Errorf("audit balance transition: %w", err)
		}
		return nil
	})
	return updated, err
}
