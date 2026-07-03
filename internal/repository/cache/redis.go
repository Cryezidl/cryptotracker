package cache

import (
	"context"
	"cryptotracker/pkg/myerrors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisCache struct {
	rdb *redis.Client
}

func NewReddisCache(addr, psw string, db int) *RedisCache {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: psw,
		DB:       db,
	})
	return &RedisCache{rdb: rdb}
}

func (c *RedisCache) SetCrypto(ctx context.Context, currency, quote string, price float64, ttl time.Duration) error {
	key := "crypto:pair:" + currency + ":" + quote
	if err := c.rdb.Set(ctx, key, price, ttl).Err(); err != nil {
		return err
	}
	return nil
}

func (c *RedisCache) GetCrypto(ctx context.Context, currency, quote string) (float64, error) {
	key := "crypto:pair:" + currency + ":" + quote
	val, err := c.rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return -1, myerrors.KeyNotFoundInCacheError
	} else if err != nil {
		return -1, myerrors.ReadingCacheError
	}

	valConverted, err := strconv.ParseFloat(val, 64)
	if err != nil {
		return -1, fmt.Errorf("failed to parse price '%s': %w", val, err)
	}

	return valConverted, nil
}
