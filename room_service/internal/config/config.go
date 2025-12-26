package config

import (
	"fmt"
	"github.com/chempik1234/room-service/pkg/config"
	"github.com/ilyakaznacheev/cleanenv"
)

// Config is the main, assembled config type
type Config struct {
	Service       RoomServiceConfig          `yaml:"room_service" env-prefix:"ROOM_SERVICE_"`
	Log           LogConfig                  `yaml:"log" env-prefix:"BACKEND_SERVICE_LOG_"`
	RetryStrategy config.RetryStrategyConfig `yaml:"retry_strategy" env-prefix:"BACKEND_SERVICE_RETRY_"`
}

// TryRead tries to read config from ENV and returns it on success
func TryRead() (*Config, error) {
	var cfg Config
	if err := cleanenv.ReadEnv(&cfg); err != nil {
		return nil,
			fmt.Errorf("failed to read env variables after accessing .env: %w", err)
	}
	return &cfg, nil
}
