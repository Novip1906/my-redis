package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

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
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)
	<-stop

	log.Info("Shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	done := make(chan struct{})
	go func() {
		app.Stop()
		close(done)
	}()

	select {
	case <-done:
		log.Info("Graceful shutdown complete")
	case <-ctx.Done():
		log.Warn("Shutdown timeout reached, forcing exit")
	}

}
