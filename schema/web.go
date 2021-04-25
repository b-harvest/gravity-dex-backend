package schema

import "time"

type ScoreBoardResponse struct {
	BlockHeight int64               `json:"blockHeight"`
	Accounts    []ScoreBoardAccount `json:"accounts"`
	UpdatedAt   time.Time           `json:"updatedAt"`
}

type ScoreBoardAccount struct {
	Username     string  `json:"username"`
	Address      string  `json:"address"`
	TotalScore   float64 `json:"totalScore"`
	TradingScore float64 `json:"tradingScore"`
	ActionScore  float64 `json:"actionScore"`
}

type PriceTableResponse struct {
	BlockHeight int64            `json:"blockHeight"`
	Pools       []PriceTablePool `json:"pools"`
	UpdatedAt   time.Time        `json:"updatedAt"`
}

type PriceTablePool struct {
	ID           uint64                  `json:"id"`
	ReserveCoins []PriceTableReserveCoin `json:"reserveCoins"`
}

type PriceTableReserveCoin struct {
	Denom       string  `json:"denom"`
	Amount      int64   `json:"amount"`
	GlobalPrice float64 `json:"globalPrice"`
}
