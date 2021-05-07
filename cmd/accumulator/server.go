package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type Server struct {
	*echo.Echo
	cm *CacheManager
}

func NewServer(cm *CacheManager) *Server {
	e := echo.New()
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())
	e.HideBanner = true
	e.HidePort = true

	s := &Server{
		Echo: e,
		cm:   cm,
	}
	s.registerRoutes()
	return s
}

func (s *Server) registerRoutes() {
	s.GET("/stats", s.GetStats)
}

func (s *Server) GetStats(c echo.Context) error {
	cache, err := s.cm.Get(c.Request().Context())
	if err != nil {
		return fmt.Errorf("get cache: %w", err)
	}
	if cache == nil {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "stats not found")
	}
	var resp struct {
		BlockHeight                   int64  `json:"blockHeight"`
		NumActiveAddresses            int    `json:"numActiveAddresses"`
		NumActiveAddressesLast24Hours int    `json:"numActiveAddressesLast24Hours"`
		NumDeposits                   int    `json:"numDeposits"`
		NumSwaps                      int    `json:"numSwaps"`
		NumTransactions               int    `json:"numTransactions"`
		NumDepositsLast24Hours        int    `json:"numDepositsLast24Hours"`
		NumSwapsLast24Hours           int    `json:"numSwapsLast24Hours"`
		NumTransactionsLast24Hours    int    `json:"numTransactionsLast24Hours"`
		TransactedCoins               string `json:"transactedCoins"`
		TransactedCoinsLast24Hours    string `json:"transactedCoinsLast24Hours"`
		SwapVolume                    string `json:"swapVolume"`
		SwapVolumeLast24Hours         string `json:"swapVolumeLast24Hours`
	}
	yesterday := time.Now().AddDate(0, 0, -1)
	resp.BlockHeight = cache.BlockHeight
	resp.NumActiveAddresses = cache.Stats.NumActiveAddresses()
	resp.NumActiveAddressesLast24Hours = cache.Stats.NumActiveAddressesSince(yesterday)
	resp.NumDeposits = cache.Stats.NumDeposits()
	resp.NumSwaps = cache.Stats.NumSwaps()
	resp.NumTransactions = resp.NumDeposits + resp.NumSwaps
	resp.NumDepositsLast24Hours = cache.Stats.NumDepositsSince(yesterday)
	resp.NumSwapsLast24Hours = cache.Stats.NumSwapsSince(yesterday)
	resp.NumTransactionsLast24Hours = resp.NumDepositsLast24Hours + resp.NumSwapsLast24Hours
	cs := cache.Stats.TransactedCoins()
	resp.TransactedCoins = cs.String()
	resp.SwapVolume = cs.Div(2).String()
	cs = cache.Stats.TransactedCoinsSince(yesterday)
	resp.TransactedCoinsLast24Hours = cs.String()
	resp.SwapVolumeLast24Hours = cs.Div(2).String()
	return c.JSON(http.StatusOK, resp)
}

func (s *Server) ShutdownWithTimeout(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return s.Shutdown(ctx)
}
