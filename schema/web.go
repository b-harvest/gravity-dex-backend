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
	Me          *ScoreBoardAccount  `json:"me"`
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

type SearchAccountRequest struct {
	Query string `query:"q"`
}

type SearchAccountResponse struct {
	BlockHeight int64 `json:"blockHeight"`
	Account     *ScoreBoardAccount
	UpdatedAt   time.Time `json:"updatedAt"`
}

type PoolsResponse struct {
	BlockHeight int64               `json:"blockHeight"`
	Pools       []PoolsResponsePool `json:"pools"`
	UpdatedAt   time.Time           `json:"updatedAt"`
}

type PoolsResponsePool struct {
	ID           uint64                     `json:"id"`
	ReserveCoins []PoolsResponseReserveCoin `json:"reserveCoins"`
	APY          float64                    `json:"apy"`
}

type PoolsResponseReserveCoin struct {
	Denom       string  `json:"denom"`
	Amount      int64   `json:"amount"`
	GlobalPrice float64 `json:"globalPrice"`
}

type PricesResponse struct {
	BlockHeight int64              `json:"blockHeight"`
	Prices      map[string]float64 `json:"prices"`
	UpdatedAt   time.Time          `json:"updatedAt"`
}
