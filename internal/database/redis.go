package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/Balr0g404/go-api-skeletton/internal/config"
)

func NewRedis(cfg *config.RedisConfig) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	const maxRetries = 5
	for i := range maxRetries {
		if err := client.Ping(context.Background()).Err(); err != nil {
			if i == maxRetries-1 {
				log.Fatalf("failed to connect to redis: %v", err)
			}
			log.Printf("redis not ready, retrying in 2s... (%d/%d): %v", i+1, maxRetries, err)
			time.Sleep(2 * time.Second)
			continue
		}
		break
	}

	return client
}