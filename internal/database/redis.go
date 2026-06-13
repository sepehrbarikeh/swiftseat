package database

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "time"
    "github.com/redis/go-redis/v9"
)

type RedisClient struct {
    Client *redis.Client
}

func InitRedis() *RedisClient {
    rdb := redis.NewClient(&redis.Options{
        Addr:     "localhost:6379",
        Password: "MySuperSecureRedisPassword", // bitnami password
        DB:       0,
    })

    // پینگ برای اطمینان از سلامت کانکشن
    ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
    defer cancel()

    if _, err := rdb.Ping(ctx).Result(); err != nil {
        log.Fatalf("❌ Redis Docker connection error : %v", err)
    }

    fmt.Println("🚀 Redis (Bitnami) connected")
    return &RedisClient{Client: rdb}
}

// SetCache دیتای ساختاریافته رو به جی‌سان تبدیل می‌کنه و می‌فرسته روی رم
func (r *RedisClient) SetCache(ctx context.Context, key string, data interface{}, ttl time.Duration) error {
    jsonData, err := json.Marshal(data)
    if err != nil {
        return err
    }
    return r.Client.Set(ctx, key, jsonData, ttl).Err()
}

// GetCache دیتای متنی رو می‌گیره و تبدیلش می‌کنه به استراکت گو
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