package app

import (
	"log/slog"

	"github.com/Novip1906/my-redis/internal/compute"
	"github.com/Novip1906/my-redis/internal/config"
	"github.com/Novip1906/my-redis/internal/network"
)

type App struct {
	server *network.TCPServer
	log    *slog.Logger
}

func NewApp(log *slog.Logger, cfg *config.Config, storage compute.Storage) (*App, error) {
	parser := compute.NewParser(storage)

	server := network.NewTCPServer(cfg.Address, parser, log)

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
