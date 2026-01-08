package app

import (
	"log/slog"

	"github.com/Novip1906/my-redis/internal/aof"
	"github.com/Novip1906/my-redis/internal/compute"
	"github.com/Novip1906/my-redis/internal/config"
	"github.com/Novip1906/my-redis/internal/network"
)

type App struct {
	server     *network.TCPServer
	parser     *compute.Parser
	cfg        *config.Config
	aofService *aof.AOF
	log        *slog.Logger
}

func NewApp(log *slog.Logger, cfg *config.Config, storage compute.Storage) (*App, error) {
	parser := compute.NewParser(storage)

	aofService, err := aof.NewAOF(cfg.AOFPath)
	if err != nil {
		log.Error("Failed to init AOF", "error", err)
	}

	server := network.NewTCPServer(cfg.Address, parser, aofService, log)

	return &App{
		server:     server,
		log:        log,
		parser:     parser,
		aofService: aofService,
		cfg:        cfg,
	}, nil
}

func (a *App) Run() error {
	a.log.Info("Restoring data from AOF...")
	err := aof.ReadAll(a.cfg.AOFPath, func(line string) {
		a.parser.ProcessCommand(line)
	})
	if err != nil {
		a.log.Error("Failed to restore AOF", "error", err)
	}
	a.log.Info("Data restored")
	return a.server.Start()
}

func (a *App) Stop() {
	a.aofService.Close()
	a.server.Stop()
}
