package main

import "github.com/wb-go/wbf/config"

func main() {
	cfg := config.New()
	err := cfg.LoadEnvFiles(".env")
	if err != nil {
		panic(err)
	}

}
