package pricetable

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/b-harvest/gravity-dex-backend/schema"
)

func TestContext_Price(t *testing.T) {
	ctx := &Context{
		stableCoinDenoms:  []string{"xusd"},
		stakingCoinDenoms: []string{"uatom", "uluna"},
		table: Table{
			"uatom": 20.0,
			"uluna": 10.0,
		},
		pools: map[string]*schema.Pool{
			"pool1": {
				ReserveCoins: []schema.Coin{
					{Denom: "uatom", Amount: 1000000},
					{Denom: "xusd", Amount: 20000000},
				},
				PoolCoin: schema.Coin{Denom: "pool1", Amount: 1000000},
			},
			"pool2": {
				ReserveCoins: []schema.Coin{
					{Denom: "uluna", Amount: 1000000},
					{Denom: "xusd", Amount: 10000000},
				},
				PoolCoin: schema.Coin{Denom: "pool2", Amount: 1000000},
			},
			"pool3": {
				ReserveCoins: []schema.Coin{
					{Denom: "uatom", Amount: 1000000},
					{Denom: "uluna", Amount: 2000000},
				},
				PoolCoin: schema.Coin{Denom: "pool3", Amount: 1000000},
			},
			"pool4": {
				ReserveCoins: []schema.Coin{
					{Denom: "pool1", Amount: 50000},
					{Denom: "pool2", Amount: 100000},
				},
				PoolCoin: schema.Coin{Denom: "pool4", Amount: 1000000},
			},
		},
	}
	for _, tc := range []struct {
		denom string
		price float64
	}{
		{"uatom", 20.0},
		{"uluna", 10.0},
		{"pool1", 40.0},
		{"pool2", 20.0},
		{"pool3", 40.0},
		{"pool4", 4.0},
	} {
		p, err := ctx.Price(tc.denom)
		require.NoError(t, err)
		require.Equal(t, tc.price, p)
	}
}
