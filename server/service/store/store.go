package store

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/b-harvest/gravity-dex-backend/config"
	"github.com/b-harvest/gravity-dex-backend/schema"
)

type Service struct {
	cfg config.ServerConfig
	mc  *mongo.Client
}

func NewService(cfg config.ServerConfig, mc *mongo.Client) *Service {
	return &Service{cfg, mc}
}

func (s *Service) CheckpointCollection() *mongo.Collection {
	return s.mc.Database(s.cfg.MongoDB.DB).Collection(s.cfg.MongoDB.CheckpointCollection)
}

func (s *Service) AccountCollection() *mongo.Collection {
	return s.mc.Database(s.cfg.MongoDB.DB).Collection(s.cfg.MongoDB.AccountCollection)
}

func (s *Service) PoolCollection() *mongo.Collection {
	return s.mc.Database(s.cfg.MongoDB.DB).Collection(s.cfg.MongoDB.PoolCollection)
}

func (s *Service) LatestBlockHeight(ctx context.Context) (int64, error) {
	var cp schema.Checkpoint
	if err := s.CheckpointCollection().FindOne(ctx, bson.M{
		schema.CheckpointBlockHeightKey: bson.M{"$exists": true},
	}).Decode(&cp); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return 0, nil
		}
		return 0, err
	}
	return cp.BlockHeight, nil
}

func (s *Service) IterateAccounts(ctx context.Context, blockHeight int64, cb func(schema.Account) (stop bool, err error)) error {
	cur, err := s.AccountCollection().Find(ctx, bson.M{schema.AccountBlockHeightKey: blockHeight})
	if err != nil {
		return fmt.Errorf("find accounts: %w", err)
	}
	defer cur.Close(ctx)
	for cur.Next(ctx) {
		var acc schema.Account
		if err := cur.Decode(&acc); err != nil {
			return fmt.Errorf("decode account: %w", err)
		}
		stop, err := cb(acc)
		if err != nil {
			return err
		}
		if stop {
			break
		}
	}
	return nil
}

func (s *Service) Pools(ctx context.Context, blockHeight int64) ([]schema.Pool, error) {
	cur, err := s.PoolCollection().Find(ctx, bson.M{schema.PoolBlockHeightKey: blockHeight})
	if err != nil {
		return nil, fmt.Errorf("find pools: %w", err)
	}
	defer cur.Close(ctx)
	var ps []schema.Pool
	if err := cur.All(ctx, &ps); err != nil {
		return nil, fmt.Errorf("decode pools: %w", err)
	}
	return ps, nil
}
