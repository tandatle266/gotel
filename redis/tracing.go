package redis

import (
	"github.com/redis/go-redis/extra/redisotel/v9"
	"github.com/redis/go-redis/v9"
)

// Client wraps redis.Client with tracing
func InstrumentRedisClient(client *redis.Client) error {
    return redisotel.InstrumentTracing(client)
}

func InstrumentRedisClusterClient(client *redis.ClusterClient) error {
	return redisotel.InstrumentTracing(client)
}