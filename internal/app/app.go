package app

import (
	"log/slog"

	"github.com/Novip1906/my-redis/internal/config"
	"github.com/Novip1906/my-redis/internal/network"
)

type App struct {
	server *network.TCPServer
	log    *slog.Logger
}

func NewApp(log *slog.Logger, cfg *config.Config, storage network.Storage) (*App, error) {
	server := network.NewTCPServer(cfg.Address, storage, log)

	return &App{
		server: server,
		log:    log,
	}, nil
}

func (a *App) Run() error {
	return a.server.Start()
}

func (a *App) Stop() {
	a.server.Stop()
}
