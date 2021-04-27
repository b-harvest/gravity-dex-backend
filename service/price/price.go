package price

import (
	"context"
	"time"
)

type Table map[string]float64

type Service interface {
	Prices(ctx context.Context, symbols ...string) (Table, error)
}

type cache struct {
	price     float64
	updatedAt time.Time
}
