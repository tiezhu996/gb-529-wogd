package service

import (
	"context"
	"strings"
	"time"

	"lng-boiloff-gas-balance/backend/internal/balance"
	"lng-boiloff-gas-balance/backend/internal/constants"
	"lng-boiloff-gas-balance/backend/internal/dto"
	"lng-boiloff-gas-balance/backend/internal/model"
	"lng-boiloff-gas-balance/backend/internal/repository"
	"lng-boiloff-gas-balance/backend/pkg/api"
)

type TransferService struct {
	repo     *repository.TransferRepository
	tankRepo *repository.TankRepository
}

func NewTransferService(repo *repository.TransferRepository, tankRepo *repository.TankRepository) *TransferService {
	return &TransferService{repo: repo, tankRepo: tankRepo}
}

func (s *TransferService) List(ctx context.Context, filter repository.TransferFilter) ([]model.TransferOperation, int64, error) {
	if filter.OperationType != "" && filter.OperationType != "inflow" && filter.OperationType != "outflow" {
		return nil, 0, api.NewError(400, "INVALID_OPERATION_TYPE", "物理转移类型筛选值无效")
	}
	if filter.Status != "" && filter.Status != "draft" && filter.Status != "confirmed" && filter.Status != "cancelled" {
		return nil, 0, api.NewError(400, "INVALID_TRANSFER_STATUS", "物理转移状态筛选值无效")
	}
	return s.repo.List(ctx, filter)
}

func (s *TransferService) Get(ctx context.Context, id uint) (model.TransferOperation, error) {
	return s.repo.Get(ctx, id)
}

func (s *TransferService) Create(ctx context.Context, request dto.CreateTransferRequest, actor repository.Actor) (model.TransferOperation, error) {
	if !constants.CanAnalyze(actor.Role) {
		return model.TransferOperation{}, api.ErrForbidden
	}
	tank, err := s.tankRepo.Get(ctx, request.TankID)
	if err != nil {
		return model.TransferOperation{}, err
	}
	if tank.TankStatus != "active" {
		return model.TransferOperation{}, api.NewError(409, "TANK_NOT_ACTIVE", "只有启用储罐可以新增物理转移")
	}
	if request.StartAt == nil || request.EndAt == nil {
		return model.TransferOperation{}, api.NewError(400, "TRANSFER_TIME_REQUIRED", "必须提供转移开始和结束时间")
	}
	start := request.StartAt.UTC()
	end := request.EndAt.UTC()
	if !end.After(start) {
		return model.TransferOperation{}, api.NewError(422, "INVALID_TRANSFER_PERIOD", "转移结束时间必须晚于开始时间")
	}
	if end.Sub(start) > 30*24*time.Hour {
		return model.TransferOperation{}, api.NewError(422, "TRANSFER_PERIOD_TOO_LONG", "单条物理转移持续时间不能超过 30 天")
	}
	if err := balance.ValidateUncertainty(request.MeasurementUncertaintyPct); err != nil {
		return model.TransferOperation{}, api.WithDetails(api.NewError(422, "INVALID_UNCERTAINTY", "转移计量不确定度无效"), map[string]any{"reason": err.Error()})
	}
	item := model.TransferOperation{
		TankID:                    request.TankID,
		OperationType:             request.OperationType,
		StartAt:                   start,
		EndAt:                     end,
		MeasuredMassKG:            request.MeasuredMassKG,
		MeasurementUncertaintyPct: request.MeasurementUncertaintyPct,
		CounterpartyRef:           strings.TrimSpace(request.CounterpartyRef),
		OperationStatus:           request.OperationStatus,
		Version:                   1,
		CreatedBy:                 actor.UserID,
	}
	if err := s.repo.Create(ctx, &item, actor); err != nil {
		return model.TransferOperation{}, err
	}
	item.Tank = &tank
	return item, nil
}

func (s *TransferService) Transition(ctx context.Context, id uint, request dto.TransitionTransferRequest, actor repository.Actor) (model.TransferOperation, error) {
	if !constants.CanAnalyze(actor.Role) {
		return model.TransferOperation{}, api.ErrForbidden
	}
	reason := strings.TrimSpace(request.Reason)
	if request.TargetStatus == "cancelled" && len(reason) < 6 {
		return model.TransferOperation{}, api.NewError(422, "CANCELLATION_REASON_REQUIRED", "取消物理转移时必须填写不少于 6 个字符的原因")
	}
	return s.repo.Transition(ctx, id, request.Version, request.TargetStatus, reason, actor)
}
