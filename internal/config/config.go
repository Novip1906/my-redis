package config

import (
	"os"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
)

type Config struct {
	Address string `yaml:"address" env-default:":6379"`
}

func LoadConfig() (*Config, error) {
	godotenv.Load()

	path := os.Getenv("CONFIG_PATH")
	if path == "" {
		path = "./configs/config.yaml"
	}

	var cfg Config
	if err := cleanenv.ReadConfig(path, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
