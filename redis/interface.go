package oteltracingredis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// CacheInterface defines the interface for cache operations
type CacheInterface interface {
	GetFromCache(ctx context.Context, key string, target any) error
	SetToCache(ctx context.Context, key string, value any, expiration time.Duration) error
	DeleteFromCache(ctx context.Context, key string) error
	Client() *redis.Client
	Close() error
}

// Ensure both Client and TracedClient implement CacheInterface
// var _ CacheInterface = (*Client)(nil)
// var _ CacheInterface = (*TracedClient)(nil)