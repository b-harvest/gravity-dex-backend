package server

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPriceTable(t *testing.T) {
	pool1 := &Pool{
		ReserveCoins:   []Coin{NewStakingCoin(1000000, "ATOM"), NewStableCoin(20000000, "USD")},
		PoolCoinDenom:  "pool/ATOM/USD",
		PoolCoinSupply: 1000000,
	}
	pool2 := &Pool{
		ReserveCoins:   []Coin{NewPoolCoin(1000000, pool1), NewStableCoin(40000000, "USD")},
		PoolCoinDenom:  "pool/ATOM/USD/USD",
		PoolCoinSupply: 1000000,
	}
	pool3 := &Pool{
		ReserveCoins:   []Coin{NewPoolCoin(1000000, pool1), NewPoolCoin(1000000, pool2)},
		PoolCoinDenom:  "pool/ATOM/USD/pool/ATOM/USD/USD",
		PoolCoinSupply: 1000000,
	}
	pt := PriceTable{
		"ATOM": 20.0,
		"BTC":  55000.0,
	}
	for _, tc := range []struct {
		coin  Coin
		value float64
	}{
		{NewStakingCoin(1, "ATOM"), 20.0},
		{NewStakingCoin(1, "BTC"), 55000.0},
		{NewPoolCoin(1, pool1), 40.0},
		{NewPoolCoin(pool1.PoolCoinSupply, pool1), 40000000.0},
		{NewPoolCoin(1, pool2), 80.0},
		{NewPoolCoin(1, pool3), 120.0},
	} {
		v, err := pt.Value(tc.coin)
		require.NoError(t, err)
		require.Equal(t, tc.value, v)
	}
}
