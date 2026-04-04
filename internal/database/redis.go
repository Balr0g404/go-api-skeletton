package database

import (
	"context"
	"fmt"
	"time"

	"github.com/Balr0g404/go-api-skeletton/internal/config"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
)

func NewRedis(cfg *config.RedisConfig) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	const maxRetries = 5
	for i := range maxRetries {
		if err := client.Ping(context.Background()).Err(); err != nil {
			if i == maxRetries-1 {
				return nil, fmt.Errorf("failed to connect to redis after %d attempts: %w", maxRetries, err)
			}
			log.Warn().Err(err).Msgf("redis not ready, retrying in 2s... (%d/%d)", i+1, maxRetries)
			time.Sleep(2 * time.Second)
			continue
		}
		break
	}

	return client, nil
}
