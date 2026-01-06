package main

import (
	"github.com/Novip1906/my-redis/internal/app"
	"github.com/Novip1906/my-redis/internal/config"
	"github.com/Novip1906/my-redis/pkg/logging"
)

func main() {
	log := logging.SetupLogger()

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Error("Config load error", "error", err)
		return
	}

	app, err := app.NewApp(log, cfg)
	if err != nil {
		log.Error("App create error", "error", err)
		return
	}

	if err = app.Run(); err != nil {
		log.Error("App run error", "error", err)
	}
}
