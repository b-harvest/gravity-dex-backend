package pricetable

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/b-harvest/gravity-dex-backend/config"
	"github.com/b-harvest/gravity-dex-backend/schema"
	"github.com/b-harvest/gravity-dex-backend/service/price"
)

func TestContext_Price(t *testing.T) {
	ctx := &Context{
		stableCoinDenoms:  []string{"uusd"},
		stakingCoinDenoms: []string{"atom", "luna"},
		denomMetadata: map[string]config.DenomMetadata{
			"uusd":  {Display: "usd", Exponent: 6},
			"uatom": {Display: "atom", Exponent: 6},
			"uluna": {Display: "luna", Exponent: 6},
		},
		priceTable: price.Table{
			"atom": 20.0,
			"luna": 10.0,
		},
		pools: map[string]*schema.Pool{
			"pool1": {
				ReserveCoins: []schema.Coin{
					{Denom: "uatom", Amount: 1000000},
					{Denom: "uusd", Amount: 20000000},
				},
				PoolCoin: schema.Coin{Denom: "pool1", Amount: 1000000},
			},
			"pool2": {
				ReserveCoins: []schema.Coin{
					{Denom: "uluna", Amount: 1000000},
					{Denom: "uusd", Amount: 10000000},
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
	for i, tc := range []struct {
		denom string
		price float64
	}{
		{"uatom", 0.00002},
		{"uluna", 0.00001},
		{"pool1", 20.0},
		{"pool2", 10.0},
		{"pool3", 0.00004},
		{"pool4", 2.0},
	} {
		p, err := ctx.Price(tc.denom)
		require.NoError(t, err)
		require.Truef(t, approxEqual(p, tc.price), "%f != %f, tc #%d", p, tc.price, i)
	}
}

func approxEqual(a, b float64) bool {
	return math.Abs(a-b) <= 0.001
}
