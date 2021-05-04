package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/b-harvest/gravity-dex-backend/schema"
)

func (s *Server) registerRoutes() {
	s.GET("/status", s.GetStatus)
	s.GET("/scoreboard", s.GetScoreBoard)
	s.GET("/scoreboard/search", s.SearchAccount)
	s.GET("/actions", s.GetActionStatus)
	s.GET("/pools", s.GetPools)
	s.GET("/prices", s.GetPrices)
	s.GET("/banner", s.GetBanner)
}

func (s *Server) GetStatus(c echo.Context) error {
	blockHeight, err := s.ss.LatestBlockHeight(c.Request().Context())
	if err != nil {
		return fmt.Errorf("get latest block height: %w", err)
	}
	return c.JSON(http.StatusOK, schema.GetStatusResponse{
		LatestBlockHeight: blockHeight,
	})
}

func (s *Server) GetScoreBoard(c echo.Context) error {
	var req schema.GetScoreBoardRequest
	if err := c.Bind(&req); err != nil {
		return err
	}
	var cache schema.AccountsCache
	if err := RetryLoadingCache(c.Request().Context(), func(ctx context.Context) error {
		var err error
		cache, err = s.LoadAccountsCache(ctx)
		return err
	}, s.cfg.CacheLoadTimeout); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return echo.NewHTTPError(http.StatusInternalServerError, "no account data found")
		}
		return fmt.Errorf("load accounts cache: %w", err)
	}
	resp := schema.GetScoreBoardResponse{
		BlockHeight: cache.BlockHeight,
		Accounts:    []schema.GetScoreBoardResponseAccount{},
		UpdatedAt:   cache.UpdatedAt,
	}
	for i, acc := range cache.Accounts {
		if req.Address != "" && acc.Address == req.Address {
			resp.Me = &schema.GetScoreBoardResponseAccount{
				Ranking:      acc.Ranking,
				Username:     acc.Username,
				Address:      acc.Address,
				TotalScore:   acc.TotalScore,
				TradingScore: acc.TradingScore,
				ActionScore:  acc.ActionScore,
				IsValid:      acc.IsValid,
			}
		}
		if i < s.cfg.ScoreBoardSize {
			resp.Accounts = append(resp.Accounts, schema.GetScoreBoardResponseAccount{
				Ranking:      acc.Ranking,
				Username:     acc.Username,
				Address:      acc.Address,
				TotalScore:   acc.TotalScore,
				TradingScore: acc.TradingScore,
				ActionScore:  acc.ActionScore,
				IsValid:      acc.IsValid,
			})
		}
	}
	return c.JSON(http.StatusOK, resp)
}

func (s *Server) SearchAccount(c echo.Context) error {
	var req schema.SearchAccountRequest
	if err := c.Bind(&req); err != nil {
		return err
	}
	if req.Query == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "query must be provided")
	}
	var cache schema.AccountsCache
	if err := RetryLoadingCache(c.Request().Context(), func(ctx context.Context) error {
		var err error
		cache, err = s.LoadAccountsCache(ctx)
		return err
	}, s.cfg.CacheLoadTimeout); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return echo.NewHTTPError(http.StatusInternalServerError, "no account data found")
		}
		return fmt.Errorf("load accounts cache: %w", err)
	}
	resp := schema.SearchAccountResponse{
		BlockHeight: cache.BlockHeight,
		UpdatedAt:   cache.UpdatedAt,
	}
	for _, acc := range cache.Accounts {
		if acc.Address == req.Query || acc.Username == req.Query {
			resp.Account = &schema.GetScoreBoardResponseAccount{
				Ranking:      acc.Ranking,
				Username:     acc.Username,
				Address:      acc.Address,
				TotalScore:   acc.TotalScore,
				TradingScore: acc.TradingScore,
				ActionScore:  acc.ActionScore,
				IsValid:      acc.IsValid,
			}
			break
		}
	}
	return c.JSON(http.StatusOK, resp)
}

func (s *Server) GetActionStatus(c echo.Context) error {
	var req schema.GetActionStatusRequest
	if err := c.Bind(&req); err != nil {
		return err
	}
	if req.Address == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "address must be provided")
	}
	var cache schema.AccountsCache
	if err := RetryLoadingCache(c.Request().Context(), func(ctx context.Context) error {
		var err error
		cache, err = s.LoadAccountsCache(ctx)
		return err
	}, s.cfg.CacheLoadTimeout); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return echo.NewHTTPError(http.StatusInternalServerError, "no account data found")
		}
		return fmt.Errorf("load accounts cache: %w", err)
	}
	resp := schema.GetActionStatusResponse{
		BlockHeight: cache.BlockHeight,
		UpdatedAt:   cache.UpdatedAt,
	}
	todayKey := time.Now().UTC().Format("2006-01-02")
	for _, acc := range cache.Accounts {
		if acc.Address == req.Address {
			resp.Account = &schema.GetActionStatusResponseAccount{
				Deposit: schema.GetActionStatusResponseStatus{
					NumDifferentPools:         acc.DepositStatus.NumDifferentPools,
					NumDifferentPoolsToday:    acc.DepositStatus.NumDifferentPoolsByDate[todayKey],
					MaxNumDifferentPoolsToday: s.cfg.MaxActionScorePerDay,
				},
				Swap: schema.GetActionStatusResponseStatus{
					NumDifferentPools:         acc.SwapStatus.NumDifferentPools,
					NumDifferentPoolsToday:    acc.SwapStatus.NumDifferentPoolsByDate[todayKey],
					MaxNumDifferentPoolsToday: s.cfg.MaxActionScorePerDay,
				},
			}
			break
		}
	}
	return c.JSON(http.StatusOK, resp)
}

func (s *Server) GetPools(c echo.Context) error {
	var cache schema.PoolsCache
	if err := RetryLoadingCache(c.Request().Context(), func(ctx context.Context) error {
		var err error
		cache, err = s.LoadPoolsCache(ctx)
		return err
	}, s.cfg.CacheLoadTimeout); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return echo.NewHTTPError(http.StatusInternalServerError, "no pool data found")
		}
		return fmt.Errorf("load pools cache: %w", err)
	}
	return c.JSON(http.StatusOK, schema.GetPoolsResponse(cache))
}

func (s *Server) GetPrices(c echo.Context) error {
	var cache schema.PricesCache
	if err := RetryLoadingCache(c.Request().Context(), func(ctx context.Context) error {
		var err error
		cache, err = s.LoadPricesCache(ctx)
		return err
	}, s.cfg.CacheLoadTimeout); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return echo.NewHTTPError(http.StatusInternalServerError, "no price data found")
		}
		return fmt.Errorf("load prices cache: %w", err)
	}
	return c.JSON(http.StatusOK, schema.GetPricesResponse(cache))
}

func (s *Server) GetBanner(c echo.Context) error {
	banner, err := s.ss.Banner(c.Request().Context(), time.Now())
	if err != nil {
		return fmt.Errorf("get banner: %w", err)
	}
	resp := schema.GetBannerResponse{}
	if banner != nil {
		var state schema.GetBannerResponseState
		if banner.StartsAt.After(time.Now()) {
			state = schema.GetBannerResponseStateUpcoming
		} else {
			state = schema.GetBannerResponseStateStarted
		}
		var text string
		switch state {
		case schema.GetBannerResponseStateUpcoming:
			text = banner.UpcomingText
		case schema.GetBannerResponseStateStarted:
			text = banner.Text
		}
		resp.Banner = &schema.GetBannerResponseBanner{
			State:    state,
			Text:     text,
			URL:      banner.URL,
			StartsAt: banner.StartsAt,
			EndsAt:   banner.EndsAt,
		}
	}
	return c.JSON(http.StatusOK, resp)
}
