package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/Novip1906/my-redis/internal/app"
	"github.com/Novip1906/my-redis/internal/config"
	"github.com/Novip1906/my-redis/internal/storage"
	"github.com/Novip1906/my-redis/pkg/logging"
)

func main() {
	log := logging.SetupLogger()

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Error("Config load error", "error", err)
		return
	}

	memoryStorage := storage.NewMemoryStorage()

	app, err := app.NewApp(log, cfg, memoryStorage)
	if err != nil {
		log.Error("App create error", "error", err)
		return
	}

	go func() {
		if err = app.Run(); err != nil {
			log.Error("App run error", "error", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	sign := <-stop
	log.Info("Stopping application...", "signal", sign)

	app.Stop()
	log.Info("Application stopped")
}
