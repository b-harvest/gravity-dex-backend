package cmd

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"

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

			ss := store.NewService(cfg.Server, mc)
			ps, err := price.NewCoinMarketCapService(cfg.Server.CoinMarketCapAPIKey)
			if err != nil {
				return fmt.Errorf("new coinmarketcap service: %w", err)
			}
			pts := pricetable.NewService(cfg.Server, ps)
			s := server.New(cfg.Server, ss, ps, pts)

			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				defer wg.Done()
				logger.Info("starting server", zap.String("addr", cfg.Server.BindAddr))
				if err := s.Start(cfg.Server.BindAddr); err != nil && !errors.Is(err, http.ErrServerClosed) {
					logger.Fatal("failed to start server", zap.Error(err))
				}
			}()

			quit := make(chan os.Signal, 1)
			signal.Notify(quit, os.Interrupt)
			<-quit

			logger.Info("gracefully shutting down")
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := s.Shutdown(ctx); err != nil {
				logger.Fatal("failed to shutdown server", zap.Error(err))
			}

			return nil
		},
	}
	return cmd
}
