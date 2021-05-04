package server

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/gomodule/redigo/redis"
	jsoniter "github.com/json-iterator/go"

	"github.com/b-harvest/gravity-dex-backend/schema"
	"github.com/b-harvest/gravity-dex-backend/service/price"
	"github.com/b-harvest/gravity-dex-backend/util"
)

var jsonit = jsoniter.ConfigCompatibleWithStandardLibrary

func (s *Server) UpdateAccountsCache(ctx context.Context, blockHeight int64, priceTable price.Table) error {
	accCaches := []schema.AccountCache{}
	now := time.Now()
	if err := s.ss.IterateAccounts(ctx, blockHeight, func(acc schema.Account) (stop bool, err error) {
		if acc.Username == "" {
			return false, nil
		}
		ts, err := s.tradingScore(acc, priceTable)
		if err != nil {
			return true, fmt.Errorf("calculate trading score for account %q: %w", acc.Address, err)
		}
		as, valid := s.actionScore(acc)
		accCaches = append(accCaches, schema.AccountCache{
			BlockHeight:  blockHeight,
			Address:      acc.Address,
			Username:     acc.Username,
			TotalScore:   ts*s.cfg.TradingScoreRatio + as*(1-s.cfg.TradingScoreRatio),
			TradingScore: ts,
			ActionScore:  as,
			IsValid:      valid,
			DepositStatus: schema.AccountCacheActionStatus{
				NumDifferentPools:       acc.DepositStatus().NumDifferentPools(),
				NumDifferentPoolsByDate: acc.DepositStatus().NumDifferentPoolsByDate(),
			},
			SwapStatus: schema.AccountCacheActionStatus{
				NumDifferentPools:       acc.SwapStatus().NumDifferentPools(),
				NumDifferentPoolsByDate: acc.SwapStatus().NumDifferentPoolsByDate(),
			},
			UpdatedAt: now,
		})
		return false, nil
	}); err != nil {
		return err
	}
	sort.SliceStable(accCaches, func(i, j int) bool {
		if accCaches[i].IsValid != accCaches[j].IsValid {
			return accCaches[i].IsValid
		}
		if accCaches[i].TotalScore != accCaches[j].TotalScore {
			return accCaches[i].TotalScore > accCaches[j].TotalScore
		}
		return accCaches[i].Address < accCaches[j].Address
	})
	for i := range accCaches {
		accCaches[i].Ranking = i + 1
		if err := s.SaveAccountCache(ctx, accCaches[i].Address, accCaches[i]); err != nil {
			return fmt.Errorf("save account cache: %w", err)
		}
	}
	sbCache := schema.ScoreBoardCache{
		BlockHeight: blockHeight,
		Accounts:    accCaches[:util.MinInt(s.cfg.ScoreBoardSize, len(accCaches))],
		UpdatedAt:   time.Now(),
	}
	if err := s.SaveScoreBoardCache(ctx, sbCache); err != nil {
		return fmt.Errorf("save cache: %w", err)
	}
	return nil
}

func (s *Server) UpdatePoolsCache(ctx context.Context, blockHeight int64, pools []schema.Pool, priceTable price.Table) error {
	cache := schema.PoolsCache{
		BlockHeight: blockHeight,
		Pools:       []schema.PoolsCachePool{},
	}
	for _, p := range pools {
		if p.PoolCoinAmount() == 0 {
			continue
		}
		var reserveCoins []schema.PoolsCacheCoin
		for _, rc := range p.ReserveCoins() {
			reserveCoins = append(reserveCoins, schema.PoolsCacheCoin{
				Denom:       rc.Denom,
				Amount:      rc.Amount,
				GlobalPrice: priceTable[rc.Denom],
			})
		}
		cs := p.SwapFeeVolumes().TotalCoins()
		feeValue := 0.0
		for denom, amount := range cs {
			feeValue += float64(amount) * priceTable[denom]
		}
		poolValue := priceTable[p.PoolCoinDenom] * float64(p.PoolCoinAmount())
		cache.Pools = append(cache.Pools, schema.PoolsCachePool{
			ID:           p.ID,
			ReserveCoins: reserveCoins,
			PoolCoin: schema.PoolsCacheCoin{
				Denom:       p.PoolCoinDenom,
				Amount:      p.PoolCoinAmount(),
				GlobalPrice: priceTable[p.PoolCoinDenom],
			},
			SwapFeeValueSinceLastHour: feeValue,
			APY:                       feeValue / poolValue * 24 * 365,
		})
	}
	sort.Slice(pools, func(i, j int) bool {
		return pools[i].ID < pools[j].ID
	})
	cache.UpdatedAt = time.Now()
	if err := s.SavePoolsCache(ctx, cache); err != nil {
		return fmt.Errorf("save cache: %w", err)
	}
	return nil
}

func (s *Server) UpdatePricesCache(ctx context.Context, blockHeight int64, priceTable price.Table) error {
	cache := schema.PricesCache{
		BlockHeight: blockHeight,
		Prices:      priceTable,
		UpdatedAt:   time.Now(),
	}
	if err := s.SavePricesCache(ctx, cache); err != nil {
		return fmt.Errorf("save cache: %w", err)
	}
	return nil
}

func (s *Server) SaveCache(ctx context.Context, key string, v interface{}) error {
	c, err := s.rp.GetContext(ctx)
	if err != nil {
		return fmt.Errorf("get redis conn: %w", err)
	}
	defer c.Close()
	b, err := jsonit.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal response: %w", err)
	}
	_, err = c.Do("SET", key, b)
	return err
}

func (s *Server) LoadCache(ctx context.Context, key string, v interface{}) error {
	c, err := s.rp.GetContext(ctx)
	if err != nil {
		return fmt.Errorf("get redis conn: %w", err)
	}
	defer c.Close()
	b, err := redis.Bytes(c.Do("GET", key))
	if err != nil {
		return fmt.Errorf("get cache bytes: %w", err)
	}
	if err := jsonit.Unmarshal(b, v); err != nil {
		return fmt.Errorf("unmarshal cache: %w", err)
	}
	return nil
}

func (s *Server) SaveAccountCache(ctx context.Context, address string, cache schema.AccountCache) error {
	return s.SaveCache(ctx, s.cfg.Redis.AccountCacheKeyPrefix+address, cache)
}

func (s *Server) SaveScoreBoardCache(ctx context.Context, cache schema.ScoreBoardCache) error {
	return s.SaveCache(ctx, s.cfg.Redis.ScoreBoardCacheKey, cache)
}

func (s *Server) SavePoolsCache(ctx context.Context, cache schema.PoolsCache) error {
	return s.SaveCache(ctx, s.cfg.Redis.PoolsCacheKey, cache)
}

func (s *Server) SavePricesCache(ctx context.Context, cache schema.PricesCache) error {
	return s.SaveCache(ctx, s.cfg.Redis.PricesCacheKey, cache)
}

func (s *Server) LoadAccountCache(ctx context.Context, address string) (cache schema.AccountCache, err error) {
	err = s.LoadCache(ctx, s.cfg.Redis.AccountCacheKeyPrefix+address, &cache)
	return
}

func (s *Server) LoadScoreBoardCache(ctx context.Context) (cache schema.ScoreBoardCache, err error) {
	err = s.LoadCache(ctx, s.cfg.Redis.ScoreBoardCacheKey, &cache)
	return
}

func (s *Server) LoadPoolsCache(ctx context.Context) (cache schema.PoolsCache, err error) {
	err = s.LoadCache(ctx, s.cfg.Redis.PoolsCacheKey, &cache)
	return
}

func (s *Server) LoadPricesCache(ctx context.Context) (cache schema.PricesCache, err error) {
	err = s.LoadCache(ctx, s.cfg.Redis.PricesCacheKey, &cache)
	return
}

func RetryLoadingCache(ctx context.Context, fn func(context.Context) error, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	ticker := util.NewImmediateTicker(time.Second)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := fn(ctx); err != nil {
				if !errors.Is(err, redis.ErrNil) {
					return err
				}
			} else {
				return nil
			}
		}
	}
}
