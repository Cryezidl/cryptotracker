package external

import (
	"context"
	"cryptotracker/internal/model"
)

type Provider interface {
	GetName() string
	GetCurrency(ctx context.Context, currency, quote string) (*model.Rate, error)
}
