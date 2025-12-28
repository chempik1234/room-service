package config

import "github.com/chempik1234/room-service/pkg/config"

// RoomServiceConfig is the config for microservice itself
type RoomServiceConfig struct {
	// GRPCPort is the port that Client gateways should connect to (requests and room streaming)
	GRPCPort int `yaml:"grpc_port" env:"GRPC_PORT"`
	// RetryStrategy - retries for gRPC operations
	RetryStrategy config.RetryStrategyConfig `yaml:"retry" env-prefix:"RETRY_"`
}

// LogConfig is the config struct for logging
//
// available log levels: "trace", "debug", "info", "warn", "error", "fatal", "panic"
// TOO: remove that or add leveling to logs
type LogConfig struct {
	LogLevel string `yaml:"level" env:"LEVEL" envDefault:"info"`
}
