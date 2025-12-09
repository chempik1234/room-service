package config

// RoomServiceConfig is the config for microservice itself
type RoomServiceConfig struct {
	// GRPCPort is the port that Client gateways should connect to (requests and room streaming)
	GRPCPort int `yaml:"grpc_port" env:"GRPC_PORT"`
}

// LogConfig is the config struct for logging
//
// available log levels: "trace", "debug", "info", "warn", "error", "fatal", "panic"
type LogConfig struct {
	LogLevel string `yaml:"level" env:"LEVEL" envDefault:"info"`
}
