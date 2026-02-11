package main

import (
	"DelayedNotifier/internal/app"
	"context"

	"github.com/wb-go/wbf/config"
	"github.com/wb-go/wbf/zlog"
)

func main() {
	cfg := config.New()
	cfg.EnableEnv("")
	err := cfg.LoadEnvFiles(".env")
	if err != nil {
		panic(err)
	}
	zlog.Init()
	newApp := app.NewApp(cfg, context.Background())
	newApp.MustRun()
}
