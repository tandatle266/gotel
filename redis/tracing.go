package oteltracingredis

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Client wraps redis.Client with tracing
type Client struct {
	client *redis.Client
	tracer trace.Tracer
}

// NewClient creates a traced redis client
func NewClient(opt *redis.Options, serviceName string) *Client {
	tracerName := fmt.Sprintf("%s-redis", serviceName)
	return &Client{
		client: redis.NewClient(opt),
		tracer: otel.Tracer(tracerName),
	}
}

func (c *Client) Client() *redis.Client {
	return c.client
}

func (c *Client) Ping(ctx context.Context) *redis.StatusCmd {
	ctx, span := c.tracer.Start(ctx, "Redis PING")
	defer span.End()

	cmd := c.client.Ping(ctx)
	recordErr(span, cmd.Err())
	return cmd
}

func (c *Client) Close() error {
	return c.client.Close()
}

func (c *Client) GetFromCache(ctx context.Context, key string, target any) error {
	ctx, span := c.tracer.Start(ctx, "Redis GET")
	defer span.End()

	err := c.client.Get(ctx, key).Scan(target)
	recordErr(span, err)
	return err
}

func (c *Client) SetToCache(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	ctx, span := c.tracer.Start(ctx, "Redis SET")
	defer span.End()

	err := c.client.Set(ctx, key, value, expiration).Err()
	recordErr(span, err)
	return err
}

func (c *Client) DeleteFromCache(ctx context.Context, key string) error {
	ctx, span := c.tracer.Start(ctx, "Redis DEL")
	defer span.End()

	err := c.client.Del(ctx, key).Err()
	recordErr(span, err)
	return err
}

// recordErr helper
func recordErr(span trace.Span, err error) {
	if err != nil && err != redis.Nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
}
