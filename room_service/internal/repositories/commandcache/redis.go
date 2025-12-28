package commandcache

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-redis/redis/v8"
	"time"
)

// RedisCommandCache - impl of ports.CommandIdShortCache with Redis
//
// Before executing command, check id with Exists - if it's stored, skip the command -
// it's already in execution (or finished)
type RedisCommandCache struct {
	client *redis.Client
	ttl    time.Duration
}

// NewRedisCommandCache - create new RedisCommandCache
func NewRedisCommandCache(client *redis.Client, ttlMs int) *RedisCommandCache {
	return &RedisCommandCache{
		client: client,
		ttl:    time.Duration(ttlMs) * time.Millisecond,
	}
}

// Exists - check if commandID is already saved in Redis
//
// if it is, skip the command - it's already in execution (or finished)
func (s *RedisCommandCache) Exists(ctx context.Context, commandID string) (bool, error) {
	key := s.generateKey(commandID)
	_, err := s.client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return false, nil // Not found
		}
		return false, fmt.Errorf("error querying redis: %w", err) // Other errors
	}

	return true, nil
}

// Save - store commandID in Redis
func (s *RedisCommandCache) Save(ctx context.Context, commandID string) error {
	_, err := s.client.Set(ctx, s.generateKey(commandID), "", s.ttl).Result()
	if err != nil {
		return fmt.Errorf("error saving command in redis: %w", err)
	}
	return nil
}

func (s *RedisCommandCache) generateKey(id string) string {
	return fmt.Sprintf("command_%s", id)
}
