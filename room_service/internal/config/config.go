package config

import (
	"fmt"
	"github.com/chempik1234/super-danis-library-golang/v2/pkg/mongodb"
	"github.com/chempik1234/super-danis-library-golang/v2/pkg/redis"
	"github.com/ilyakaznacheev/cleanenv"
)

// Config is the main, assembled config type
type Config struct {
	Service RoomServiceConfig `yaml:"room_service" env-prefix:"ROOM_SERVICE_"`
	Log     LogConfig         `yaml:"log" env-prefix:"ROOM_SERVICE_LOG_"`
	MongoDB mongodb.Config    `yaml:"mongodb" env-prefix:"ROOM_SERVICE_MONGODB_"`
	Redis   redis.Config      `yaml:"redis" env-prefix:"ROOM_SERVICE_REDIS_"`
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
