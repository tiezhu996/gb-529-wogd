package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"

	"lng-boiloff-gas-balance/backend/internal/model"
	"lng-boiloff-gas-balance/backend/pkg/api"
)

type Actor struct {
	UserID    uint
	Email     string
	Role      string
	RequestID string
}

type AuditFilter struct {
	ActorEmail string
	Action     string
	EntityType string
	From       *time.Time
	To         *time.Time
	Page       int
	PageSize   int
}

type SupportRepository struct {
	db *gorm.DB
}

func NewSupportRepository(db *gorm.DB) *SupportRepository {
	return &SupportRepository{db: db}
}

func (r *SupportRepository) Ping(ctx context.Context) error {
	sqlDB, err := r.db.DB()
	if err != nil {
		return fmt.Errorf("obtain database connection: %w", err)
	}
	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}
	return nil
}

func (r *SupportRepository) FindUserByEmail(ctx context.Context, email string) (model.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return model.User{}, api.NewError(401, "INVALID_CREDENTIALS", "邮箱或密码不正确")
		}
		return model.User{}, fmt.Errorf("find user by email: %w", err)
	}
	return user, nil
}

func (r *SupportRepository) FindUserByID(ctx context.Context, id uint) (model.User, error) {
	var user model.User
	if err := r.db.WithContext(ctx).First(&user, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return model.User{}, api.NewError(401, "USER_NOT_FOUND", "登录用户不存在或已停用")
		}
		return model.User{}, fmt.Errorf("find user by id: %w", err)
	}
	return user, nil
}

func (r *SupportRepository) ListAudits(ctx context.Context, filter AuditFilter) ([]model.AuditEvent, int64, error) {
	query := r.db.WithContext(ctx).Model(&model.AuditEvent{})
	if filter.ActorEmail != "" {
		query = query.Where("actor_email = ?", filter.ActorEmail)
	}
	if filter.Action != "" {
		query = query.Where("action = ?", filter.Action)
	}
	if filter.EntityType != "" {
		query = query.Where("entity_type = ?", filter.EntityType)
	}
	if filter.From != nil {
		query = query.Where("created_at >= ?", filter.From.UTC())
	}
	if filter.To != nil {
		query = query.Where("created_at <= ?", filter.To.UTC())
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count audit events: %w", err)
	}
	var events []model.AuditEvent
	offset := (filter.Page - 1) * filter.PageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(filter.PageSize).Find(&events).Error; err != nil {
		return nil, 0, fmt.Errorf("list audit events: %w", err)
	}
	return events, total, nil
}

func NewAudit(actor Actor, action, entityType string, entityID uint, before, after any) model.AuditEvent {
	return model.AuditEvent{
		RequestID:  actor.RequestID,
		UserID:     actor.UserID,
		ActorEmail: actor.Email,
		Action:     action,
		EntityType: entityType,
		EntityID:   entityID,
		BeforeJSON: marshalSummary(before),
		AfterJSON:  marshalSummary(after),
		CreatedAt:  time.Now().UTC(),
	}
}

func marshalSummary(value any) string {
	if value == nil {
		return "{}"
	}
	data, err := json.Marshal(value)
	if err != nil {
		return `{"summary":"unavailable"}`
	}
	return string(data)
}

func normalizePage(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}
