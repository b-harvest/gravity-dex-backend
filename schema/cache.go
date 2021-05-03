package schema

import "time"

type AccountsCache struct {
	BlockHeight int64                  `json:"blockHeight"`
	Accounts    []AccountsCacheAccount `json:"accounts"`
	UpdatedAt   time.Time              `json:"updatedAt"`
}

type AccountsCacheAccount struct {
	Address       string                   `json:"address"`
	Username      string                   `json:"username"`
	Ranking       int                      `json:"ranking"`
	TotalScore    float64                  `json:"totalScore"`
	TradingScore  float64                  `json:"tradingScore"`
	ActionScore   float64                  `json:"actionScore"`
	IsValid       bool                     `json:"isValid"`
	DepositStatus AccountCacheActionStatus `json:"depositStatus"`
	SwapStatus    AccountCacheActionStatus `json:"swapStatus"`
}

type AccountCacheActionStatus struct {
	NumDifferentPools       int            `json:"numDifferentPools"`
	NumDifferentPoolsByDate map[string]int `json:"numDifferentPoolsByDate"`
}

type PoolsCache struct {
	BlockHeight int64            `json:"blockHeight"`
	Pools       []PoolsCachePool `json:"pools"`
	UpdatedAt   time.Time        `json:"updatedAt"`
}

type PoolsCachePool struct {
	ID           uint64           `json:"id"`
	ReserveCoins []PoolsCacheCoin `json:"reserveCoins"`
	PoolCoin     PoolsCacheCoin   `json:"poolCoin"`
	APY          float64          `json:"apy"`
}

type PoolsCacheCoin struct {
	Denom       string  `json:"denom"`
	Amount      int64   `json:"amount"`
	GlobalPrice float64 `json:"globalPrice"`
}

type PricesCache struct {
	BlockHeight int64              `json:"blockHeight"`
	Prices      map[string]float64 `json:"prices"`
	UpdatedAt   time.Time          `json:"updatedAt"`
}
