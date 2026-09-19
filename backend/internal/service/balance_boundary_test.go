package service

import (
	"context"
	"strconv"
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

func TestClassifyPeriodTransfers(t *testing.T) {
	start := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)
	transfers := []model.TransferOperation{
		{ID: 1, OperationType: "inflow", StartAt: start.Add(time.Hour), EndAt: start.Add(2 * time.Hour)},    // contained
		{ID: 2, OperationType: "outflow", StartAt: start.Add(-time.Hour), EndAt: start.Add(time.Hour)},      // crosses start
		{ID: 3, OperationType: "outflow", StartAt: end.Add(-time.Hour), EndAt: end.Add(time.Hour)},          // crosses end
		{ID: 4, OperationType: "inflow", StartAt: start.Add(-2 * time.Hour), EndAt: end.Add(2 * time.Hour)}, // crosses both
		{ID: 5, OperationType: "outflow", StartAt: end, EndAt: end.Add(time.Hour)},                          // starts at end: outside, not a crossing
		{ID: 6, OperationType: "inflow", StartAt: start.Add(-time.Hour), EndAt: start},                      // ends at start: intersects as touching, not crossing
	}
	contained, violations := classifyPeriodTransfers(transfers, start, end)
	if len(contained) != 1 || contained[0].ID != 1 {
		t.Fatalf("contained = %v, want only transfer 1", idsOf(contained))
	}
	if len(violations) != 4 {
		t.Fatalf("violations = %d, want 4 (ID 2 start, ID 3 end, ID 4 both)", len(violations))
	}
	byKey := map[string]PeriodBoundaryViolation{}
	for _, violation := range violations {
		byKey[violation.Key()] = violation
	}
	if v, ok := byKey[key(2, "period_start")]; !ok || !v.OutOfBoundaryAt.Equal(transfer(transfers, 2).StartAt) {
		t.Fatalf("missing start crossing for transfer 2: %+v", violations)
	}
	if _, ok := byKey[key(3, "period_end")]; !ok {
		t.Fatalf("missing end crossing for transfer 3: %+v", violations)
	}
	if _, ok := byKey[key(4, "period_start")]; !ok {
		t.Fatalf("missing start crossing for spanning transfer 4")
	}
	if _, ok := byKey[key(4, "period_end")]; !ok {
		t.Fatalf("missing end crossing for spanning transfer 4")
	}
	for _, id := range []uint{5, 6} {
		for _, violation := range violations {
			if violation.TransferID == id {
				t.Fatalf("boundary-touching transfer %d must not be reported as crossing", id)
			}
		}
	}
}

func (v PeriodBoundaryViolation) Key() string { return key(v.TransferID, v.CrossedBoundary) }

func key(id uint, boundary string) string {
	return strconv.FormatUint(uint64(id), 10) + ":" + boundary
}

func transfer(items []model.TransferOperation, id uint) model.TransferOperation {
	for _, item := range items {
		if item.ID == id {
			return item
		}
	}
	return model.TransferOperation{}
}

func idsOf(items []model.TransferOperation) []uint {
	ids := make([]uint, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}

func newBoundaryTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:balance-boundary-" + strconv.FormatInt(time.Now().UnixNano(), 10) + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.AutoMigrate(
		&model.User{}, &model.StorageTank{}, &model.MeasurementSnapshot{},
		&model.TransferOperation{}, &model.BalanceRun{}, &model.AuditEvent{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func boundaryTestFixture(t *testing.T, db *gorm.DB) (model.StorageTank, time.Time, time.Time) {
	t.Helper()
	curve, _ := balance.NewCapacityCurve([]float64{0, 15000})
	raw, _ := curve.Marshal()
	tank := model.StorageTank{
		TankCode: "TK-BND", Name: "边界测试罐", NominalCapacityM3: 180000, MinLevelM: 0, MaxLevelM: 12,
		ReferenceDensityKGM3: 452, ReferenceTemperatureC: -160, ThermalExpansionPerC: 0.0035,
		CapacityCurveJSON: datatypes.JSON(raw), CoefficientVersion: "CV-BND", TankStatus: "active", Version: 1,
	}
	if err := db.Create(&tank).Error; err != nil {
		t.Fatalf("create tank: %v", err)
	}
	start := time.Date(2026, 9, 10, 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)
	snapshots := []model.MeasurementSnapshot{
		{TankID: tank.ID, MeasuredAt: start.Add(-time.Hour), CalculatedLiquidMassKG: 50_000_000, MeasurementUncertaintyPct: 0.3, QualityFlag: constants.QualityGood, SourceNote: "opening"},
		{TankID: tank.ID, MeasuredAt: end.Add(-time.Hour), CalculatedLiquidMassKG: 49_800_000, MeasurementUncertaintyPct: 0.3, QualityFlag: constants.QualityGood, SourceNote: "closing"},
	}
	for index := range snapshots {
		if err := db.Create(&snapshots[index]).Error; err != nil {
			t.Fatalf("create snapshot: %v", err)
		}
	}
	return tank, start, end
}

func boundaryService(db *gorm.DB) *BalanceService {
	tankRepo := repository.NewTankRepository(db)
	return NewBalanceService(
		repository.NewBalanceRepository(db), tankRepo,
		repository.NewMeasurementRepository(db), repository.NewTransferRepository(db),
	)
}

func boundaryActor() repository.Actor {
	return repository.Actor{UserID: 99, Email: "analyst@lng.local", Role: constants.RoleProcessAnalyst, RequestID: "req-boundary"}
}

func TestRunRejectsConfirmedTransferCrossingBoundaryWithoutRecord(t *testing.T) {
	db := newBoundaryTestDB(t)
	tank, start, end := boundaryTestFixture(t, db)
	crossing := model.TransferOperation{
		TankID: tank.ID, OperationType: "inflow",
		StartAt: start.Add(-2 * time.Hour), EndAt: start.Add(2 * time.Hour),
		MeasuredMassKG: 200000, MeasurementUncertaintyPct: 0.2,
		CounterpartyRef: "CROSSING-METER", OperationStatus: "confirmed", Version: 1, CreatedBy: 99,
	}
	if err := db.Create(&crossing).Error; err != nil {
		t.Fatalf("create crossing transfer: %v", err)
	}

	_, err := boundaryService(db).Run(context.Background(), dto.RunBalanceRequest{
		TankID: tank.ID, PeriodStart: &start, PeriodEnd: &end,
	}, boundaryActor())
	var appErr *api.Error
	if err == nil {
		t.Fatal("expected boundary crossing to be rejected")
	}
	if !asAPIError(err, &appErr) || appErr.Code != "TRANSFER_CROSSES_PERIOD_BOUNDARY" {
		t.Fatalf("error = %v, want TRANSFER_CROSSES_PERIOD_BOUNDARY", err)
	}
	details := appErr.Details["violations"].([]PeriodBoundaryViolation)
	if len(details) != 1 || details[0].TransferID != crossing.ID || details[0].CrossedBoundary != "period_start" {
		t.Fatalf("unexpected violation details: %+v", appErr.Details)
	}
	if !details[0].OutOfBoundaryAt.Equal(crossing.StartAt) {
		t.Fatalf("out-of-boundary time = %s, want %s", details[0].OutOfBoundaryAt, crossing.StartAt)
	}
	assertNoBalanceRows(t, db, tank.ID)
}

func TestRunIgnoresDraftAndCancelledAndAcceptsContained(t *testing.T) {
	db := newBoundaryTestDB(t)
	tank, start, end := boundaryTestFixture(t, db)
	transfers := []model.TransferOperation{
		// Draft that spans the whole period: must be ignored.
		{TankID: tank.ID, OperationType: "inflow", StartAt: start.Add(-time.Hour), EndAt: end.Add(time.Hour),
			MeasuredMassKG: 500000, MeasurementUncertaintyPct: 0.2, CounterpartyRef: "DRAFT", OperationStatus: "draft", Version: 1, CreatedBy: 99},
		// Cancelled crossing: must be ignored.
		{TankID: tank.ID, OperationType: "outflow", StartAt: start.Add(-time.Hour), EndAt: start.Add(time.Hour),
			MeasuredMassKG: 300000, MeasurementUncertaintyPct: 0.2, CounterpartyRef: "CANCELLED", OperationStatus: "cancelled", Version: 2, CreatedBy: 99},
		// Confirmed and fully contained: counts into the balance.
		{TankID: tank.ID, OperationType: "inflow", StartAt: start.Add(2 * time.Hour), EndAt: start.Add(3 * time.Hour),
			MeasuredMassKG: 100000, MeasurementUncertaintyPct: 0.2, CounterpartyRef: "CONTAINED", OperationStatus: "confirmed", Version: 1, CreatedBy: 99},
	}
	for index := range transfers {
		if err := db.Create(&transfers[index]).Error; err != nil {
			t.Fatalf("create transfer: %v", err)
		}
	}
	run, err := boundaryService(db).Run(context.Background(), dto.RunBalanceRequest{
		TankID: tank.ID, PeriodStart: &start, PeriodEnd: &end,
	}, boundaryActor())
	if err != nil {
		t.Fatalf("run with contained transfer: %v", err)
	}
	if run.NetTransferKG != 100000 {
		t.Fatalf("net transfer = %f, want 100000 (draft/cancelled excluded)", run.NetTransferKG)
	}
	if run.BalanceStatus != constants.BalanceCalculating {
		t.Fatalf("status = %s, want calculating", run.BalanceStatus)
	}
	var audits int64
	if err := db.Model(&model.AuditEvent{}).Where("entity_type = ? AND entity_id = ?", "balance_run", run.ID).Count(&audits).Error; err != nil {
		t.Fatalf("count audits: %v", err)
	}
	if audits != 2 {
		t.Fatalf("audits for run = %d, want queued + calculated", audits)
	}
}

func TestRunWithContainedTransferThenConcurrentCancel(t *testing.T) {
	db := newBoundaryTestDB(t)
	tank, start, end := boundaryTestFixture(t, db)
	transfer := model.TransferOperation{
		TankID: tank.ID, OperationType: "outflow", StartAt: start.Add(2 * time.Hour), EndAt: start.Add(3 * time.Hour),
		MeasuredMassKG: 80000, MeasurementUncertaintyPct: 0.25, CounterpartyRef: "CONTENTION",
		OperationStatus: "confirmed", Version: 1, CreatedBy: 99,
	}
	if err := db.Create(&transfer).Error; err != nil {
		t.Fatalf("create transfer: %v", err)
	}
	run, err := boundaryService(db).Run(context.Background(), dto.RunBalanceRequest{
		TankID: tank.ID, PeriodStart: &start, PeriodEnd: &end,
	}, boundaryActor())
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if run.NetTransferKG != -80000 {
		t.Fatalf("net transfer = %f, want -80000", run.NetTransferKG)
	}
	// Cancelling after the committed run must not mutate or delete the stored result.
	if _, err := repository.NewTransferRepository(db).Transition(context.Background(), transfer.ID, 1, "cancelled", "期间后取消测试", boundaryActor()); err != nil {
		t.Fatalf("cancel transfer: %v", err)
	}
	stored, err := repository.NewBalanceRepository(db).Get(context.Background(), run.ID)
	if err != nil {
		t.Fatalf("reload run: %v", err)
	}
	if stored.NetTransferKG != -80000 || stored.BalanceStatus != constants.BalanceCalculating {
		t.Fatalf("stored run mutated by later cancellation: %+v", stored)
	}
}

func asAPIError(err error, target **api.Error) bool {
	for err != nil {
		if apiErr, ok := err.(*api.Error); ok {
			*target = apiErr
			return true
		}
		type wrapper interface{ Unwrap() error }
		next, ok := err.(wrapper)
		if !ok {
			return false
		}
		err = next.Unwrap()
	}
	return false
}

func assertNoBalanceRows(t *testing.T, db *gorm.DB, tankID uint) {
	t.Helper()
	var runs int64
	if err := db.Model(&model.BalanceRun{}).Where("tank_id = ?", tankID).Count(&runs).Error; err != nil {
		t.Fatalf("count runs: %v", err)
	}
	if runs != 0 {
		t.Fatalf("rejected calculation left %d balance run records", runs)
	}
	var audits int64
	if err := db.Model(&model.AuditEvent{}).Where("entity_type = ?", "balance_run").Count(&audits).Error; err != nil {
		t.Fatalf("count balance audits: %v", err)
	}
	if audits != 0 {
		t.Fatalf("rejected calculation left %d balance audit events", audits)
	}
}
