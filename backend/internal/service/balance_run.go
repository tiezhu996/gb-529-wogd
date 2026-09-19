package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"gorm.io/datatypes"

	"lng-boiloff-gas-balance/backend/internal/balance"
	"lng-boiloff-gas-balance/backend/internal/constants"
	"lng-boiloff-gas-balance/backend/internal/dto"
	"lng-boiloff-gas-balance/backend/internal/model"
	"lng-boiloff-gas-balance/backend/internal/repository"
	"lng-boiloff-gas-balance/backend/pkg/api"
)

const balanceAlgorithmVersion = "mass-balance-v1.0"

type BalanceService struct {
	repo            *repository.BalanceRepository
	tankRepo        *repository.TankRepository
	measurementRepo *repository.MeasurementRepository
	transferRepo    *repository.TransferRepository
}

func NewBalanceService(repo *repository.BalanceRepository, tankRepo *repository.TankRepository, measurementRepo *repository.MeasurementRepository, transferRepo *repository.TransferRepository) *BalanceService {
	return &BalanceService{repo: repo, tankRepo: tankRepo, measurementRepo: measurementRepo, transferRepo: transferRepo}
}

func (s *BalanceService) List(ctx context.Context, filter repository.BalanceFilter) ([]model.BalanceRun, int64, error) {
	if filter.Status != "" && !constants.ValidBalanceStatus(constants.BalanceStatus(filter.Status)) {
		return nil, 0, api.NewError(400, "INVALID_BALANCE_STATUS", "平衡状态筛选值无效")
	}
	return s.repo.List(ctx, filter)
}

func (s *BalanceService) Get(ctx context.Context, id uint) (model.BalanceRun, error) {
	return s.repo.Get(ctx, id)
}

func (s *BalanceService) Run(ctx context.Context, request dto.RunBalanceRequest, actor repository.Actor) (model.BalanceRun, error) {
	if !constants.CanAnalyze(actor.Role) {
		return model.BalanceRun{}, api.ErrForbidden
	}
	if request.PeriodStart == nil || request.PeriodEnd == nil {
		return model.BalanceRun{}, api.NewError(400, "BALANCE_PERIOD_REQUIRED", "必须提供平衡期间起止时间")
	}
	start, end := request.PeriodStart.UTC(), request.PeriodEnd.UTC()
	if !end.After(start) {
		return model.BalanceRun{}, api.NewError(422, "INVALID_BALANCE_PERIOD", "平衡期间结束时间必须晚于开始时间")
	}
	if end.Sub(start) > 90*24*time.Hour {
		return model.BalanceRun{}, api.NewError(422, "BALANCE_PERIOD_TOO_LONG", "单次质量平衡期间不能超过 90 天")
	}
	// Existence check before the transaction; status and period inputs are
	// re-read and locked inside CalculateAndStore.
	if _, err := s.tankRepo.Get(ctx, request.TankID); err != nil {
		return model.BalanceRun{}, err
	}
	run, err := s.repo.CalculateAndStore(ctx, request.TankID, start, end, actor, func(inputs repository.PeriodInputs) (*model.BalanceRun, error) {
		contained, violations := classifyPeriodTransfers(inputs.Transfer, start, end)
		if len(violations) > 0 {
			return nil, periodBoundaryError(violations)
		}
		calculation, snapshotJSON, evidenceJSON, err := calculateBalanceRun(inputs.Tank, inputs.Opening, inputs.Closing, contained, start, end)
		if err != nil {
			return nil, err
		}
		return &model.BalanceRun{
			TankID:             inputs.Tank.ID,
			PeriodStart:        start,
			PeriodEnd:          end,
			BalanceStatus:      constants.BalanceCalculating,
			InputSnapshotJSON:  datatypes.JSON(snapshotJSON),
			EvidenceJSON:       datatypes.JSON(evidenceJSON),
			OpeningMassKG:      calculation.OpeningMassKG,
			ClosingMassKG:      calculation.ClosingMassKG,
			NetTransferKG:      calculation.NetTransferKG,
			EstimatedBOGKG:     calculation.EstimatedBOGKG,
			UncertaintyKG:      calculation.UncertaintyKG,
			IntervalLowerKG:    calculation.IntervalLowerKG,
			IntervalUpperKG:    calculation.IntervalUpperKG,
			DeviationPct:       calculation.DeviationPct,
			DeviationLevel:     calculation.DeviationLevel,
			CoefficientVersion: inputs.Tank.CoefficientVersion,
			Version:            2,
			CreatedBy:          actor.UserID,
		}, nil
	})
	if err != nil {
		return model.BalanceRun{}, err
	}
	tank, err := s.tankRepo.Get(ctx, request.TankID)
	if err != nil {
		return model.BalanceRun{}, err
	}
	run.Tank = &tank
	return run, nil
}

// PeriodBoundaryViolation identifies a confirmed transfer that crosses a
// period boundary instead of lying wholly inside the period.
type PeriodBoundaryViolation struct {
	TransferID      uint      `json:"transfer_id"`
	OperationType   string    `json:"operation_type"`
	CrossedBoundary string    `json:"crossed_boundary"`
	BoundaryAt      time.Time `json:"boundary_at"`
	OutOfBoundaryAt time.Time `json:"out_of_boundary_at"`
}

// classifyPeriodTransfers partitions the confirmed transfers that intersect the
// closed [start, end] period into fully contained transfers and crossings.
// Transfers merely touching a boundary (end == start or start == end) are not
// inside the period and are silently excluded from the balance without being a
// crossing; a crossing is strictly spanning across a boundary. Draft and
// cancelled transfers are already excluded by the repository query.
func classifyPeriodTransfers(transfers []model.TransferOperation, start, end time.Time) ([]model.TransferOperation, []PeriodBoundaryViolation) {
	contained := make([]model.TransferOperation, 0, len(transfers))
	violations := make([]PeriodBoundaryViolation, 0)
	for _, transfer := range transfers {
		// Only transfers intersecting the open interval (start, end) are in
		// scope; a transfer ending exactly at the start or beginning exactly at
		// the end carries no mass across the period and is neither counted nor a
		// crossing.
		if !transfer.StartAt.Before(end) || !transfer.EndAt.After(start) {
			continue
		}
		crossesStart := transfer.StartAt.Before(start) && transfer.EndAt.After(start)
		crossesEnd := transfer.EndAt.After(end) && transfer.StartAt.Before(end)
		if !crossesStart && !crossesEnd {
			contained = append(contained, transfer)
			continue
		}
		if crossesStart {
			violations = append(violations, PeriodBoundaryViolation{
				TransferID: transfer.ID, OperationType: transfer.OperationType,
				CrossedBoundary: "period_start", BoundaryAt: start, OutOfBoundaryAt: transfer.StartAt,
			})
		}
		if crossesEnd {
			violations = append(violations, PeriodBoundaryViolation{
				TransferID: transfer.ID, OperationType: transfer.OperationType,
				CrossedBoundary: "period_end", BoundaryAt: end, OutOfBoundaryAt: transfer.EndAt,
			})
		}
	}
	return contained, violations
}

func periodBoundaryError(violations []PeriodBoundaryViolation) error {
	first := violations[0]
	return api.WithDetails(api.NewError(422, "TRANSFER_CROSSES_PERIOD_BOUNDARY",
		fmt.Sprintf("已确认转移 #%d 跨越期间%s，平衡已整体拒绝；请调整期间或先处理该转移。",
			first.TransferID, boundaryName(first.CrossedBoundary))), map[string]any{
		"violations": violations,
	})
}

func boundaryName(boundary string) string {
	if boundary == "period_start" {
		return "起点"
	}
	return "终点"
}

type calculatedBalance struct {
	OpeningMassKG   float64
	ClosingMassKG   float64
	NetTransferKG   float64
	EstimatedBOGKG  float64
	UncertaintyKG   float64
	IntervalLowerKG float64
	IntervalUpperKG float64
	DeviationPct    float64
	DeviationLevel  constants.DeviationLevel
}

type balanceEvidence struct {
	AlgorithmVersion string                   `json:"algorithm_version"`
	Equation         map[string]float64       `json:"equation"`
	Uncertainty      dto.UncertaintyBreakdown `json:"uncertainty"`
	SafetyBoundary   string                   `json:"safety_boundary"`
}

func calculateBalanceRun(tank model.StorageTank, opening, closing model.MeasurementSnapshot, transfers []model.TransferOperation, start, end time.Time) (calculatedBalance, []byte, []byte, error) {
	inflows, outflows := make([]float64, 0), make([]float64, 0)
	uncertaintyInputs := []balance.UncertaintyInput{
		{Source: "opening_snapshot", EntityID: opening.ID, MassKG: opening.CalculatedLiquidMassKG, UncertaintyPct: opening.MeasurementUncertaintyPct},
		{Source: "closing_snapshot", EntityID: closing.ID, MassKG: closing.CalculatedLiquidMassKG, UncertaintyPct: closing.MeasurementUncertaintyPct},
	}
	components := []dto.UncertaintyComponent{
		{Source: "opening_snapshot", EntityID: opening.ID, MassKG: opening.CalculatedLiquidMassKG, UncertaintyPct: opening.MeasurementUncertaintyPct},
		{Source: "closing_snapshot", EntityID: closing.ID, MassKG: closing.CalculatedLiquidMassKG, UncertaintyPct: closing.MeasurementUncertaintyPct},
	}
	for _, transfer := range transfers {
		if transfer.OperationType == "inflow" {
			inflows = append(inflows, transfer.MeasuredMassKG)
		} else {
			outflows = append(outflows, transfer.MeasuredMassKG)
		}
		source := "transfer_" + transfer.OperationType
		uncertaintyInputs = append(uncertaintyInputs, balance.UncertaintyInput{
			Source: source, EntityID: transfer.ID, MassKG: transfer.MeasuredMassKG, UncertaintyPct: transfer.MeasurementUncertaintyPct,
		})
		components = append(components, dto.UncertaintyComponent{
			Source: source, EntityID: transfer.ID, MassKG: transfer.MeasuredMassKG, UncertaintyPct: transfer.MeasurementUncertaintyPct,
		})
	}
	net, err := balance.NetTransfer(inflows, outflows)
	if err != nil {
		return calculatedBalance{}, nil, nil, fmt.Errorf("calculate net transfer: %w", err)
	}
	deviation, err := balance.PhysicalBalance(opening.CalculatedLiquidMassKG, net, closing.CalculatedLiquidMassKG)
	if err != nil {
		return calculatedBalance{}, nil, nil, fmt.Errorf("calculate physical mass balance: %w", err)
	}
	propagated, err := balance.PropagateUncertainty(uncertaintyInputs)
	if err != nil {
		return calculatedBalance{}, nil, nil, fmt.Errorf("propagate measurement uncertainty: %w", err)
	}
	for index := range components {
		components[index].AbsoluteKG = propagated.Components[index]
	}
	valid := opening.QualityFlag != constants.QualityInvalid && closing.QualityFlag != constants.QualityInvalid
	level := balance.ClassifyDeviation(deviation, propagated.CombinedKG, valid)
	lower, upper := balance.ConfidenceInterval(deviation, propagated.CombinedKG)
	breakdown := dto.UncertaintyBreakdown{
		CombinedKG:   propagated.CombinedKG,
		LowerKG:      lower,
		UpperKG:      upper,
		Relationship: level,
		Components:   components,
	}
	evidence := balanceEvidence{
		AlgorithmVersion: balanceAlgorithmVersion,
		Equation: map[string]float64{
			"opening_mass_kg":                  opening.CalculatedLiquidMassKG,
			"net_transfer_kg":                  net,
			"closing_mass_kg":                  closing.CalculatedLiquidMassKG,
			"estimated_bog_and_unexplained_kg": deviation,
		},
		Uncertainty:    breakdown,
		SafetyBoundary: "未解释差异仅为工程分析结果，不直接认定为泄漏或安全事件。",
	}
	inputSnapshot := map[string]any{
		"algorithm_version":   balanceAlgorithmVersion,
		"coefficient_version": tank.CoefficientVersion,
		"period_start":        start,
		"period_end":          end,
		"tank":                tank,
		"opening_snapshot":    opening,
		"closing_snapshot":    closing,
		"confirmed_transfers": transfers,
	}
	snapshotJSON, err := json.Marshal(inputSnapshot)
	if err != nil {
		return calculatedBalance{}, nil, nil, fmt.Errorf("marshal immutable balance input snapshot: %w", err)
	}
	evidenceJSON, err := json.Marshal(evidence)
	if err != nil {
		return calculatedBalance{}, nil, nil, fmt.Errorf("marshal balance evidence: %w", err)
	}
	return calculatedBalance{
		OpeningMassKG:   opening.CalculatedLiquidMassKG,
		ClosingMassKG:   closing.CalculatedLiquidMassKG,
		NetTransferKG:   net,
		EstimatedBOGKG:  deviation,
		UncertaintyKG:   propagated.CombinedKG,
		IntervalLowerKG: lower,
		IntervalUpperKG: upper,
		DeviationPct:    balance.DeviationPercent(deviation, opening.CalculatedLiquidMassKG),
		DeviationLevel:  level,
	}, snapshotJSON, evidenceJSON, nil
}

func (s *BalanceService) Submit(ctx context.Context, id uint, request dto.SubmitBalanceRequest, actor repository.Actor) (model.BalanceRun, error) {
	if !constants.CanAnalyze(actor.Role) {
		return model.BalanceRun{}, api.ErrForbidden
	}
	return s.repo.Transition(ctx, id, request.Version, constants.BalancePendingReview, "提交独立复核", nil, actor)
}

func (s *BalanceService) Review(ctx context.Context, id uint, request dto.ReviewBalanceRequest, actor repository.Actor) (model.BalanceRun, error) {
	if !constants.CanReview(actor.Role) {
		return model.BalanceRun{}, api.ErrForbidden
	}
	if request.TargetStatus != constants.BalanceAccepted && request.TargetStatus != constants.BalanceRejected {
		return model.BalanceRun{}, api.NewError(422, "INVALID_REVIEW_DECISION", "复核目标状态只能是 accepted 或 rejected")
	}
	note := strings.TrimSpace(request.ReviewNote)
	return s.repo.Transition(ctx, id, request.Version, request.TargetStatus, note, &actor.UserID, actor)
}

func (s *BalanceService) Invalidate(ctx context.Context, id uint, request dto.InvalidateBalanceRequest, actor repository.Actor) (model.BalanceRun, error) {
	if !constants.CanAdmin(actor.Role) {
		return model.BalanceRun{}, api.ErrForbidden
	}
	note := strings.TrimSpace(request.Reason)
	return s.repo.Transition(ctx, id, request.Version, constants.BalanceInvalidated, note, &actor.UserID, actor)
}

func (s *BalanceService) Uncertainty(ctx context.Context, id uint) (dto.UncertaintyBreakdown, error) {
	run, err := s.repo.Get(ctx, id)
	if err != nil {
		return dto.UncertaintyBreakdown{}, err
	}
	var evidence balanceEvidence
	if err := json.Unmarshal(run.EvidenceJSON, &evidence); err != nil {
		return dto.UncertaintyBreakdown{}, fmt.Errorf("decode stored uncertainty evidence: %w", err)
	}
	if math.Abs(evidence.Uncertainty.CombinedKG-run.UncertaintyKG) > 0.01 {
		return dto.UncertaintyBreakdown{}, api.NewError(500, "EVIDENCE_INTEGRITY_ERROR", "存储的不确定度证据与运行结果不一致")
	}
	evidence.Uncertainty.BalanceRunID = run.ID
	return evidence.Uncertainty, nil
}
