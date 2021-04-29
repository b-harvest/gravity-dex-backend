package price

import (
	"context"
)

type Table map[string]float64

type Service interface {
	Prices(ctx context.Context, symbols ...string) (Table, error)
}
