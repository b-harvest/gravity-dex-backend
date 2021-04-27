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

var json = jsoniter.ConfigCompatibleWithStandardLibrary

func (s *Server) UpdateScoreBoardCache(ctx context.Context, blockHeight int64, priceTable price.Table) error {
	resp := schema.ScoreBoardResponse{
		BlockHeight: blockHeight,
		Accounts:    []schema.ScoreBoardAccount{},
	}
	if err := s.ss.IterateAccounts(ctx, blockHeight, func(acc schema.Account) (stop bool, err error) {
		ts, err := s.tradingScore(acc, priceTable)
		if err != nil {
			return true, fmt.Errorf("calculate trading score for account %q: %w", acc.Address, err)
		}
		as, valid := s.actionScore(acc)
		resp.Accounts = append(resp.Accounts, schema.ScoreBoardAccount{
			Username:     acc.Username,
			Address:      acc.Address,
			TotalScore:   ts*s.cfg.TradingScoreRatio + as*(1-s.cfg.TradingScoreRatio),
			TradingScore: ts,
			ActionScore:  as,
			IsValid:      valid,
		})
		return false, nil
	}); err != nil {
		return err
	}
	sort.SliceStable(resp.Accounts, func(i, j int) bool {
		if resp.Accounts[i].IsValid != resp.Accounts[j].IsValid {
			return resp.Accounts[i].IsValid
		}
		return resp.Accounts[i].TotalScore > resp.Accounts[j].TotalScore
	})
	for i := range resp.Accounts {
		resp.Accounts[i].Ranking = i + 1
	}
	resp.UpdatedAt = time.Now()
	if err := s.SaveScoreBoardCache(ctx, resp); err != nil {
		return fmt.Errorf("save cache: %w", err)
	}
	return nil
}

func (s *Server) UpdatePoolsCache(ctx context.Context, blockHeight int64, pools []schema.Pool, priceTable price.Table) error {
	resp := schema.PoolsResponse{
		BlockHeight: blockHeight,
		Pools:       []schema.PoolsResponsePool{},
	}
	for _, p := range pools {
		var reserveCoins []schema.PoolsResponseReserveCoin
		for _, rc := range p.ReserveCoins {
			reserveCoins = append(reserveCoins, schema.PoolsResponseReserveCoin{
				Denom:       rc.Denom,
				Amount:      rc.Amount,
				GlobalPrice: priceTable[rc.Denom],
			})
		}
		resp.Pools = append(resp.Pools, schema.PoolsResponsePool{
			ID:           p.ID,
			ReserveCoins: reserveCoins,
		})
	}
	resp.UpdatedAt = time.Now()
	if err := s.SavePoolsCache(ctx, resp); err != nil {
		return fmt.Errorf("save cache: %w", err)
	}
	return nil
}

func (s *Server) UpdatePricesCache(ctx context.Context, blockHeight int64, priceTable price.Table) error {
	resp := schema.PricesResponse{
		BlockHeight: blockHeight,
		Prices:      priceTable,
		UpdatedAt:   time.Now(),
	}
	if err := s.SavePricesCache(ctx, resp); err != nil {
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
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("marshal response: %w", err)
	}
	_, err = c.Do("SET", key, b)
	return err
}

func (s *Server) LoadCache(ctx context.Context, key string) ([]byte, error) {
	c, err := s.rp.GetContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("get redis conn: %w", err)
	}
	defer c.Close()
	return redis.Bytes(c.Do("GET", key))
}

func (s *Server) SaveScoreBoardCache(ctx context.Context, resp schema.ScoreBoardResponse) error {
	return s.SaveCache(ctx, s.cfg.Redis.ScoreBoardCacheKey, resp)
}

func (s *Server) SavePoolsCache(ctx context.Context, resp schema.PoolsResponse) error {
	return s.SaveCache(ctx, s.cfg.Redis.PoolsCacheKey, resp)
}

func (s *Server) SavePricesCache(ctx context.Context, resp schema.PricesResponse) error {
	return s.SaveCache(ctx, s.cfg.Redis.PricesCacheKey, resp)
}

func (s *Server) LoadScoreBoardCache(ctx context.Context) (resp schema.ScoreBoardResponse, err error) {
	b, err := s.LoadCache(ctx, s.cfg.Redis.ScoreBoardCacheKey)
	if err != nil {
		return resp, err
	}
	err = json.Unmarshal(b, &resp)
	if err != nil {
		return resp, fmt.Errorf("unmarshal response: %w", err)
	}
	return
}

func (s *Server) LoadPoolsCache(ctx context.Context) (resp schema.PoolsResponse, err error) {
	b, err := s.LoadCache(ctx, s.cfg.Redis.PoolsCacheKey)
	if err != nil {
		return resp, err
	}
	err = json.Unmarshal(b, &resp)
	if err != nil {
		return resp, fmt.Errorf("unmarshal response: %w", err)
	}
	return
}

func (s *Server) LoadPricesCache(ctx context.Context) (resp schema.PricesResponse, err error) {
	b, err := s.LoadCache(ctx, s.cfg.Redis.PricesCacheKey)
	if err != nil {
		return resp, err
	}
	err = json.Unmarshal(b, &resp)
	if err != nil {
		return resp, fmt.Errorf("unmarshal response: %w", err)
	}
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
