package server

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/b-harvest/gravity-dex-backend/server/service/price"
	"github.com/b-harvest/gravity-dex-backend/server/service/pricetable"
	"github.com/b-harvest/gravity-dex-backend/server/service/store"
)

type Server struct {
	*echo.Echo
	ss  *store.Service
	ps  price.Service
	pts *pricetable.Service
}

func New(ss *store.Service, ps price.Service, pts *pricetable.Service) *Server {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	// e.Debug = false
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	s := &Server{e, ss, ps, pts}
	s.registerRoutes()
	return s
}

func (s *Server) registerRoutes() {
	s.GET("/scoreboard", s.GetScoreBoard)
	s.GET("/prices", s.GetPrices)
}

func (s *Server) GetScoreBoard(c echo.Context) error {
	return nil
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
