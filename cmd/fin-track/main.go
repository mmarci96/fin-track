package main

import (
	"github.com/mmarci96/fin-track/internal/config"
	"github.com/mmarci96/fin-track/internal/repository"
	"github.com/mmarci96/fin-track/internal/router"
	"github.com/mmarci96/fin-track/pkg/logger"
)

func main() {
	cfg := config.NewConfig()
	logger.Init(cfg.LogLevel, cfg.RuntimeEnv)

	db, err := repository.NewDatabase(cfg.DatabaseURL())
	if err != nil {
		panic(err)
	}

	defer db.DB.Close()
	router := router.SetupRouter(db, cfg)

	addr := cfg.Host + ":" + cfg.Port
	err = router.Run(addr)
	if err != nil {
		panic(err)
	}
}
