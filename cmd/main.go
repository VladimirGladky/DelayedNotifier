package main

import (
	"github.com/wb-go/wbf/config"
	"github.com/wb-go/wbf/zlog"
)

func main() {
	cfg := config.New()
	err := cfg.LoadEnvFiles(".env")
	if err != nil {
		panic(err)
	}
	zlog.Init()

}
