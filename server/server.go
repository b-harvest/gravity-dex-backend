package server

import (
	"fmt"
	"net/http"
	"sort"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/b-harvest/gravity-dex-backend/config"
	"github.com/b-harvest/gravity-dex-backend/schema"
	"github.com/b-harvest/gravity-dex-backend/server/service/price"
	"github.com/b-harvest/gravity-dex-backend/server/service/pricetable"
	"github.com/b-harvest/gravity-dex-backend/server/service/store"
)

type Server struct {
	*echo.Echo
	cfg config.ServerConfig
	ss  *store.Service
	ps  price.Service
	pts *pricetable.Service
}

func New(cfg config.ServerConfig, ss *store.Service, ps price.Service, pts *pricetable.Service) *Server {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	// e.Debug = false
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	s := &Server{e, cfg, ss, ps, pts}
	s.registerRoutes()
	return s
}

func (s *Server) registerRoutes() {
	s.GET("/scoreboard", s.GetScoreBoard)
	s.GET("/prices", s.GetPrices)
}

func (s *Server) GetScoreBoard(c echo.Context) error {
	ctx := c.Request().Context()
	blockHeight, err := s.ss.LatestBlockHeight(ctx)
	if err != nil {
		return fmt.Errorf("get latest block height: %w", err)
	}
	ps, err := s.ss.Pools(ctx, blockHeight)
	if err != nil {
		return fmt.Errorf("get pools: %w", err)
	}
	t, err := s.pts.PriceTable(ctx, ps)
	if err != nil {
		return fmt.Errorf("get price table: %w", err)
	}
	type User struct {
		Username     string  `json:"username"`
		Address      string  `json:"address"`
		TotalScore   float64 `json:"totalScore"`
		TradingScore float64 `json:"tradingScore"`
		ActionScore  float64 `json:"actionScore"`
	}
	resp := struct {
		Users []User `json:"users"`
	}{
		Users: []User{},
	}
	if err := s.ss.IterateAccounts(ctx, blockHeight, func(acc schema.Account) (stop bool, err error) {
		ts, err := s.tradingScore(acc, t)
		if err != nil {
			return true, fmt.Errorf("calculate trading score for account %q: %w", acc.Address, err)
		}
		as := s.actionScore(acc)
		resp.Users = append(resp.Users, User{
			Username:     acc.Username,
			Address:      acc.Address,
			TotalScore:   ts*s.cfg.TradingScoreRatio + as*(1-s.cfg.TradingScoreRatio),
			TradingScore: ts,
			ActionScore:  as,
		})
		return false, nil
	}); err != nil {
		return err
	}
	sort.Slice(resp.Users, func(i, j int) bool { return resp.Users[i].TotalScore > resp.Users[j].TotalScore })
	resp.Users = resp.Users[:minInt(s.cfg.ScoreBoardSize, len(resp.Users))]
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
		score += float64(minInt(s.cfg.MaxActionScorePerDay, len(acc.Actions[k].Deposits)))
		score += float64(minInt(s.cfg.MaxActionScorePerDay, len(acc.Actions[k].Swaps)))
	}
	score /= float64((2 * s.cfg.MaxActionScorePerDay) * len(s.cfg.TradingDates))
	return score * 100
}

func (s *Server) GetPrices(c echo.Context) error {
	ctx := c.Request().Context()
	blockHeight, err := s.ss.LatestBlockHeight(ctx)
	if err != nil {
		return fmt.Errorf("get latest block height: %w", err)
	}
	ps, err := s.ss.Pools(ctx, blockHeight)
	if err != nil {
		return fmt.Errorf("get pools: %w", err)
	}
	t, err := s.pts.PriceTable(ctx, ps)
	if err != nil {
		return fmt.Errorf("get price table: %w", err)
	}
	type ReserveCoin struct {
		Denom       string  `json:"denom"`
		Amount      int64   `json:"amount"`
		GlobalPrice float64 `json:"globalPrice"`
	}
	type Pool struct {
		ID           uint64        `json:"id"`
		ReserveCoins []ReserveCoin `json:"reserveCoins"`
	}
	pools := []Pool{}
	for _, p := range ps {
		var reserveCoins []ReserveCoin
		for _, rc := range p.ReserveCoins {
			reserveCoins = append(reserveCoins, ReserveCoin{
				rc.Denom, rc.Amount, t[rc.Denom],
			})
		}
		pools = append(pools, Pool{
			ID:           p.ID,
			ReserveCoins: reserveCoins,
		})
	}
	return c.JSON(http.StatusOK, struct {
		Pools []Pool `json:"pools"`
	}{
		pools,
	})
}
