package repository

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"lng-boiloff-gas-balance/backend/internal/model"
	"lng-boiloff-gas-balance/backend/pkg/api"
)

type TransferFilter struct {
	TankID        uint
	OperationType string
	Status        string
	From          *time.Time
	To            *time.Time
	Page          int
	PageSize      int
}

type TransferRepository struct {
	db *gorm.DB
}

func NewTransferRepository(db *gorm.DB) *TransferRepository { return &TransferRepository{db: db} }

func (r *TransferRepository) List(ctx context.Context, filter TransferFilter) ([]model.TransferOperation, int64, error) {
	filter.Page, filter.PageSize = normalizePage(filter.Page, filter.PageSize)
	query := r.db.WithContext(ctx).Model(&model.TransferOperation{})
	if filter.TankID > 0 {
		query = query.Where("tank_id = ?", filter.TankID)
	}
	if filter.OperationType != "" {
		query = query.Where("operation_type = ?", filter.OperationType)
	}
	if filter.Status != "" {
		query = query.Where("operation_status = ?", filter.Status)
	}
	if filter.From != nil {
		query = query.Where("end_at >= ?", filter.From.UTC())
	}
	if filter.To != nil {
		query = query.Where("start_at <= ?", filter.To.UTC())
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count transfer operations: %w", err)
	}
	var items []model.TransferOperation
	if err := query.Preload("Tank").Order("start_at DESC, id DESC").
		Offset((filter.Page - 1) * filter.PageSize).Limit(filter.PageSize).Find(&items).Error; err != nil {
		return nil, 0, fmt.Errorf("list transfer operations: %w", err)
	}
	return items, total, nil
}

func (r *TransferRepository) Get(ctx context.Context, id uint) (model.TransferOperation, error) {
	var item model.TransferOperation
	if err := r.db.WithContext(ctx).Preload("Tank").First(&item, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return model.TransferOperation{}, api.NewError(404, "TRANSFER_NOT_FOUND", "物理转移记录不存在")
		}
		return model.TransferOperation{}, fmt.Errorf("get transfer operation: %w", err)
	}
	return item, nil
}

func (r *TransferRepository) Create(ctx context.Context, item *model.TransferOperation, actor Actor) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var overlaps int64
		if err := tx.Model(&model.TransferOperation{}).
			Where("tank_id = ? AND operation_status <> ? AND start_at < ? AND end_at > ?", item.TankID, "cancelled", item.EndAt, item.StartAt).
			Count(&overlaps).Error; err != nil {
			return fmt.Errorf("check transfer time overlap: %w", err)
		}
		if overlaps > 0 {
			return api.NewError(409, "TRANSFER_TIME_OVERLAP", "该储罐已有时间重叠的未取消物理转移")
		}
		if err := tx.Create(item).Error; err != nil {
			return fmt.Errorf("create transfer operation: %w", err)
		}
		audit := NewAudit(actor, "transfer_operation.created", "transfer_operation", item.ID, nil, item)
		if err := tx.Create(&audit).Error; err != nil {
			return fmt.Errorf("audit transfer operation: %w", err)
		}
		return nil
	})
}

func (r *TransferRepository) Transition(ctx context.Context, id, version uint, target, reason string, actor Actor) (model.TransferOperation, error) {
	var updated model.TransferOperation
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var before model.TransferOperation
		if err := tx.First(&before, id).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return api.NewError(404, "TRANSFER_NOT_FOUND", "物理转移记录不存在")
			}
			return fmt.Errorf("load transfer operation: %w", err)
		}
		if before.Version != version {
			return api.NewError(409, "TRANSFER_VERSION_CONFLICT", "物理转移记录版本已变化，请刷新后重试")
		}
		allowed := before.OperationStatus == "draft" && (target == "confirmed" || target == "cancelled")
		allowed = allowed || (before.OperationStatus == "confirmed" && target == "cancelled")
		if !allowed {
			return api.WithDetails(api.NewError(409, "INVALID_TRANSFER_TRANSITION", "当前转移状态不允许目标迁移"), map[string]any{
				"current": before.OperationStatus, "target": target,
			})
		}
		result := tx.Model(&model.TransferOperation{}).
			Where("id = ? AND version = ? AND operation_status = ?", id, version, before.OperationStatus).
			Updates(map[string]any{"operation_status": target, "version": gorm.Expr("version + 1")})
		if result.Error != nil {
			return fmt.Errorf("transition transfer operation: %w", result.Error)
		}
		if result.RowsAffected != 1 {
			return api.NewError(409, "TRANSFER_VERSION_CONFLICT", "物理转移记录被其他请求更新")
		}
		if err := tx.First(&updated, id).Error; err != nil {
			return fmt.Errorf("reload transfer operation: %w", err)
		}
		audit := NewAudit(actor, "transfer_operation."+target, "transfer_operation", id, before, map[string]any{"record": updated, "reason": reason})
		if err := tx.Create(&audit).Error; err != nil {
			return fmt.Errorf("audit transfer transition: %w", err)
		}
		return nil
	})
	return updated, err
}

func (r *TransferRepository) ConfirmedForPeriod(ctx context.Context, tankID uint, start, end time.Time) ([]model.TransferOperation, error) {
	var items []model.TransferOperation
	if err := r.db.WithContext(ctx).
		Where("tank_id = ? AND operation_status = ? AND start_at >= ? AND end_at <= ?", tankID, "confirmed", start.UTC(), end.UTC()).
		Order("start_at ASC, id ASC").Find(&items).Error; err != nil {
		return nil, fmt.Errorf("list confirmed period transfers: %w", err)
	}
	return items, nil
}
