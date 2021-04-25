package server

import (
	"context"
	"fmt"

	"golang.org/x/sync/errgroup"

	"github.com/b-harvest/gravity-dex-backend/util"
)

func (s *Server) RunBackgroundUpdater(ctx context.Context) error {
	ticker := util.NewImmediateTicker(s.cfg.CacheUpdateInterval)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			s.logger.Debug("updating caches")
			if err := s.UpdateCaches(ctx); err != nil {
				return fmt.Errorf("update caches: %w", err)
			}
		}
	}
}

func (s *Server) UpdateCaches(ctx context.Context) error {
	blockHeight, err := s.ss.LatestBlockHeight(ctx)
	if err != nil {
		return fmt.Errorf("get latest block height: %w", err)
	}
	pools, err := s.ss.Pools(ctx, blockHeight)
	if err != nil {
		return fmt.Errorf("get pools: %w", err)
	}
	t, err := s.pts.PriceTable(ctx, pools)
	if err != nil {
		return fmt.Errorf("get price table: %w", err)
	}
	eg, ctx2 := errgroup.WithContext(ctx)
	eg.Go(func() error {
		if err := s.UpdateScoreBoardCache(ctx2, blockHeight, t); err != nil {
			return fmt.Errorf("update score board cache: %w", err)
		}
		return nil
	})
	eg.Go(func() error {
		if err := s.UpdatePriceTableCache(ctx2, blockHeight, pools, t); err != nil {
			return fmt.Errorf("update price table cache: %w", err)
		}
		return nil
	})
	return eg.Wait()
}