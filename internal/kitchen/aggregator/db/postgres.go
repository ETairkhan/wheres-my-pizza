package db

import (
	"context"
	"fmt"
	"sync"
	"time"
	"wheres-my-pizza/internal/kitchen/app/core"
	"wheres-my-pizza/internal/xpkg/config"
	"wheres-my-pizza/internal/xpkg/logger"

	"github.com/jackc/pgx/v5"
)

type DB struct {
	ctx   context.Context
	cfg   *config.Postgres
	mylog logger.Logger
	conn  *pgx.Conn
	mu    *sync.Mutex
}

// Start initializes and returns a new DB instance with a single connection
func Start(ctx context.Context, dbCfg *config.Postgres, mylog logger.Logger) (core.IDB, error) {
	d := &DB{
		cfg:   dbCfg,
		ctx:   ctx,
		mylog: mylog,
		mu:    &sync.Mutex{},
	}

	if err := d.connect(); err != nil {
		return nil, err
	}

	return d, nil
}

func (d *DB) GetConn() *pgx.Conn{
	return d.conn
}

// Stop closes the connection
func (d *DB) Close() error {
	if err := d.conn.Close(d.ctx); err != nil {
		return fmt.Errorf("close database connection: %v", err)
	}
	return nil
}

// IsAlive pings the DB to verify it's responsive
func (d *DB) IsAlive() error {
	if d.conn == nil {
		return fmt.Errorf("DB is not initialized")
	}
	if err := d.conn.Ping(d.ctx); err != nil {
		if err := d.connect(); err != nil {
			return fmt.Errorf("ping failed: %w", err)
		}
	}
	return nil
}

func (d *DB) Reconnect() error {
	log := d.mylog.Action("reconnecting")
	attempt := 10
	for i := 0; i < attempt; i++ {
		log.Info("reconnecting attempt", "attempt-number", i+1)
		err := d.IsAlive()
		if err == nil {
			return nil
		}
		time.Sleep(time.Second * 1)
	}
	return fmt.Errorf("reconnecting failed")
}

func (d *DB) connect() error {
	// Establish connection
	conn, err := pgx.Connect(d.ctx, fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		d.cfg.User,
		d.cfg.Password,
		d.cfg.Host,
		d.cfg.Port,
		d.cfg.Database,
	))
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	d.conn = conn
	return nil
}
