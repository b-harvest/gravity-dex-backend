package schema

import "time"

type StatusResponse struct {
	LatestBlockHeight int64 `json:"latestBlockHeight"`
}

type ScoreBoardRequest struct {
	Address string `query:"address"`
}

type ScoreBoardResponse struct {
	BlockHeight int64               `json:"blockHeight"`
	Me          *ScoreBoardAccount  `json:"me,omitempty"`
	Accounts    []ScoreBoardAccount `json:"accounts"`
	UpdatedAt   time.Time           `json:"updatedAt"`
}

type ScoreBoardAccount struct {
	Ranking      int     `json:"ranking"`
	Username     string  `json:"username"`
	Address      string  `json:"address"`
	TotalScore   float64 `json:"totalScore"`
	TradingScore float64 `json:"tradingScore"`
	ActionScore  float64 `json:"actionScore"`
	IsValid      bool    `json:"isValid"`
}

type PoolsResponse struct {
	BlockHeight int64               `json:"blockHeight"`
	Pools       []PoolsResponsePool `json:"pools"`
	UpdatedAt   time.Time           `json:"updatedAt"`
}

type PoolsResponsePool struct {
	ID           uint64                     `json:"id"`
	ReserveCoins []PoolsResponseReserveCoin `json:"reserveCoins"`
}

type PoolsResponseReserveCoin struct {
	Denom       string  `json:"denom"`
	Amount      int64   `json:"amount"`
	GlobalPrice float64 `json:"globalPrice"`
}

type CoinsResponse struct {
	BlockHeight int64               `json:"blockHeight"`
	Coins       []CoinsResponseCoin `json:"coins"`
	UpdatedAt   time.Time           `json:"updatedAt"`
}

type CoinsResponseCoin struct {
	Denom       string  `json:"denom"`
	GlobalPrice float64 `json:"globalPrice"`
}
