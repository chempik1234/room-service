package config

import (
	"github.com/wb-go/wbf/retry"
	"time"
)

// RetryStrategyConfig is the retry strategy config struct
//
// specifies how retry operations will be handled
//
// supposed to be used for multiple things like RABBITMQ_RETRIES, EMAIL_RETRIES, etc.
type RetryStrategyConfig struct {
	Attempts          int     `env:"ATTEMPTS" env-default:"3"`
	DelayMilliseconds int     `env:"DELAY_MILLISECONDS" env-default:"500"`
	Backoff           float64 `env:"BACKOFF" env-default:"1"`
}

// ToStrategy converts an already read config to usable format which is retry.Strategy
//
// Example:
//
//	redisRetryStrategy := cfg.RedisRetryConfig.ToStrategy()
func (cfg *RetryStrategyConfig) ToStrategy() retry.Strategy {
	return retry.Strategy{
		Attempts: cfg.Attempts,
		Delay:    time.Duration(cfg.DelayMilliseconds) * time.Millisecond,
		Backoff:  cfg.Backoff,
	}
}
