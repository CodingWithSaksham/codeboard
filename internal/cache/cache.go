package cache

import (
	"context"
	"log"
	"time"

	"github.com/creative-computing-society/codeboard/internal/config"
	"github.com/redis/go-redis/v9"
)

var (
	Client *redis.Client
	Ctx    = context.Background()
)

func Connect() {
	opts, err := redis.ParseURL(config.C.RedisURL)
	if err != nil {
		log.Fatalf("failed to parse redis URL: %v", err)
	}

	Client = redis.NewClient(opts)

	if _, err := Client.Ping(Ctx).Result(); err != nil {
		log.Fatalf("failed to connect to redis: %v", err)
	}

	log.Println("Redis connected successfully")
}

func Set(key string, value any, ttl time.Duration) error {
	return Client.Set(Ctx, key, value, ttl).Err()
}

func Get(key string) (string, error) {
	return Client.Get(Ctx, key).Result()
}

func Del(key string) error {
	return Client.Del(Ctx, key).Err()
}
