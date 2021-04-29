package price

import (
	"context"
	"fmt"
	"strings"

	"github.com/b-harvest/gravity-dex-backend/config"
)

type Table map[string]float64

type Service interface {
	Prices(ctx context.Context, symbols ...string) (Table, error)
}

type service struct {
	cmc Service
	cn  Service
}

func NewService(cfg config.ServerConfig) (Service, error) {
	cmc, err := NewCoinMarketCapService(cfg.CoinMarketCap.APIKey, cfg.CoinMarketCap.UpdateInterval)
	if err != nil {
		return nil, fmt.Errorf("new coinmarketcap service: %w", err)
	}
	return &service{
		cmc,
		NewCyberNodeService(cfg.CyberNode.UpdateInterval),
	}, nil
}

func (s *service) Prices(ctx context.Context, symbols ...string) (Table, error) {
	hasGcyb := false
	for i, symbol := range symbols {
		if strings.ToLower(symbol) == "gcyb" {
			hasGcyb = true
			symbols = append(symbols[:i], symbols[i+1:]...)
			break
		}
	}
	res := make(Table)
	if hasGcyb {
		t, err := s.cn.Prices(ctx, "gcyb")
		if err != nil {
			return nil, err
		}
		for k, v := range t {
			res[k] = v
		}
	}
	t, err := s.cmc.Prices(ctx, symbols...)
	if err != nil {
		return nil, err
	}
	for k, v := range t {
		res[k] = v
	}
	return res, nil
}
