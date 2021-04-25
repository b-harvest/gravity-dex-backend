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
	"github.com/b-harvest/gravity-dex-backend/server/service/price"
	"github.com/b-harvest/gravity-dex-backend/server/service/pricetable"
	"github.com/b-harvest/gravity-dex-backend/server/service/store"
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
	s.GET("/scoreboard", s.GetScoreBoard)
	s.GET("/pricetable", s.GetPriceTable)
}

func (s *Server) GetScoreBoard(c echo.Context) error {
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

func (s *Server) actionScore(acc schema.Account) float64 {
	score := 0.0
	for _, k := range s.cfg.TradingDates {
		score += float64(util.MinInt(s.cfg.MaxActionScorePerDay, len(acc.Actions[k].Deposits)))
		score += float64(util.MinInt(s.cfg.MaxActionScorePerDay, len(acc.Actions[k].Swaps)))
	}
	score /= float64((2 * s.cfg.MaxActionScorePerDay) * len(s.cfg.TradingDates))
	return score * 100
}

func (s *Server) GetPriceTable(c echo.Context) error {
	var resp schema.PriceTableResponse
	if err := RetryLoadingCache(c.Request().Context(), func(ctx context.Context) error {
		var err error
		resp, err = s.LoadPriceTableCache(ctx)
		return err
	}, s.cfg.CacheLoadTimeout); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return echo.NewHTTPError(http.StatusInternalServerError, "no score board data found")
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
