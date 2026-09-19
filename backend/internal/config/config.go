package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"lng-boiloff-gas-balance/backend/internal/balance"
	"lng-boiloff-gas-balance/backend/internal/constants"
	"lng-boiloff-gas-balance/backend/internal/model"
)

type Config struct {
	Port                  string
	DBDriver              string
	DBDSN                 string
	DBHost                string
	DBPort                string
	DBName                string
	DBUser                string
	DBPassword            string
	DBSSLMode             string
	DBAutoMigrate         bool
	SeedData              bool
	JWTSecret             string
	CORSOrigins           []string
	DefaultUncertaintyPct float64
	LogLevel              slog.Level
}

func Load() (Config, error) {
	cfg := Config{
		Port:                  env("PORT", "8080"),
		DBDriver:              strings.ToLower(env("DB_DRIVER", "postgres")),
		DBDSN:                 os.Getenv("DB_DSN"),
		DBHost:                env("DB_HOST", "127.0.0.1"),
		DBPort:                env("DB_INTERNAL_PORT", "5432"),
		DBName:                env("DB_NAME", "lng_balance"),
		DBUser:                env("DB_USER", "lng_app"),
		DBPassword:            env("DB_PASSWORD", "lng-local-development-password"),
		DBSSLMode:             env("DB_SSLMODE", "disable"),
		DBAutoMigrate:         envBool("DB_AUTO_MIGRATE", true),
		SeedData:              envBool("SEED_DATA", true),
		JWTSecret:             os.Getenv("JWT_SECRET"),
		CORSOrigins:           splitCSV(env("CORS_ORIGINS", "http://127.0.0.1:18529,http://localhost:18529")),
		DefaultUncertaintyPct: envFloat("DEFAULT_UNCERTAINTY_PCT", 0.35),
		LogLevel:              parseLogLevel(env("LOG_LEVEL", "info")),
	}
	if len(cfg.JWTSecret) < 32 {
		return Config{}, fmt.Errorf("JWT_SECRET must contain at least 32 characters")
	}
	if cfg.DBDriver != "postgres" && cfg.DBDriver != "sqlite" {
		return Config{}, fmt.Errorf("unsupported DB_DRIVER %q", cfg.DBDriver)
	}
	if cfg.DBDriver == "sqlite" && cfg.DBDSN == "" {
		return Config{}, errors.New("DB_DSN is required for sqlite")
	}
	if err := balance.ValidateUncertainty(cfg.DefaultUncertaintyPct); err != nil {
		return Config{}, fmt.Errorf("DEFAULT_UNCERTAINTY_PCT: %w", err)
	}
	return cfg, nil
}

func OpenDatabase(cfg Config) (*gorm.DB, error) {
	var dialector gorm.Dialector
	if cfg.DBDriver == "sqlite" {
		dialector = sqlite.Open(cfg.DBDSN)
	} else {
		dsn := fmt.Sprintf(
			"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=UTC",
			cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBSSLMode,
		)
		dialector = postgres.Open(dsn)
	}
	db, err := gorm.Open(dialector, &gorm.Config{Logger: logger.Default.LogMode(logger.Silent), TranslateError: true})
	if err != nil {
		return nil, fmt.Errorf("open %s database: %w", cfg.DBDriver, err)
	}
	if cfg.DBAutoMigrate {
		if err := migrate(db); err != nil {
			return nil, fmt.Errorf("migrate database: %w", err)
		}
	}
	if cfg.SeedData {
		if err := seed(db); err != nil {
			return nil, fmt.Errorf("seed database: %w", err)
		}
	}
	return db, nil
}

func migrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&model.User{},
		&model.StorageTank{},
		&model.MeasurementSnapshot{},
		&model.TransferOperation{},
		&model.BalanceRun{},
		&model.AuditEvent{},
	)
}

func seed(db *gorm.DB) error {
	var count int64
	if err := db.Model(&model.User{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	return db.Transaction(func(tx *gorm.DB) error {
		hash, err := bcrypt.GenerateFromPassword([]byte("LngBalance!2026"), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("hash seed password: %w", err)
		}
		users := []model.User{
			{Email: "analyst@lng.local", DisplayName: "LNG 工艺分析员", PasswordHash: string(hash), Role: constants.RoleProcessAnalyst, Active: true},
			{Email: "reviewer@lng.local", DisplayName: "独立计量复核员", PasswordHash: string(hash), Role: constants.RoleReviewer, Active: true},
			{Email: "admin@lng.local", DisplayName: "系统管理员", PasswordHash: string(hash), Role: constants.RoleAdmin, Active: true},
		}
		if err := tx.Create(&users).Error; err != nil {
			return fmt.Errorf("create seed users: %w", err)
		}
		curveA, _ := balance.NewCapacityCurve([]float64{0, 15000})
		curveB, _ := balance.NewCapacityCurve([]float64{0, 13500})
		curveAJSON, _ := curveA.Marshal()
		curveBJSON, _ := curveB.Marshal()
		tanks := []model.StorageTank{
			{TankCode: "TK-101", Name: "北区 LNG 储罐", NominalCapacityM3: 180000, MinLevelM: 0, MaxLevelM: 12, ReferenceDensityKGM3: 452, ReferenceTemperatureC: -160, ThermalExpansionPerC: 0.0035, CapacityCurveJSON: datatypes.JSON(curveAJSON), CoefficientVersion: "CV-2026.08-A", TankStatus: "active", Version: 1},
			{TankCode: "TK-202", Name: "南区 LNG 储罐", NominalCapacityM3: 162000, MinLevelM: 0, MaxLevelM: 12, ReferenceDensityKGM3: 449, ReferenceTemperatureC: -160, ThermalExpansionPerC: 0.0034, CapacityCurveJSON: datatypes.JSON(curveBJSON), CoefficientVersion: "CV-2026.07-B", TankStatus: "active", Version: 1},
		}
		if err := tx.Create(&tanks).Error; err != nil {
			return fmt.Errorf("create seed tanks: %w", err)
		}
		now := time.Now().UTC().Truncate(time.Second)
		snapshots := []model.MeasurementSnapshot{
			seedSnapshot(tanks[0], now.Add(-25*time.Hour), 8.20, -160.4, 111, 451.8, 0.28, constants.QualityGood, "班次交接人工复核", users[0].ID),
			seedSnapshot(tanks[0], now.Add(-1*time.Hour), 8.14, -159.9, 113, 451.2, 0.30, constants.QualityGood, "期间末液位温度密度复核", users[0].ID),
			seedSnapshot(tanks[1], now.Add(-25*time.Hour), 7.75, -160.2, 108, 448.8, 0.32, constants.QualityGood, "南区期初离线计量", users[0].ID),
			seedSnapshot(tanks[1], now.Add(-1*time.Hour), 7.70, -159.7, 110, 448.3, 0.34, constants.QualitySuspect, "温度稳定性待复核但可计算", users[0].ID),
		}
		if err := tx.Create(&snapshots).Error; err != nil {
			return fmt.Errorf("create seed snapshots: %w", err)
		}
		transfers := []model.TransferOperation{
			{TankID: tanks[0].ID, OperationType: "inflow", StartAt: now.Add(-20 * time.Hour), EndAt: now.Add(-19 * time.Hour), MeasuredMassKG: 425000, MeasurementUncertaintyPct: 0.22, CounterpartyRef: "JETTY-A-METER-01", OperationStatus: "confirmed", Version: 1, CreatedBy: users[0].ID},
			{TankID: tanks[0].ID, OperationType: "outflow", StartAt: now.Add(-8 * time.Hour), EndAt: now.Add(-7 * time.Hour), MeasuredMassKG: 96500, MeasurementUncertaintyPct: 0.25, CounterpartyRef: "SENDOUT-METER-02", OperationStatus: "confirmed", Version: 1, CreatedBy: users[0].ID},
			{TankID: tanks[1].ID, OperationType: "outflow", StartAt: now.Add(-12 * time.Hour), EndAt: now.Add(-11 * time.Hour), MeasuredMassKG: 82000, MeasurementUncertaintyPct: 0.30, CounterpartyRef: "SENDOUT-METER-03", OperationStatus: "confirmed", Version: 1, CreatedBy: users[0].ID},
		}
		if err := tx.Create(&transfers).Error; err != nil {
			return fmt.Errorf("create seed transfers: %w", err)
		}
		audit := model.AuditEvent{
			RequestID: "seed-bootstrap", UserID: users[2].ID, ActorEmail: users[2].Email,
			Action: "system.seeded", EntityType: "system", EntityID: 1,
			BeforeJSON: "{}", AfterJSON: `{"tanks":2,"snapshots":4,"confirmed_transfers":3}`,
			CreatedAt: now,
		}
		if err := tx.Create(&audit).Error; err != nil {
			return fmt.Errorf("create seed audit: %w", err)
		}
		return nil
	})
}

func seedSnapshot(tank model.StorageTank, measuredAt time.Time, level, temperature, pressure, density, uncertainty float64, quality constants.QualityFlag, note string, userID uint) model.MeasurementSnapshot {
	curve, _ := balance.ParseCapacityCurve(tank.CapacityCurveJSON)
	result, _ := balance.CalculateSnapshotMass(balance.SnapshotMassInput{
		LevelM: level, MinimumLevelM: tank.MinLevelM, MaximumLevelM: tank.MaxLevelM,
		NominalCapacityM3: tank.NominalCapacityM3, DensityKGM3: density,
		TemperatureC: temperature, ReferenceTemperatureC: tank.ReferenceTemperatureC,
		ThermalExpansionPerC: tank.ThermalExpansionPerC, Curve: curve,
	})
	return model.MeasurementSnapshot{
		TankID: tank.ID, MeasuredAt: measuredAt, LiquidLevelM: level, LiquidTempC: temperature,
		VaporPressureKPA: pressure, DensityKGM3: density, CalculatedVolumeM3: result.VolumeM3,
		TemperatureDensityKGM3: result.CorrectedDensityKGM3, CalculatedLiquidMassKG: result.LiquidMassKG,
		MeasurementUncertaintyPct: uncertainty, QualityFlag: quality, SourceNote: note, CreatedBy: userID,
	}
}

func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envFloat(key string, fallback float64) float64 {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func parseLogLevel(value string) slog.Level {
	switch strings.ToLower(value) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
