package cmd

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/b-harvest/gravity-dex-backend/config"
	"github.com/b-harvest/gravity-dex-backend/server"
	"github.com/b-harvest/gravity-dex-backend/server/service/price"
	"github.com/b-harvest/gravity-dex-backend/server/service/pricetable"
	"github.com/b-harvest/gravity-dex-backend/server/service/store"
)

func ServerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "server",
		Short: "run web server",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true

			cfg, err := config.Load("config.yml")
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}
			if err := cfg.Server.Validate(); err != nil {
				return fmt.Errorf("validate server config: %w", err)
			}

			logger, err := cfg.Server.Log.Build()
			if err != nil {
				return fmt.Errorf("build logger: %w", err)
			}
			defer logger.Sync()

			mc, err := mongo.Connect(context.Background(), options.Client().ApplyURI(cfg.Server.MongoDB.URI))
			if err != nil {
				return fmt.Errorf("connect mongodb: %w", err)
			}
			defer mc.Disconnect(context.Background())

			rp := &redis.Pool{
				Dial: func() (redis.Conn, error) {
					return redis.DialURL(cfg.Server.Redis.URI)
				},
			}
			defer rp.Close()
			conn := rp.Get()
			if _, err := conn.Do("PING"); err != nil {
				conn.Close()
				return fmt.Errorf("connect redis: %w", err)
			}
			conn.Close()

			ss := store.NewService(cfg.Server, mc)
			ps, err := price.NewCoinMarketCapService(cfg.Server.CoinMarketCapAPIKey)
			if err != nil {
				return fmt.Errorf("new coinmarketcap service: %w", err)
			}
			pts := pricetable.NewService(cfg.Server, ps)
			s := server.New(cfg.Server, ss, ps, pts, rp, logger)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			eg, ctx2 := errgroup.WithContext(ctx)
			eg.Go(func() error {
				logger.Info("starting server", zap.String("addr", cfg.Server.BindAddr))
				if err := s.Start(cfg.Server.BindAddr); err != nil && !errors.Is(err, http.ErrServerClosed) {
					return fmt.Errorf("run server: %w", err)
				}
				return nil
			})
			eg.Go(func() error {
				return s.RunBackgroundUpdater(ctx2)
			})

			quit := make(chan os.Signal, 1)
			signal.Notify(quit, os.Interrupt)
			<-quit

			logger.Info("gracefully shutting down")
			if err := s.ShutdownWithTimeout(10 * time.Second); err != nil {
				return fmt.Errorf("shutdown server: %w", err)
			}

			cancel()
			if err := eg.Wait(); !errors.Is(err, context.Canceled) {
				return err
			}
			return nil
		},
	}
	return cmd
}
