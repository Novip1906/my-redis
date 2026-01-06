package logging

import (
	"log/slog"
	"os"
)

func SetupLogger() *slog.Logger {
	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})

	return slog.New(handler)
}
