package app

import (
	"log/slog"

	"github.com/Novip1906/my-redis/internal/config"
	"github.com/Novip1906/my-redis/internal/network"
	"github.com/Novip1906/my-redis/internal/storage"
)

type App struct {
	server *network.TCPServer
	log    *slog.Logger
}

func NewApp(log *slog.Logger, cfg *config.Config) (*App, error) {
	memoryStorage := storage.NewStorage()

	server := network.NewTCPServer(cfg.Address, memoryStorage, log)

	return &App{
		server: server,
		log:    log,
	}, nil
}

func (a *App) Run() error {
	return a.server.Start()
}
