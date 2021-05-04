package transformer

import (
	"context"
	"fmt"
	"os"

	sdk "github.com/cosmos/cosmos-sdk/types"
	liquiditytypes "github.com/tendermint/liquidity/x/liquidity/types"
	"go.uber.org/zap"

	"github.com/b-harvest/gravity-dex-backend/schema"
)

type StateUpdates struct {
	depositStatusByAddress ActionStatusByAddress
	swapStatusByAddress    ActionStatusByAddress
	swapVolumesByPoolID    VolumesByPoolID
}

type ActionStatusByAddress map[string]schema.AccountActionStatus

func (m ActionStatusByAddress) ActionStatus(addr string) schema.AccountActionStatus {
	st, ok := m[addr]
	if !ok {
		st = schema.NewAccountActionStatus()
		m[addr] = st
	}
	return st
}

type VolumesByPoolID map[uint64]schema.Volumes

func (m VolumesByPoolID) Volumes(poolID uint64) schema.Volumes {
	v, ok := m[poolID]
	if !ok {
		v = make(schema.Volumes)
		m[poolID] = v
	}
	return v
}

func (t *Transformer) AccStateUpdates(ctx context.Context, startingBlockHeight int64) (*StateUpdates, *BlockData, error) {
	blockHeight := startingBlockHeight
	updates := &StateUpdates{
		depositStatusByAddress: make(ActionStatusByAddress),
		swapStatusByAddress:    make(ActionStatusByAddress),
		swapVolumesByPoolID:    make(VolumesByPoolID),
	}
	ignoredAddresses := t.cfg.IgnoredAddressesSet()
	var lastData *BlockData
	for {
		select {
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		default:
		}
		var data *BlockData
		var err error
		if blockHeight == startingBlockHeight {
			data, err = t.WaitForBlockData(ctx, blockHeight)
			if err != nil {
				return nil, nil, fmt.Errorf("wait for block data: %w", err)
			}
		} else {
			data, err = t.ReadBlockData(ctx, blockHeight)
			if err != nil {
				if !os.IsNotExist(err) {
					return nil, nil, fmt.Errorf("read block data: %w", err)
				}
				break
			}
		}
		lastData = data
		tm := data.Header.Time.UTC()
		dateKey := tm.Format("2006-01-02")
		t.logger.Debug("handling block data", zap.Int64("height", blockHeight), zap.Time("time", tm))
		for _, evt := range data.Events {
			switch evt.Type {
			case liquiditytypes.EventTypeDepositToPool:
				attrs := eventAttrsFromEvent(evt)
				addr, err := attrs.DepositorAddr()
				if err != nil {
					return nil, nil, err
				}
				if _, ok := ignoredAddresses[addr]; ok {
					continue
				}
				poolID, err := attrs.PoolID()
				if err != nil {
					return nil, nil, err
				}
				st := updates.depositStatusByAddress.ActionStatus(addr)
				st.IncreaseCount(poolID, dateKey, 1)
			case liquiditytypes.EventTypeSwapTransacted:
				attrs := eventAttrsFromEvent(evt)
				addr, err := attrs.SwapRequesterAddr()
				if err != nil {
					return nil, nil, err
				}
				if _, ok := ignoredAddresses[addr]; ok {
					continue
				}
				poolID, err := attrs.PoolID()
				if err != nil {
					return nil, nil, err
				}
				offerCoinFee, err := attrs.OfferCoinFee()
				if err != nil {
					return nil, nil, err
				}
				swapPrice, err := attrs.SwapPrice()
				if err != nil {
					return nil, nil, err
				}
				poolByID := data.PoolByID()
				pool, ok := poolByID[poolID]
				if !ok {
					return nil, nil, fmt.Errorf("pool id %d not found: %w", poolID, err)
				}
				demandCoinDenom, ok := oppositeReserveCoinDenom(pool, offerCoinFee.Denom)
				if !ok {
					return nil, nil, fmt.Errorf("opposite reserve coin denom not found")
				}
				var demandCoinFee sdk.Coin
				if offerCoinFee.Denom < demandCoinDenom {
					demandCoinFee = sdk.NewCoin(demandCoinDenom, offerCoinFee.Amount.ToDec().Quo(swapPrice).TruncateInt())
				} else {
					demandCoinFee = sdk.NewCoin(demandCoinDenom, offerCoinFee.Amount.ToDec().Mul(swapPrice).TruncateInt())
				}
				st := updates.swapStatusByAddress.ActionStatus(addr)
				st.IncreaseCount(poolID, dateKey, 1)
				updates.swapVolumesByPoolID.Volumes(poolID).AddCoins(tm, schema.CoinMap{
					offerCoinFee.Denom:  offerCoinFee.Amount.Int64(),
					demandCoinFee.Denom: demandCoinFee.Amount.Int64(),
				})
			}
		}
		blockHeight++
	}
	return updates, lastData, nil
}
