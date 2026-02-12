package main

import (
	"DelayedNotifier/pkg/logger"
	"DelayedNotifier/pkg/postgres"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	pg "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/wb-go/wbf/config"
	"go.uber.org/zap"
)

func main() {
	ctx := context.Background()

	ctx, err := logger.New(ctx)
	if err != nil {
		panic(err)
	}

	cfg := config.New()
	cfg.EnableEnv("")
	if err := cfg.LoadEnvFiles(".env"); err != nil {
		panic(err)
	}

	db, err := postgres.NewPostgres(cfg)
	if err != nil {
		panic(err)
	}

	command := "up"
	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	logger.GetLoggerFromCtx(ctx).Info("Running migration command", zap.String("command", command))

	driver, err := pg.WithInstance(db.Master, &pg.Config{})
	if err != nil {
		logger.GetLoggerFromCtx(ctx).Error("Failed to create migration driver", zap.Error(err))
		panic(err)
	}

	migrationsPath, err := filepath.Abs("./migrations")
	if err != nil {
		logger.GetLoggerFromCtx(ctx).Error("Failed to get absolute path", zap.Error(err))
		panic(err)
	}

	migrationsURL := fmt.Sprintf("file://%s", migrationsPath)
	logger.GetLoggerFromCtx(ctx).Info("Using migrations path",
		zap.String("path", migrationsPath),
		zap.String("url", migrationsURL))

	m, err := migrate.NewWithDatabaseInstance(
		migrationsURL,
		"postgres",
		driver,
	)
	if err != nil {
		logger.GetLoggerFromCtx(ctx).Error("Failed to create migrate instance", zap.Error(err))
		panic(err)
	}

	switch command {
	case "up":
		if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			logger.GetLoggerFromCtx(ctx).Error("Migration up failed", zap.Error(err))
			panic(err)
		}
		logger.GetLoggerFromCtx(ctx).Info("Migrations applied successfully!")

	case "down":
		if err := m.Steps(-1); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			logger.GetLoggerFromCtx(ctx).Error("Migration down failed", zap.Error(err))
			panic(err)
		}
		logger.GetLoggerFromCtx(ctx).Info("Last migration rolled back!")

	case "reset":
		if err := m.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			logger.GetLoggerFromCtx(ctx).Error("Migration reset failed", zap.Error(err))
			panic(err)
		}
		logger.GetLoggerFromCtx(ctx).Info("All migrations rolled back!")

	case "version":
		version, dirty, err := m.Version()
		if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
			logger.GetLoggerFromCtx(ctx).Error("Migration version failed", zap.Error(err))
			panic(err)
		}
		if errors.Is(err, migrate.ErrNilVersion) {
			logger.GetLoggerFromCtx(ctx).Info("No migrations applied yet")
		} else {
			logger.GetLoggerFromCtx(ctx).Info("Current migration version",
				zap.Uint("version", version),
				zap.Bool("dirty", dirty))
		}

	default:
		logger.GetLoggerFromCtx(ctx).Error("Unknown command", zap.String("command", command))
		os.Exit(1)
	}
}
