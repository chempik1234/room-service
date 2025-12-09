package config

import (
	"fmt"
	"github.com/ilyakaznacheev/cleanenv"
)

// Config is the main, assembled config type
type Config struct {
	Service RoomServiceConfig `yaml:"room_service" env-prefix:"ROOM_SERVICE_"`
	Log     LogConfig         `yaml:"log" env-prefix:"BACKEND_SERVICE_LOG_"`
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
