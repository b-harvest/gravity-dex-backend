package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"

	"github.com/b-harvest/gravity-dex-backend/config"
	"github.com/b-harvest/gravity-dex-backend/schema"
	"github.com/b-harvest/gravity-dex-backend/service/price"
	"github.com/b-harvest/gravity-dex-backend/service/pricetable"
	"github.com/b-harvest/gravity-dex-backend/service/store"
	"github.com/b-harvest/gravity-dex-backend/util"
)

type Server struct {
	*echo.Echo
	cfg    config.ServerConfig
	ss     *store.Service
	ps     price.Service
	pts    *pricetable.Service
	rp     *redis.Pool
	logger *zap.Logger
}

func New(cfg config.ServerConfig, ss *store.Service, ps price.Service, pts *pricetable.Service, rp *redis.Pool, logger *zap.Logger) *Server {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Debug = cfg.Debug
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())
	s := &Server{e, cfg, ss, ps, pts, rp, logger}
	s.registerRoutes()
	return s
}

func (s *Server) registerRoutes() {
	s.GET("/status", s.GetStatus)
	s.GET("/scoreboard", s.GetScoreBoard)
	s.GET("/pools", s.GetPools)
	s.GET("/prices", s.GetPrices)
}

func (s *Server) GetStatus(c echo.Context) error {
	blockHeight, err := s.ss.LatestBlockHeight(c.Request().Context())
	if err != nil {
		return fmt.Errorf("get latest block height: %w", err)
	}
	return c.JSON(http.StatusOK, schema.StatusResponse{
		LatestBlockHeight: blockHeight,
	})
}

func (s *Server) GetScoreBoard(c echo.Context) error {
	var req schema.ScoreBoardRequest
	if err := c.Bind(&req); err != nil {
		return err
	}
	var resp schema.ScoreBoardResponse
	if err := RetryLoadingCache(c.Request().Context(), func(ctx context.Context) error {
		var err error
		resp, err = s.LoadScoreBoardCache(ctx)
		return err
	}, s.cfg.CacheLoadTimeout); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return echo.NewHTTPError(http.StatusInternalServerError, "no score board data found")
		}
		return fmt.Errorf("load cache: %w", err)
	}
	var me *schema.ScoreBoardAccount
	if req.Address != "" {
		for _, acc := range resp.Accounts {
			if acc.Address == req.Address {
				me = &acc
				break
			}
		}
		if me == nil {
			return echo.NewHTTPError(http.StatusNotFound, "account not found")
		}
		resp.Me = me
	}
	resp.Accounts = resp.Accounts[:util.MinInt(s.cfg.ScoreBoardSize, len(resp.Accounts))]
	return c.JSON(http.StatusOK, resp)
}

func (s *Server) tradingScore(acc schema.Account, priceTable price.Table) (float64, error) {
	denoms := make(map[string]struct{})
	for _, d := range s.cfg.AvailableDenoms() {
		denoms[d] = struct{}{}
	}
	v := 0.0 // total usd value of the user's balances
	for _, c := range acc.Coins {
		if _, ok := denoms[c.Denom]; !ok {
			continue
		}
		p, ok := priceTable[c.Denom]
		if !ok {
			return 0, fmt.Errorf("no price for denom %q", c.Denom)
		}
		v += p * float64(c.Amount)
	}
	return (v - s.cfg.InitialBalancesValue) / s.cfg.InitialBalancesValue * 100, nil
}

func (s *Server) actionScore(acc schema.Account) (score float64, valid bool) {
	ds := make(map[uint64]struct{}) // pool id set which user deposited
	ss := make(map[uint64]struct{}) // pool id set which user swapped
	for _, k := range s.cfg.TradingDates {
		for _, da := range acc.Actions[k].Deposits {
			ds[da.PoolID] = struct{}{}
		}
		for _, sa := range acc.Actions[k].Swaps {
			ss[sa.PoolID] = struct{}{}
		}
		score += float64(util.MinInt(s.cfg.MaxActionScorePerDay, len(acc.Actions[k].Deposits)))
		score += float64(util.MinInt(s.cfg.MaxActionScorePerDay, len(acc.Actions[k].Swaps)))
	}
	score /= float64((2 * s.cfg.MaxActionScorePerDay) * len(s.cfg.TradingDates))
	score *= 100
	valid = len(ds) >= 3 && len(ss) >= 3
	return
}

func (s *Server) GetPools(c echo.Context) error {
	var resp schema.PoolsResponse
	if err := RetryLoadingCache(c.Request().Context(), func(ctx context.Context) error {
		var err error
		resp, err = s.LoadPoolsCache(ctx)
		return err
	}, s.cfg.CacheLoadTimeout); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return echo.NewHTTPError(http.StatusInternalServerError, "no pool data found")
		}
		return fmt.Errorf("load cache: %w", err)
	}
	return c.JSON(http.StatusOK, resp)
}

func (s *Server) GetPrices(c echo.Context) error {
	var resp schema.PricesResponse
	if err := RetryLoadingCache(c.Request().Context(), func(ctx context.Context) error {
		var err error
		resp, err = s.LoadPricesCache(ctx)
		return err
	}, s.cfg.CacheLoadTimeout); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return echo.NewHTTPError(http.StatusInternalServerError, "no price data found")
		}
		return fmt.Errorf("load cache: %w", err)
	}
	return c.JSON(http.StatusOK, resp)
}

func (s *Server) ShutdownWithTimeout(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return s.Shutdown(ctx)
}
