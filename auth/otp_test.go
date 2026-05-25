package auth

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
)

type memoryOTPCache struct {
	values map[string]string
}

func newMemoryOTPCache() *memoryOTPCache {
	return &memoryOTPCache{values: map[string]string{}}
}

func (c *memoryOTPCache) Get(_ context.Context, key string) *redis.StringCmd {
	value, ok := c.values[key]
	if !ok {
		return redis.NewStringResult("", redis.Nil)
	}
	return redis.NewStringResult(value, nil)
}

func (c *memoryOTPCache) Set(_ context.Context, key string, value interface{}, _ time.Duration) *redis.StatusCmd {
	c.values[key] = fmt.Sprint(value)
	return redis.NewStatusResult("OK", nil)
}

func (c *memoryOTPCache) Del(_ context.Context, keys ...string) *redis.IntCmd {
	var removed int64
	for _, key := range keys {
		if _, ok := c.values[key]; ok {
			delete(c.values, key)
			removed++
		}
	}
	return redis.NewIntResult(removed, nil)
}

func TestCheckCodeDoesNotConsumeOTPBeforeMutationSucceeds(t *testing.T) {
	ctx := context.Background()
	cache := newMemoryOTPCache()
	email := "user@example.com"

	setCode(ctx, cache, email, "123456")
	if !checkCode(ctx, cache, email, "123456") {
		t.Fatalf("expected first otp check to pass")
	}
	if !checkCode(ctx, cache, email, "123456") {
		t.Fatalf("expected otp to remain available until caller consumes it")
	}

	deleteCode(ctx, cache, email)
	if checkCode(ctx, cache, email, "123456") {
		t.Fatalf("expected consumed otp to be rejected")
	}
}
