package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/b-harvest/gravity-dex-backend/config"
	"github.com/b-harvest/gravity-dex-backend/schema"
)

type Service struct {
	cfg config.MongoDBConfig
	mc  *mongo.Client
}

func NewService(cfg config.MongoDBConfig, mc *mongo.Client) *Service {
	return &Service{cfg, mc}
}

func (s *Service) CheckpointCollection() *mongo.Collection {
	return s.mc.Database(s.cfg.DB).Collection(s.cfg.CheckpointCollection)
}

func (s *Service) AccountCollection() *mongo.Collection {
	return s.mc.Database(s.cfg.DB).Collection(s.cfg.AccountCollection)
}

func (s *Service) AccountMetadataCollection() *mongo.Collection {
	return s.mc.Database(s.cfg.DB).Collection(s.cfg.AccountMetadataCollection)
}

func (s *Service) PoolCollection() *mongo.Collection {
	return s.mc.Database(s.cfg.DB).Collection(s.cfg.PoolCollection)
}

func (s *Service) BannerCollection() *mongo.Collection {
	return s.mc.Database(s.cfg.DB).Collection(s.cfg.BannerCollection)
}

func (s *Service) EnsureDBIndexes(ctx context.Context) ([]string, error) {
	var res []string
	for _, x := range []struct {
		coll *mongo.Collection
		is   []mongo.IndexModel
	}{
		{s.AccountCollection(), []mongo.IndexModel{
			{Keys: bson.D{{schema.AccountBlockHeightKey, 1}}},
			{Keys: bson.D{{schema.AccountBlockHeightKey, 1}, {schema.AccountAddressKey, 1}}},
		}},
		{s.AccountMetadataCollection(), []mongo.IndexModel{
			{Keys: bson.D{{schema.AccountMetadataAddressKey, 1}}},
		}},
		{s.PoolCollection(), []mongo.IndexModel{
			{Keys: bson.D{{schema.PoolBlockHeightKey, 1}}},
			{Keys: bson.D{{schema.PoolBlockHeightKey, 1}, {schema.PoolIDKey, 1}}},
			{Keys: bson.D{{schema.PoolReserveCoinsKey, 1}}},
			{Keys: bson.D{{schema.PoolPoolCoinKey, 1}}},
		}},
		{s.BannerCollection(), []mongo.IndexModel{
			{Keys: bson.D{{schema.BannerVisibleAtKey, 1}}},
			{Keys: bson.D{{schema.BannerStartsAtKey, 1}}},
			{Keys: bson.D{{schema.BannerEndsAtKey, 1}}},
		}},
	} {
		names, err := x.coll.Indexes().CreateMany(ctx, x.is)
		if err != nil {
			return res, err
		}
		res = append(res, names...)
	}
	return res, nil
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
	cur, err := s.AccountCollection().Aggregate(ctx, bson.A{
		bson.M{
			"$match": bson.M{
				schema.AccountBlockHeightKey: blockHeight,
			},
		},
		bson.M{
			"$lookup": bson.M{
				"from":         s.cfg.AccountMetadataCollection,
				"foreignField": schema.AccountMetadataAddressKey,
				"localField":   schema.AccountAddressKey,
				"as":           schema.AccountMetadataKey,
			},
		},
		bson.M{
			"$set": bson.M{
				schema.AccountMetadataKey: bson.M{
					"$arrayElemAt": bson.A{"$" + schema.AccountMetadataKey, 0},
				},
			},
		},
		bson.M{
			"$match": bson.M{
				schema.AccountMetadataKey + "." + schema.AccountMetadataIsBlockedKey: bson.M{
					"$in": bson.A{false, nil},
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("aggregate accounts: %w", err)
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
	cur, err := s.PoolCollection().Find(ctx, bson.M{
		schema.PoolBlockHeightKey:  blockHeight,
		schema.PoolReserveCoinsKey: bson.M{"$ne": nil},
		schema.PoolPoolCoinKey:     bson.M{"$ne": nil},
	})
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

func (s *Service) Banner(ctx context.Context, now time.Time) (*schema.Banner, error) {
	var b schema.Banner
	if err := s.BannerCollection().FindOne(ctx, bson.M{
		schema.BannerVisibleAtKey: bson.M{
			"$lte": now,
		},
		schema.BannerEndsAtKey: bson.M{
			"$gt": now,
		},
	}, options.FindOne().SetSort(bson.M{schema.BannerStartsAtKey: -1})).Decode(&b); err != nil {
		if !errors.Is(err, mongo.ErrNoDocuments) {
			return nil, err
		}
		return nil, nil
	}
	return &b, nil
}
