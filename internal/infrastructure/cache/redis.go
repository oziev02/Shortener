package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisCache представляет кэш на основе Redis
type RedisCache struct {
	client *redis.Client
	ttl    time.Duration
}

// NewRedisCache создаёт новый экземпляр Redis кэша
func NewRedisCache(addr string, password string, db int, ttl time.Duration) (*RedisCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &RedisCache{
		client: client,
		ttl:    ttl,
	}, nil
}

// Get получает значение из кэша
func (r *RedisCache) Get(ctx context.Context, key string, dest interface{}) error {
	val, err := r.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return ErrCacheMiss
	}
	if err != nil {
		return fmt.Errorf("failed to get from cache: %w", err)
	}

	if err := json.Unmarshal([]byte(val), dest); err != nil {
		return fmt.Errorf("failed to unmarshal cache value: %w", err)
	}

	return nil
}

// Set сохраняет значение в кэш
func (r *RedisCache) Set(ctx context.Context, key string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal cache value: %w", err)
	}

	if err := r.client.Set(ctx, key, data, r.ttl).Err(); err != nil {
		return fmt.Errorf("failed to set cache: %w", err)
	}

	return nil
}

// Delete удаляет значение из кэша
func (r *RedisCache) Delete(ctx context.Context, key string) error {
	if err := r.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete from cache: %w", err)
	}
	return nil
}

// Close закрывает подключение к Redis
func (r *RedisCache) Close() error {
	return r.client.Close()
}

// ErrCacheMiss ошибка отсутствия значения в кэше
var ErrCacheMiss = fmt.Errorf("cache miss")
