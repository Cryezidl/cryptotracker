package cache

import (
	"context"
	"cryptotracker/internal/model"
	"cryptotracker/pkg/myerrors"
	"encoding/json"
	"fmt"
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

func (c *RedisCache) SetCrypto(ctx context.Context, currency, quote string, rate *model.Rate, ttl time.Duration) error {
	jsonData, err := json.Marshal(rate)
	if err != nil {
		return fmt.Errorf("failed to marshal rate: %w", err)
	}

	key := "crypto:pair:" + currency + ":" + quote

	if err := c.rdb.Set(ctx, key, jsonData, ttl).Err(); err != nil {
		return err
	}
	return nil
}

func (c *RedisCache) GetCrypto(ctx context.Context, currency, quote string) (*model.Rate, error) {
	key := "crypto:pair:" + currency + ":" + quote
	val, err := c.rdb.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, myerrors.KeyNotFoundInCacheError
	} else if err != nil {
		return nil, myerrors.ReadingCacheError
	}

	valConverted := &model.Rate{}
	if err = json.Unmarshal(val, valConverted); err != nil {
		return nil, fmt.Errorf("failed to parse rate '%s': %w", val, err)
	}

	return valConverted, nil
}
