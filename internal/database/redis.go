package database

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"swift-seat/internal/config"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	Client *redis.Client
	config *config.Config
}

func InitRedis(cfg *config.Config) *RedisClient {
	addr := fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort)
	pwd := cfg.RedisPassword
	db := cfg.RedisDB

	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: pwd,
		DB:       db,
	})


	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	if _, err := rdb.Ping(ctx).Result(); err != nil {
		log.Fatalf("❌ Redis connection error : %v", err)
	}

	fmt.Printf("🚀 Redis connected (%s)\n", addr)
	return &RedisClient{Client: rdb}
}

func (r *RedisClient) SetCache(ctx context.Context, key string, data interface{}, ttl time.Duration) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return r.Client.Set(ctx, key, jsonData, ttl).Err()
}


func (r *RedisClient) GetCache(ctx context.Context, key string, dest interface{}) (bool, error) {
	val, err := r.Client.Get(ctx, key).Result()
	if err == redis.Nil {
		return false, nil // Cache Miss
	} else if err != nil {
		return false, err //
	}

	return true, json.Unmarshal([]byte(val), dest) // Cache Hit
}

// DeleteCache
func (r *RedisClient) DeleteCache(ctx context.Context, keys ...string) error {
	return r.Client.Del(ctx, keys...).Err()
}
