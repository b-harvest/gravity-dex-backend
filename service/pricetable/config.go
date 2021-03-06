package pricetable

import (
	"fmt"
)

type Config struct {
	CoinDenoms    []string        `yaml:"coin_denoms"`
	ManualPrices  []ManualPrice   `yaml:"manual_prices"`
	DenomMetadata []DenomMetadata `yaml:"denom_metadata"`
}

func (cfg Config) QueryableDenoms() []string {
	var denoms []string
	mm := cfg.ManualPricesMap()
	for _, d := range cfg.CoinDenoms {
		if _, ok := mm[d]; !ok {
			denoms = append(denoms, d)
		}
	}
	return denoms
}

func (cfg Config) AvailableDenoms() []string {
	denoms := cfg.CoinDenoms
	for _, md := range cfg.DenomMetadata {
		denoms = append(denoms, md.Denom)
	}
	return denoms
}

func (cfg Config) ManualPricesMap() map[string]ManualPrice {
	m := make(map[string]ManualPrice)
	for _, mp := range cfg.ManualPrices {
		m[mp.Denom] = mp
	}
	return m
}

func (cfg Config) DenomMetadataMap() map[string]DenomMetadata {
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

var DefaultConfig = Config{
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
		"com",
		"dsm",
		"run",
	},
	ManualPrices: []ManualPrice{
		{Denom: "run", MinPrice: 1.0, MaxPrice: 1.0},
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
		{Denom: "ucom", Display: "com", Exponent: 6},
		{Denom: "udsm", Display: "dsm", Exponent: 6},
		{Denom: "xrun", Display: "run", Exponent: 6},
	},
}

func (cfg Config) Validate() error {
	if len(cfg.CoinDenoms) == 0 {
		return fmt.Errorf("'coin_denoms' is empty")
	}
	if len(cfg.DenomMetadata) == 0 {
		return fmt.Errorf("'denom_metadata' is empty")
	}
	return nil
}
