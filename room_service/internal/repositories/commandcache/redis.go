package commandcache

import "context"

// RedisCommandCache - impl of ports.CommandIdShortCache with Redis
type RedisCommandCache struct {
}

// NewRedisCommandCache - create new RedisCommandCache
func NewRedisCommandCache() *RedisCommandCache {
	return &RedisCommandCache{}
}

func (r RedisCommandCache) Set(ctx context.Context, key string, value struct{}) error {
	//TODO implement me
	panic("implement me")
}

func (r RedisCommandCache) Get(ctx context.Context, key string) (struct{}, bool, error) {
	//TODO implement me
	panic("implement me")
}

func (r RedisCommandCache) GetKeys() []string {
	//TODO implement me
	panic("implement me")
}

func (r RedisCommandCache) GetKeysAmount() int {
	//TODO implement me
	panic("implement me")
}
