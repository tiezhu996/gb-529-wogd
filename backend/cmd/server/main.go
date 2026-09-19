package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"lng-boiloff-gas-balance/backend/internal/config"
	"lng-boiloff-gas-balance/backend/internal/handler"
	"lng-boiloff-gas-balance/backend/internal/repository"
	"lng-boiloff-gas-balance/backend/internal/router"
	"lng-boiloff-gas-balance/backend/internal/service"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("configuration_failed", "error", err)
		os.Exit(1)
	}
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
	slog.SetDefault(log)
	db, err := config.OpenDatabase(cfg)
	if err != nil {
		log.Error("database_failed", "error", err)
		os.Exit(1)
	}

	tankRepo := repository.NewTankRepository(db)
	measurementRepo := repository.NewMeasurementRepository(db)
	transferRepo := repository.NewTransferRepository(db)
	balanceRepo := repository.NewBalanceRepository(db)
	supportRepo := repository.NewSupportRepository(db)

	authService := service.NewAuthService(supportRepo, cfg.JWTSecret)
	tankService := service.NewTankService(tankRepo)
	measurementService := service.NewMeasurementService(measurementRepo, tankRepo)
	transferService := service.NewTransferService(transferRepo, tankRepo)
	balanceService := service.NewBalanceService(balanceRepo, tankRepo, measurementRepo, transferRepo)
	auditService := service.NewAuditService(supportRepo)

	handlers := router.Handlers{
		Support:     handler.NewSupportHandler(authService, auditService),
		Tank:        handler.NewTankHandler(tankService),
		Measurement: handler.NewMeasurementHandler(measurementService),
		Transfer:    handler.NewTransferHandler(transferService),
		Balance:     handler.NewBalanceHandler(balanceService),
	}
	engine := router.New(log, authService, handlers, cfg.CORSOrigins)
	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           engine,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverErrors := make(chan error, 1)
	go func() {
		log.Info("server_started", "port", cfg.Port, "db_driver", cfg.DBDriver)
		serverErrors <- server.ListenAndServe()
	}()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	select {
	case sig := <-stop:
		log.Info("shutdown_requested", "signal", sig.String())
	case serveErr := <-serverErrors:
		if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			log.Error("server_failed", "error", serveErr)
		}
	}
	shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownContext); err != nil {
		log.Error("graceful_shutdown_failed", "error", err)
	}
	sqlDB, err := db.DB()
	if err == nil {
		_ = sqlDB.Close()
	}
	log.Info("server_stopped")
}
