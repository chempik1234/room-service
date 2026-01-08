package config

import "github.com/chempik1234/room-service/pkg/config"

// RoomServiceConfig - config for microservice itself
type RoomServiceConfig struct {
	// GRPCPort - port that Client gateways should connect to (requests and room streaming)
	GRPCPort int `yaml:"grpc_port" env:"GRPC_PORT" env-default:"50050"`
	// RetryStrategy - retries for gRPC operations
	RetryStrategy config.RetryStrategyConfig `yaml:"retry" env-prefix:"RETRY_"`
	// UseAuth - enable API key authentication
	UseAuth bool `env:"USE_AUTH" env-default:"true"`
}

// LogConfig - config struct for logging
//
// available log levels: "trace", "debug", "info", "warn", "error", "fatal", "panic"
// TOO: remove that or add leveling to logs
type LogConfig struct {
	LogLevel string `yaml:"level" env:"LEVEL" env-default:"info"`
}

// MongoDBRoomsRepoConfig - config for rooms repo params
type MongoDBRoomsRepoConfig struct {
	Database        string `yaml:"database" env:"DATABASE" env-default:"rooms_db"`
	RoomsCollection string `yaml:"rooms_collection" env:"ROOMS_COLLECTION" env-default:"rooms"`
	ReadConcern     string `yaml:"read_concern" env:"READ_CONCERN" env-default:"available"`
	WriteConcern    string `yaml:"write_concern" env:"WRITE_CONCERN" env-default:"w: majority, j: true"`
}
