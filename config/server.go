package config

import (
	"fmt"

	"go.uber.org/zap"
)

var DefaultServerConfig = ServerConfig{
	BindAddr:         "0.0.0.0:8080",
	StableCoinDenoms: []string{"run"},
	StakingCoinDenoms: []string{
		"atom",
		"xrn",
		"btsg",
		"dvpn",
		"xprt",
		"akt",
		"luna",
		"ngm",
		"gcyb",
		"iris",
	},
	DenomMetadata: map[string]DenomMetadata{
		"uatom": {"atom", 6},
		"uxrn":  {"xrn", 6},
		"ubtsg": {"btsg", 6},
		"udvpn": {"dvpn", 6},
		"uxprt": {"xprt", 6},
		"uakt":  {"akt", 6},
		"uluna": {"luna", 6},
		"ungm":  {"ngm", 6},
		"ugcyb": {"gcyb", 6},
		"uiris": {"iris", 6},
		"xrun":  {"run", 6},
	},
	TradingDates: []string{
		"2021-05-04",
		"2021-05-05",
		"2021-05-06",
		"2021-05-07",
		"2021-05-08",
		"2021-05-09",
		"2021-05-10",
	},
	MaxActionScorePerDay: 3,
	InitialBalancesValue: 30000,
	TradingScoreRatio:    0.9,
	ScoreBoardSize:       100,
	MongoDB:              DefaultMongoDBConfig,
	Log:                  zap.NewProductionConfig(),
}

type ServerConfig struct {
	BindAddr             string                   `yaml:"bind_addr"`
	StableCoinDenoms     []string                 `yaml:"stable_coin_denoms"`
	StakingCoinDenoms    []string                 `yaml:"staking_coin_denoms"`
	DenomMetadata        map[string]DenomMetadata `yaml:"denom_metadata"`
	CoinMarketCapAPIKey  string                   `yaml:"cmc_api_key"`
	TradingDates         []string                 `yaml:"trading_dates"`
	MaxActionScorePerDay int                      `yaml:"max_trading_score_per_day"`
	InitialBalancesValue float64                  `yaml:"initial_balances_value"`
	TradingScoreRatio    float64                  `yaml:"trading_score_ratio"`
	ScoreBoardSize       int                      `yaml:"score_board_size"`
	MongoDB              MongoDBConfig            `yaml:"mongodb"`
	Log                  zap.Config               `yaml:"log"`
}

func (cfg ServerConfig) Validate() error {
	if len(cfg.StableCoinDenoms) == 0 {
		return fmt.Errorf("'stable_coin_denoms' is empty")
	}
	if len(cfg.StakingCoinDenoms) == 0 {
		return fmt.Errorf("'staking_coin_denoms' is empty")
	}
	if len(cfg.DenomMetadata) == 0 {
		return fmt.Errorf("'denom_metadata' is empty")
	}
	if cfg.CoinMarketCapAPIKey == "" {
		return fmt.Errorf("'cmc_api_key' is required")
	}
	if len(cfg.TradingDates) == 0 {
		return fmt.Errorf("'trading_dates' is empty")
	}
	if cfg.InitialBalancesValue <= 0 {
		return fmt.Errorf("'initial_balances_value' must be positive")
	}
	if cfg.TradingScoreRatio < 0 || cfg.TradingScoreRatio > 1 {
		return fmt.Errorf("'trading_score_ratio' must be between 0~1")
	}
	return nil
}

func (cfg ServerConfig) AvailableDenoms() []string {
	denoms := append(cfg.StableCoinDenoms, cfg.StakingCoinDenoms...)
	for denom := range cfg.DenomMetadata {
		denoms = append(denoms, denom)
	}
	return denoms
}

type DenomMetadata struct {
	Display  string `yaml:"display"`
	Exponent int    `yaml:"exponent"`
}
