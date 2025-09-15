package db

import (
	"context"
	"fmt"
	"time"
	"wheres-my-pizza/pkg/config"
	"wheres-my-pizza/pkg/logger"

	"github.com/jackc/pgx/v5/pgxpool"
)

func ConnectDB(cfg *config.DatabaseConfig, log *logger.Logger) (*pgxpool.Pool, error) {
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database)

	poolConfig, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, err
	}

	err = pool.Ping(ctx)
	if err != nil {
		return nil, err
	}

	log.Info("startup", "db_connected", "Connected to PostgreSQL database")
	return pool, nil
}
