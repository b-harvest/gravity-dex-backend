package config

import (
	"fmt"
	"time"

	"go.uber.org/zap"
)

var DefaultServerConfig = ServerConfig{
	Debug:    false,
	BindAddr: "0.0.0.0:8080",
	CoinDenoms: []string{
		"atom",
		"regen",
		"btsg",
		"dvpn",
		"xprt",
		"akt",
		"luna",
		"ngm",
		"gcyb",
		"iris",
		"run",
	},
	ManualPrices: []ManualPrice{
		{Denom: "run", MinPrice: 1.0, MaxPrice: 1.0},
		{Denom: "regen", MinPrice: 1.0, MaxPrice: 2.0},
	},
	DenomMetadata: []DenomMetadata{
		{Denom: "uatom", Display: "atom", Exponent: 6},
		{Denom: "uregen", Display: "regen", Exponent: 6},
		{Denom: "ubtsg", Display: "btsg", Exponent: 6},
		{Denom: "udvpn", Display: "dvpn", Exponent: 6},
		{Denom: "uxprt", Display: "xprt", Exponent: 6},
		{Denom: "uakt", Display: "akt", Exponent: 6},
		{Denom: "uluna", Display: "luna", Exponent: 6},
		{Denom: "ungm", Display: "ngm", Exponent: 6},
		{Denom: "ugcyb", Display: "gcyb", Exponent: 6},
		{Denom: "uiris", Display: "iris", Exponent: 6},
		{Denom: "xrun", Display: "run", Exponent: 6},
	},
	CoinMarketCap: DefaultCoinMarketCapConfig,
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
	CacheLoadTimeout:     10 * time.Second,
	CacheUpdateInterval:  10 * time.Second,
	MongoDB:              DefaultMongoDBConfig,
	Redis:                DefaultRedisConfig,
	Log:                  zap.NewProductionConfig(),
}

type ServerConfig struct {
	Debug                bool                `yaml:"debug"`
	BindAddr             string              `yaml:"bind_addr"`
	CoinDenoms           []string            `yaml:"coin_denoms"`
	ManualPrices         []ManualPrice       `yaml:"manual_prices"`
	DenomMetadata        []DenomMetadata     `yaml:"denom_metadata"`
	CoinMarketCap        CoinMarketCapConfig `yaml:"coinmarketcap"`
	CyberNode            CyberNodeConfig     `yaml:"cybernode"`
	TradingDates         []string            `yaml:"trading_dates"`
	MaxActionScorePerDay int                 `yaml:"max_trading_score_per_day"`
	InitialBalancesValue float64             `yaml:"initial_balances_value"`
	TradingScoreRatio    float64             `yaml:"trading_score_ratio"`
	ScoreBoardSize       int                 `yaml:"score_board_size"`
	CacheLoadTimeout     time.Duration       `yaml:"cache_load_timeout"`
	CacheUpdateInterval  time.Duration       `yaml:"cache_update_interval"`
	MongoDB              MongoDBConfig       `yaml:"mongodb"`
	Redis                RedisConfig         `yaml:"redis"`
	Log                  zap.Config          `yaml:"log"`
}

func (cfg ServerConfig) Validate() error {
	if len(cfg.CoinDenoms) == 0 {
		return fmt.Errorf("'coin_denoms' is empty")
	}
	if len(cfg.DenomMetadata) == 0 {
		return fmt.Errorf("'denom_metadata' is empty")
	}
	if cfg.CoinMarketCap.APIKey == "" {
		return fmt.Errorf("'coinmarketcap.api_key' is required")
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

func (cfg ServerConfig) QueryableDenoms() []string {
	var denoms []string
	mm := cfg.ManualPricesMap()
	for _, d := range cfg.CoinDenoms {
		if _, ok := mm[d]; !ok {
			denoms = append(denoms, d)
		}
	}
	return denoms
}

func (cfg ServerConfig) AvailableDenoms() []string {
	denoms := cfg.CoinDenoms
	for _, md := range cfg.DenomMetadata {
		denoms = append(denoms, md.Denom)
	}
	return denoms
}

func (cfg ServerConfig) ManualPricesMap() map[string]ManualPrice {
	m := make(map[string]ManualPrice)
	for _, mp := range cfg.ManualPrices {
		m[mp.Denom] = mp
	}
	return m
}

func (cfg ServerConfig) DenomMetadataMap() map[string]DenomMetadata {
	m := make(map[string]DenomMetadata)
	for _, md := range cfg.DenomMetadata {
		m[md.Denom] = md
	}
	return m
}

type DenomMetadata struct {
	Denom    string `yaml:"denom"`
	Display  string `yaml:"display"`
	Exponent int    `yaml:"exponent"`
}

type ManualPrice struct {
	Denom    string  `yaml:"denom"`
	MinPrice float64 `yaml:"min_price"`
	MaxPrice float64 `yaml:"max_price"`
}

var DefaultCoinMarketCapConfig = CoinMarketCapConfig{
	UpdateInterval: time.Minute,
}

type CoinMarketCapConfig struct {
	APIKey         string        `yaml:"api_key"`
	UpdateInterval time.Duration `yaml:"update_interval"`
}

var DefaultCyberNodeConfig = CyberNodeConfig{
	UpdateInterval: time.Minute,
}

type CyberNodeConfig struct {
	UpdateInterval time.Duration `yaml:"update_interval"`
}
