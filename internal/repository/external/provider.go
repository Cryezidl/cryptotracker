package external

import (
	"context"
)

type Provider interface {
	GetName() string
	GetCurrency(ctx context.Context, currency, quote string) (float64, error)
}
