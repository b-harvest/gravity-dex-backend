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

type PricesResponse struct {
	BlockHeight int64                `json:"blockHeight"`
	Coins       []PricesResponseCoin `json:"coins"`
	UpdatedAt   time.Time            `json:"updatedAt"`
}

type PricesResponseCoin struct {
	Denom       string  `json:"denom"`
	GlobalPrice float64 `json:"globalPrice"`
}
