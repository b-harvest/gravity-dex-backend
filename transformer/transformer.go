package transformer

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	jsoniter "github.com/json-iterator/go"
	liquiditytypes "github.com/tendermint/liquidity/x/liquidity/types"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/b-harvest/gravity-dex-backend/config"
	"github.com/b-harvest/gravity-dex-backend/schema"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

type Transformer struct {
	cfg    config.TransformerConfig
	mc     *mongo.Client
	logger *zap.Logger
}

func New(cfg config.TransformerConfig, mc *mongo.Client, logger *zap.Logger) (*Transformer, error) {
	return &Transformer{cfg, mc, logger}, nil
}

func (t *Transformer) CheckpointCollection() *mongo.Collection {
	return t.mc.Database(t.cfg.MongoDB.DB).Collection(t.cfg.MongoDB.CheckpointCollection)
}

func (t *Transformer) AccountCollection() *mongo.Collection {
	return t.mc.Database(t.cfg.MongoDB.DB).Collection(t.cfg.MongoDB.AccountCollection)
}

func (t *Transformer) PoolCollection() *mongo.Collection {
	return t.mc.Database(t.cfg.MongoDB.DB).Collection(t.cfg.MongoDB.PoolCollection)
}

func (t *Transformer) Run(ctx context.Context) error {
	for {
		t.logger.Debug("getting latest block height")
		h, err := t.LatestBlockHeight(ctx)
		if err != nil {
			return fmt.Errorf("get latest block height: %w", err)
		}
		t.logger.Debug("got latest block height", zap.Int64("height", h))
		t.logger.Debug("pruning old state", zap.Int64("height", h))
		if err := t.PruneOldState(ctx, h+int64(t.cfg.PruningOffset)); err != nil {
			return fmt.Errorf("prune old state: %w", err)
		}
		data, err := t.WaitForBlockData(ctx, h+1)
		if err != nil {
			return fmt.Errorf("wait for next block data: %w", err)
		}
		balances := data.Balances()
		t.logger.Info("updating state", zap.Int64("height", h+1))
		if err := t.UpdateState(ctx, h, data, balances); err != nil {
			return fmt.Errorf("update state: %w", err)
		}
		t.logger.Debug("updating latest block height", zap.Int64("height", h+1))
		if err := t.UpdateLatestBlockHeight(ctx, h+1); err != nil {
			return fmt.Errorf("update latest block height: %w", err)
		}
	}
}

func (t *Transformer) LatestBlockHeight(ctx context.Context) (int64, error) {
	var cp schema.Checkpoint
	if err := t.CheckpointCollection().FindOne(ctx, bson.M{
		schema.CheckpointBlockHeightKey: bson.M{"$exists": true},
	}).Decode(&cp); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return 0, nil
		}
		return 0, err
	}
	return cp.BlockHeight, nil
}

func (t *Transformer) UpdateLatestBlockHeight(ctx context.Context, height int64) error {
	if _, err := t.CheckpointCollection().UpdateOne(ctx, bson.M{
		schema.CheckpointBlockHeightKey: bson.M{"$exists": true},
	}, bson.M{
		"$set": bson.M{
			schema.CheckpointBlockHeightKey: height,
			schema.CheckpointTimestampKey:   time.Now(),
		},
	}, options.Update().SetUpsert(true)); err != nil {
		return err
	}
	return nil
}

func (t *Transformer) PruneOldState(ctx context.Context, currentBlockHeight int64) error {
	if _, err := t.AccountCollection().DeleteMany(ctx, bson.M{
		schema.AccountBlockHeightKey: bson.M{"$lt": currentBlockHeight},
	}); err != nil {
		return fmt.Errorf("prune account collection: %w", err)
	}
	if _, err := t.PoolCollection().DeleteMany(ctx, bson.M{
		schema.PoolBlockHeightKey: bson.M{"$lt": currentBlockHeight},
	}); err != nil {
		return fmt.Errorf("prune pool collection: %w", err)
	}
	return nil
}

func (t *Transformer) blockDataFilename(blockHeight int64) string {
	bs := int64(t.cfg.BlockDataBucketSize)
	p := blockHeight / bs * bs
	return filepath.Join(t.cfg.BlockDataDir, fmt.Sprintf(t.cfg.BlockDataFilename, p, blockHeight))
}

func (t *Transformer) ReadBlockData(blockHeight int64) (*BlockData, error) {
	f, err := os.Open(t.blockDataFilename(blockHeight))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var data BlockData
	if err := json.NewDecoder(f).Decode(&data); err != nil {
		return nil, err
	}
	return &data, nil
}

func (t *Transformer) WaitForBlockData(ctx context.Context, blockHeight int64) (*BlockData, error) {
	ticker := newImmediateTicker(t.cfg.BlockDataWaitingInterval)
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
		}
		t.logger.Debug("waiting for the block data", zap.Int64("height", blockHeight))
		data, err := t.ReadBlockData(blockHeight)
		if err != nil {
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("read block data: %w", err)
			}
		} else {
			return data, nil
		}
	}
}

func (t *Transformer) UpdateState(ctx context.Context, currentBlockHeight int64, data *BlockData, balances map[string][]schema.Coin) error {
	eg, ctx2 := errgroup.WithContext(ctx)
	eg.Go(func() error {
		if err := t.UpdateAccounts(ctx2, currentBlockHeight, data, balances); err != nil {
			return fmt.Errorf("update accounts: %w", err)
		}
		return nil
	})
	eg.Go(func() error {
		if err := t.UpdatePools(ctx2, currentBlockHeight, data, balances); err != nil {
			return fmt.Errorf("update pools: %w", err)
		}
		return nil
	})
	return eg.Wait()
}

func (t *Transformer) UpdateAccounts(ctx context.Context, currentBlockHeight int64, data *BlockData, balances map[string][]schema.Coin) error {
	now := time.Now()
	dateKey := now.Format("2006-01-02")
	newDeposits := make(map[string][]schema.DepositAction)
	newSwaps := make(map[string][]schema.SwapAction)
	for _, evt := range data.Events {
		switch evt.Type {
		case liquiditytypes.EventTypeDepositToPool:
			addr, ok := eventAttributeValue(evt, liquiditytypes.AttributeValueDepositor)
			if !ok {
				return fmt.Errorf("attr %q not found", liquiditytypes.AttributeValueDepositor)
			}
			newDeposits[addr] = append(newDeposits[addr], schema.DepositAction{Timestamp: now})
		case liquiditytypes.EventTypeSwapTransacted:
			addr, ok := eventAttributeValue(evt, liquiditytypes.AttributeValueSwapRequester)
			if !ok {
				return fmt.Errorf("attr %q not found", liquiditytypes.AttributeValueSwapRequester)
			}
			newSwaps[addr] = append(newSwaps[addr], schema.SwapAction{Timestamp: now})
		}
	}
	var writes []mongo.WriteModel
	for _, b := range data.BankModuleState.Balances {
		coins := []schema.Coin{}
		for _, c := range b.Coins {
			coins = append(coins, schema.Coin{Denom: c.Denom, Amount: c.Amount.Int64()})
		}
		var acc schema.Account
		if err := t.AccountCollection().FindOne(ctx, bson.M{
			schema.AccountBlockHeightKey: currentBlockHeight,
			schema.AccountAddressKey:     b.Address,
		}).Decode(&acc); err != nil && !errors.Is(err, mongo.ErrNoDocuments) {
			return fmt.Errorf("find account: %w", err)
		}
		if acc.Actions == nil {
			acc.Actions = make(map[string]schema.ActionState)
		}
		deposits := append(acc.Actions[dateKey].Deposits, newDeposits[b.Address]...)
		if deposits == nil {
			deposits = []schema.DepositAction{}
		}
		swaps := append(acc.Actions[dateKey].Swaps, newSwaps[b.Address]...)
		if swaps == nil {
			swaps = []schema.SwapAction{}
		}
		acc.Actions[dateKey] = schema.ActionState{
			Deposits: deposits,
			Swaps:    swaps,
		}
		writes = append(writes,
			mongo.NewUpdateOneModel().
				SetFilter(bson.M{
					schema.AccountBlockHeightKey: currentBlockHeight + 1,
					schema.AccountAddressKey:     b.Address,
				}).
				SetUpdate(bson.M{
					"$set": bson.M{
						schema.AccountCoinsKey:   coins,
						schema.AccountActionsKey: acc.Actions,
					},
					"$setOnInsert": bson.M{
						schema.AccountUsernameKey: acc.Username,
					},
				}).
				SetUpsert(true))
	}
	if len(writes) > 0 {
		if _, err := t.AccountCollection().BulkWrite(ctx, writes); err != nil {
			return fmt.Errorf("bulk write: %w", err)
		}
	}
	return nil
}

func (t *Transformer) UpdatePools(ctx context.Context, currentBlockHeight int64, data *BlockData, balances map[string][]schema.Coin) error {
	var writes []mongo.WriteModel
	for _, p := range data.Pools {
		var reserveCoins []schema.Coin
		for _, d := range p.ReserveCoinDenoms {
			for _, c := range balances[p.ReserveAccountAddress] {
				if c.Denom == d {
					reserveCoins = append(reserveCoins, c)
					break
				}
			}
		}
		sort.Slice(reserveCoins, func(i, j int) bool { return reserveCoins[i].Denom < reserveCoins[j].Denom })
		poolCoin := schema.Coin{
			Denom:  p.PoolCoinDenom,
			Amount: data.BankModuleState.Supply.AmountOf(p.PoolCoinDenom).Int64(),
		}
		writes = append(writes,
			mongo.NewUpdateOneModel().
				SetFilter(bson.M{
					schema.PoolBlockHeightKey: currentBlockHeight + 1,
					schema.PoolIDKey:          p.Id,
				}).
				SetUpdate(bson.M{
					"$set": bson.M{
						schema.PoolReserveCoins: reserveCoins,
						schema.PoolPoolCoinKey:  poolCoin,
					},
				}).
				SetUpsert(true))
	}
	if len(writes) > 0 {
		if _, err := t.PoolCollection().BulkWrite(ctx, writes); err != nil {
			return fmt.Errorf("bulk write: %w", err)
		}
	}
	return nil
}

type BlockData struct {
	BankModuleState banktypes.GenesisState `json:"bank_module_states"`
	Events          sdk.Events             `json:"end_block_events"`
	Pools           []liquiditytypes.Pool  `json:"pools"`
}

func (d *BlockData) Balances() map[string][]schema.Coin {
	balances := make(map[string][]schema.Coin)
	for _, b := range d.BankModuleState.Balances {
		coins := []schema.Coin{}
		for _, c := range b.Coins {
			coins = append(coins, schema.Coin{Denom: c.Denom, Amount: c.Amount.Int64()})
		}
		balances[b.Address] = coins
	}
	return balances
}

func eventAttributeValue(event sdk.Event, key string) (string, bool) {
	for _, attr := range event.Attributes {
		if string(attr.Key) == key {
			return string(attr.Value), true
		}
	}
	return "", false
}
