package main

import (
	"DelayedNotifier/internal/app"
	"DelayedNotifier/pkg/logger"
	"context"

	"github.com/wb-go/wbf/config"
)

func main() {
	cfg := config.New()
	cfg.EnableEnv("")
	err := cfg.LoadEnvFiles(".env")
	if err != nil {
		panic(err)
	}

	ctx, err := logger.New(context.Background())
	if err != nil {
		panic(err)
	}

	newApp := app.NewApp(cfg, ctx)
	newApp.MustRun()
}
