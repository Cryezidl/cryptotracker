package cache

import (
	"context"
	"time"
)

type CacheRepository interface {
	SetCrypto(ctx context.Context, currency, quote string, price float64, ttl time.Duration) error
	GetCrypto(ctx context.Context, currency, quote string) (float64, error)
}
