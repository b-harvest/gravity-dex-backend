package pricetable

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/b-harvest/gravity-dex-backend/config"
	"github.com/b-harvest/gravity-dex-backend/schema"
	"github.com/b-harvest/gravity-dex-backend/server/service/price"
)

type Service struct {
	cfg config.ServerConfig
	ps  price.Service
}

func NewService(cfg config.ServerConfig, ps price.Service) *Service {
	return &Service{cfg, ps}
}

func (s *Service) PriceTable(ctx context.Context, pools []schema.Pool) (price.Table, error) {
	t, err := s.ps.Prices(ctx, s.cfg.StakingCoinDenoms...)
	if err != nil {
		return nil, fmt.Errorf("get prices: %w", err)
	}
	poolByPoolCoinDenom := make(map[string]*schema.Pool)
	for _, p := range pools {
		poolByPoolCoinDenom[p.PoolCoin.Denom] = &p
	}
	c := &Context{
		s.cfg.StableCoinDenoms,
		s.cfg.StakingCoinDenoms,
		s.cfg.DenomMetadata,
		t,
		poolByPoolCoinDenom,
	}
	denoms := append(s.cfg.StableCoinDenoms, s.cfg.StakingCoinDenoms...)
	for denom := range s.cfg.DenomMetadata {
		denoms = append(denoms, denom)
	}
	for denom := range poolByPoolCoinDenom {
		denoms = append(denoms, denom)
	}
	for _, denom := range denoms {
		if _, ok := t[denom]; !ok {
			_, err := c.Price(denom)
			if err != nil {
				return nil, fmt.Errorf("get price of denom %q: %w", denom, err)
			}
		}
	}
	return c.priceTable, nil
}

type Context struct {
	stableCoinDenoms  []string
	stakingCoinDenoms []string
	denomMetadata     map[string]config.DenomMetadata
	priceTable        price.Table
	pools             map[string]*schema.Pool
}

func (c *Context) IsStableCoinDenom(denom string) bool {
	return stringInSlice(denom, c.stableCoinDenoms)
}

func (c *Context) IsStakingCoinDenom(denom string) bool {
	return stringInSlice(denom, c.stakingCoinDenoms)
}

func (c *Context) IsPoolCoinDenom(denom string) bool {
	if !strings.HasPrefix(denom, "pool") {
		return false
	}
	_, ok := c.pools[denom]
	return ok
}

func (c *Context) Price(denom string) (float64, error) {
	p, ok := c.priceTable[denom]
	if !ok {
		switch {
		case c.IsStableCoinDenom(denom):
			p = 1
		case c.IsStakingCoinDenom(denom):
			return 0, fmt.Errorf("staking coin denom %q's price must be in price table", denom)
		case c.IsPoolCoinDenom(denom):
			pool := c.pools[denom]
			sum := 0.0
			for _, rc := range pool.ReserveCoins {
				tp, err := c.Price(rc.Denom)
				if err != nil {
					return 0, err
				}
				sum += tp * float64(rc.Amount)
			}
			p = 1 / float64(pool.PoolCoin.Amount) * sum
		default:
			md, ok := c.denomMetadata[denom]
			if !ok {
				return 0, fmt.Errorf("unknown denom type: %s", denom)
			}
			tp, err := c.Price(md.Display)
			if err != nil {
				return 0, err
			}
			p = tp / math.Pow10(md.Exponent)
		}
		c.priceTable[denom] = p
	}
	return p, nil
}
