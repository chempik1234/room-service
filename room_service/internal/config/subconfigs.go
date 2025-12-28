package config

import "github.com/chempik1234/room-service/pkg/config"

// RoomServiceConfig - config for microservice itself
type RoomServiceConfig struct {
	// GRPCPort - port that Client gateways should connect to (requests and room streaming)
	GRPCPort int `yaml:"grpc_port" env:"GRPC_PORT"`
	// RetryStrategy - retries for gRPC operations
	RetryStrategy config.RetryStrategyConfig `yaml:"retry" env-prefix:"RETRY_"`
}

// LogConfig - config struct for logging
//
// available log levels: "trace", "debug", "info", "warn", "error", "fatal", "panic"
// TOO: remove that or add leveling to logs
type LogConfig struct {
	LogLevel string `yaml:"level" env:"LEVEL" envDefault:"info"`
}

// MongoDBRoomsRepoConfig - config for rooms repo params
type MongoDBRoomsRepoConfig struct {
	Database        string `yaml:"database" env:"DATABASE" envDefault:"rooms_db"`
	RoomsCollection string `yaml:"rooms_collection" env:"ROOMS_COLLECTION" envDefault:"rooms"`
	ReadConcern     string `yaml:"read_concern" env:"READ_CONCERN" envDefault:"available"`
	WriteConcern    string `yaml:"write_concern" env:"WRITE_CONCERN" envDefault:"w: majority, j: true"`
}
