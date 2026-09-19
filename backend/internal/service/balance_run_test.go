package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"gorm.io/datatypes"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"lng-boiloff-gas-balance/backend/internal/balance"
	"lng-boiloff-gas-balance/backend/internal/constants"
	"lng-boiloff-gas-balance/backend/internal/dto"
	"lng-boiloff-gas-balance/backend/internal/model"
	"lng-boiloff-gas-balance/backend/internal/repository"
	"lng-boiloff-gas-balance/backend/pkg/api"
)

func boundaryTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:balance-boundary-529?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	models := []any{&model.User{}, &model.StorageTank{}, &model.MeasurementSnapshot{}, &model.TransferOperation{}, &model.BalanceRun{}, &model.AuditEvent{}}
	if err := db.AutoMigrate(models...); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func boundaryServices(db *gorm.DB) (*BalanceService, *repository.TransferRepository) {
	tankRepo := repository.NewTankRepository(db)
	measurementRepo := repository.NewMeasurementRepository(db)
	transferRepo := repository.NewTransferRepository(db)
	balanceRepo := repository.NewBalanceRepository(db)
	return NewBalanceService(balanceRepo, tankRepo, measurementRepo, transferRepo), transferRepo
}

func seedBoundaryTank(t *testing.T, db *gorm.DB, code string) (model.StorageTank, time.Time, time.Time) {
	t.Helper()
	curve, err := balance.NewCapacityCurve([]float64{0, 15000})
	if err != nil {
		t.Fatalf("curve: %v", err)
	}
	raw, err := curve.Marshal()
	if err != nil {
		t.Fatalf("marshal curve: %v", err)
	}
	start := time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)
	tank := model.StorageTank{
		TankCode: code, Name: "边界测试罐", NominalCapacityM3: 180000, MinLevelM: 0, MaxLevelM: 12,
		ReferenceDensityKGM3: 452, ReferenceTemperatureC: -160, ThermalExpansionPerC: 0.0035,
		CapacityCurveJSON: datatypes.JSON(raw), CoefficientVersion: "CV-T1", TankStatus: "active",
	}
	if err := db.Create(&tank).Error; err != nil {
		t.Fatalf("create tank: %v", err)
	}
	snapshot := func(at time.Time, mass float64, flag constants.QualityFlag) model.MeasurementSnapshot {
		return model.MeasurementSnapshot{
			TankID: tank.ID, MeasuredAt: at, LiquidLevelM: 10, LiquidTempC: -160, VaporPressureKPA: 110,
			DensityKGM3: 452, CalculatedVolumeM3: 150000, TemperatureDensityKGM3: 452,
			CalculatedLiquidMassKG: mass, MeasurementUncertaintyPct: 0.3, QualityFlag: flag,
			SourceNote: "boundary test", CreatedBy: 1,
		}
	}
	snapshots := []model.MeasurementSnapshot{
		snapshot(start.Add(-time.Hour), 55_000_000, constants.QualityGood),
		snapshot(end.Add(-time.Hour), 54_900_000, constants.QualityGood),
	}
	for index := range snapshots {
		if err := db.Create(&snapshots[index]).Error; err != nil {
			t.Fatalf("create snapshot: %v", err)
		}
	}
	return tank, start, end
}

func boundaryTransfer(tankID uint, operationType string, startAt, endAt time.Time, mass float64, status string) model.TransferOperation {
	return model.TransferOperation{
		TankID: tankID, OperationType: operationType, StartAt: startAt, EndAt: endAt, MeasuredMassKG: mass,
		MeasurementUncertaintyPct: 0.25, CounterpartyRef: "BOUNDARY-METER", OperationStatus: status,
		Version: 1, CreatedBy: 1,
	}
}

var boundaryActor = repository.Actor{UserID: 1, Email: "analyst@lng.local", Role: constants.RoleProcessAnalyst, RequestID: "boundary-test"}

func TestRunBalanceReadsEvidenceInsideTransactionAndRejectsCrossingTransfer(t *testing.T) {
	db := boundaryTestDB(t)
	svc, transferRepo := boundaryServices(db)

	cases := []struct {
		name        string
		kind        string
		startOffset time.Duration
		endOffset   time.Duration
		status      string
		wantCode    string
		wantRecord  bool
	}{
		{name: "fully contained confirmed inflow counts", kind: "inflow", startOffset: 2 * time.Hour, endOffset: 3 * time.Hour, status: "confirmed", wantRecord: true},
		{name: "confirmed transfer crossing opening is rejected", kind: "inflow", startOffset: -1 * time.Hour, endOffset: 2 * time.Hour, status: "confirmed", wantCode: "TRANSFER_CROSSES_PERIOD_BOUNDARY"},
		{name: "confirmed transfer crossing closing is rejected", kind: "outflow", startOffset: 23 * time.Hour, endOffset: 25 * time.Hour, status: "confirmed", wantCode: "TRANSFER_CROSSES_PERIOD_BOUNDARY"},
		{name: "draft transfer crossing boundaries never affects run", kind: "outflow", startOffset: -2 * time.Hour, endOffset: 26 * time.Hour, status: "draft", wantRecord: true},
		{name: "cancelled transfer crossing boundaries never affects run", kind: "outflow", startOffset: -2 * time.Hour, endOffset: 26 * time.Hour, status: "cancelled", wantRecord: true},
	}

	for index, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tank, start, end := seedBoundaryTank(t, db, fmt.Sprintf("TK-B%d", index))
			transfer := boundaryTransfer(tank.ID, tc.kind, start.Add(tc.startOffset), start.Add(tc.endOffset), 100000, tc.status)
			if err := db.Create(&transfer).Error; err != nil {
				t.Fatalf("create transfer: %v", err)
			}
			before := countBalanceRuns(t, db, tank.ID)
			_, err := svc.Run(context.Background(), dto.RunBalanceRequest{
				TankID: tank.ID, PeriodStart: &start, PeriodEnd: &end,
			}, boundaryActor)
			after := countBalanceRuns(t, db, tank.ID)
			if tc.wantCode != "" {
				if err == nil {
					t.Fatal("expected boundary rejection, run succeeded")
				}
				var appErr *api.Error
				if !errors.As(err, &appErr) || appErr.Code != tc.wantCode {
					t.Fatalf("error = %v, want code %s", err, tc.wantCode)
				}
				if after != before {
					t.Fatalf("rejected run created %d balance record(s)", after-before)
				}
				if appErr.Details["transfer_id"] != transfer.ID {
					t.Fatalf("details transfer_id = %v, want %d", appErr.Details["transfer_id"], transfer.ID)
				}
				if appErr.Details["crossed_at"] == nil {
					t.Fatal("rejection details must carry the out-of-bound timestamp")
				}
			} else {
				if err != nil {
					t.Fatalf("run: %v", err)
				}
				if after != before+1 {
					t.Fatalf("expected one new balance record, delta = %d", after-before)
				}
				items, err := transferRepo.ConfirmedForPeriod(context.Background(), tank.ID, start, end)
				if err != nil {
					t.Fatalf("reload period transfers: %v", err)
				}
				if tc.kind == "inflow" && len(items) != 1 {
					t.Fatalf("contained confirmed inflow must count, got %d transfers", len(items))
				}
			}
		})
	}
}

func TestRunBalanceConcurrentConfirmIsRecheckedWithinTransaction(t *testing.T) {
	db := boundaryTestDB(t)
	svc, transferRepo := boundaryServices(db)
	tank, start, end := seedBoundaryTank(t, db, "TK-CONCURRENT")

	// Simulate the race ordering: the transfer is still a draft when the run
	// transaction starts, then gets confirmed (crossing the opening boundary)
	// before the run re-reads transfers.
	draft := boundaryTransfer(tank.ID, "outflow", start.Add(-2*time.Hour), start.Add(2*time.Hour), 70000, "draft")
	if err := db.Create(&draft).Error; err != nil {
		t.Fatalf("create draft: %v", err)
	}
	confirmed, err := transferRepo.Transition(context.Background(), draft.ID, 1, "confirmed", "", boundaryActor)
	if err != nil {
		t.Fatalf("confirm crossing transfer: %v", err)
	}
	_, err = svc.Run(context.Background(), dto.RunBalanceRequest{
		TankID: tank.ID, PeriodStart: &start, PeriodEnd: &end,
	}, boundaryActor)
	if err == nil {
		t.Fatal("run must reject when the confirmed transfer crosses the period")
	}
	var appErr *api.Error
	if !errors.As(err, &appErr) || appErr.Code != "TRANSFER_CROSSES_PERIOD_BOUNDARY" {
		t.Fatalf("error = %v, want boundary rejection", err)
	}
	if appErr.Details["transfer_id"] != confirmed.ID {
		t.Fatalf("rejection must name transfer #%d, got %v", confirmed.ID, appErr.Details["transfer_id"])
	}
	if countBalanceRuns(t, db, tank.ID) != 0 {
		t.Fatal("failed concurrent run must not persist a balance record")
	}
}

func countBalanceRuns(t *testing.T, db *gorm.DB, tankID uint) int64 {
	t.Helper()
	var count int64
	if err := db.Model(&model.BalanceRun{}).Where("tank_id = ?", tankID).Count(&count).Error; err != nil {
		t.Fatalf("count balance runs: %v", err)
	}
	return count
}

func TestCalculateBalanceRunProducesReplayEvidence(t *testing.T) {
	curve, _ := balance.NewCapacityCurve([]float64{0, 15000})
	raw, _ := curve.Marshal()
	tank := model.StorageTank{
		ID: 1, TankCode: "TK-TEST", NominalCapacityM3: 180000, MinLevelM: 0, MaxLevelM: 12,
		ReferenceDensityKGM3: 452, ReferenceTemperatureC: -160, ThermalExpansionPerC: 0.0035,
		CapacityCurveJSON: datatypes.JSON(raw), CoefficientVersion: "CV-T1", TankStatus: "active",
	}
	start := time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)
	opening := model.MeasurementSnapshot{
		ID: 10, TankID: 1, MeasuredAt: start.Add(-time.Hour), CalculatedLiquidMassKG: 55_000_000,
		MeasurementUncertaintyPct: 0.30, QualityFlag: constants.QualityGood,
	}
	closing := model.MeasurementSnapshot{
		ID: 11, TankID: 1, MeasuredAt: end.Add(-time.Hour), CalculatedLiquidMassKG: 55_100_000,
		MeasurementUncertaintyPct: 0.32, QualityFlag: constants.QualityGood,
	}
	transfers := []model.TransferOperation{
		{ID: 20, OperationType: "inflow", MeasuredMassKG: 250000, MeasurementUncertaintyPct: 0.2},
		{ID: 21, OperationType: "outflow", MeasuredMassKG: 90000, MeasurementUncertaintyPct: 0.25},
	}
	calculated, snapshot, evidence, err := calculateBalanceRun(tank, opening, closing, transfers, start, end)
	if err != nil {
		t.Fatalf("calculate run: %v", err)
	}
	if calculated.NetTransferKG != 160000 || calculated.EstimatedBOGKG != 60000 {
		t.Fatalf("unexpected equation result: %+v", calculated)
	}
	if len(snapshot) < 200 || len(evidence) < 200 {
		t.Fatalf("expected replay snapshot and evidence, got %d/%d bytes", len(snapshot), len(evidence))
	}
	if calculated.DeviationLevel != constants.DeviationWithinUncertainty {
		t.Fatalf("unexpected deviation level: %s", calculated.DeviationLevel)
	}
}

func TestBalanceStateMachine(t *testing.T) {
	valid := []struct {
		from constants.BalanceStatus
		to   constants.BalanceStatus
	}{
		{constants.BalanceQueued, constants.BalanceCalculating},
		{constants.BalanceCalculating, constants.BalancePendingReview},
		{constants.BalancePendingReview, constants.BalanceAccepted},
		{constants.BalancePendingReview, constants.BalanceRejected},
	}
	for _, transition := range valid {
		if !constants.CanTransitionBalance(transition.from, transition.to) {
			t.Fatalf("expected valid transition %s -> %s", transition.from, transition.to)
		}
	}
	if constants.CanTransitionBalance(constants.BalanceAccepted, constants.BalanceCalculating) {
		t.Fatal("accepted result must remain immutable")
	}
}
