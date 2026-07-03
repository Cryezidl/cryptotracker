package cache

import (
	"context"
	"cryptotracker/internal/model"
	"time"
)

type CacheRepository interface {
	SetCrypto(ctx context.Context, currency, quote string, rate *model.Rate, ttl time.Duration) error
	GetCrypto(ctx context.Context, currency, quote string) (*model.Rate, error)
}
