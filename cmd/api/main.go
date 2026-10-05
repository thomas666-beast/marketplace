package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/thomas666-beast/marketplace/internal/auth"
	"github.com/thomas666-beast/marketplace/internal/catalog"
	"github.com/thomas666-beast/marketplace/internal/config"
	"github.com/thomas666-beast/marketplace/internal/httpserver"
	"github.com/thomas666-beast/marketplace/internal/i18n"
	"github.com/thomas666-beast/marketplace/internal/orders"
	"github.com/thomas666-beast/marketplace/internal/postgres"
	"github.com/thomas666-beast/marketplace/internal/users"
	"github.com/thomas666-beast/marketplace/internal/delivery"
	"github.com/thomas666-beast/marketplace/internal/seller"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		logger.Error("config load failed", "err", err)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	bundle, err := i18n.NewBundle()
	if err != nil {
		logger.Error("i18n load failed", "err", err)
		os.Exit(1)
	}
	logger.Info("i18n loaded")

	if cfg.AppEnv == "development" {
		logger.Info("running migrations")
		if err := postgres.RunMigrations(cfg.PostgresDSN()); err != nil {
			logger.Error("migrations failed", "err", err)
			os.Exit(1)
		}
		logger.Info("migrations applied")
	}

	db, err := postgres.Connect(ctx, cfg.PostgresDSN())
	if err != nil {
		logger.Error("postgres connect failed", "err", err)
		os.Exit(1)
	}
	defer db.Close()
	logger.Info("postgres connected")

	tokens, err := auth.NewTokenManager(cfg.JWTSecret)
	if err != nil {
		logger.Error("token manager init failed", "err", err)
		os.Exit(1)
	}

	userRepo := users.NewRepository(db.Pool)
	categoryRepo := catalog.NewCategoryRepository(db.Pool)
	productRepo := catalog.NewProductRepository(db.Pool)
	orderRepo := orders.NewRepository(db.Pool)
	pickupPointRepo := delivery.NewPickupPointRepository(db.Pool)
	deliveryRepo := delivery.NewRepository(db.Pool)
	addressRepo := seller.NewAddressRepository(db.Pool)

	srv := httpserver.New(
		cfg.AppPort, db, bundle, logger,
		userRepo, categoryRepo, productRepo, orderRepo,
		pickupPointRepo, deliveryRepo, addressRepo, tokens,
	)

	go func() {
		if err := srv.Start(); err != nil {
			logger.Error("http server stopped", "err", err)
		}
	}()

	<-ctx.Done()
	logger.Info("shutdown signal received")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown failed", "err", err)
	}
	logger.Info("goodbye")
}
