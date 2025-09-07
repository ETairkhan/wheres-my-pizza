package db

import (
	"context"
	"fmt"
	"wheres-my-pizza/internal/xpkg/config"

	pgx "github.com/jackc/pgx/v5"
)

type DB struct {
	Conn *pgx.Conn
	Ctx  context.Context
}

// Start initializes and returns a new DB instance with a single connection
func Start(ctx context.Context, dbCfg *config.Postgres) (*DB, error) {
	// Build DSN
	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		dbCfg.User,
		dbCfg.Password,
		dbCfg.Host,
		dbCfg.Port,
		dbCfg.Database,
	)

	// Establish connection
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Verify connection
	if err := conn.Ping(ctx); err != nil {
		conn.Close(ctx)
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &DB{
		Conn: conn,
		Ctx:  ctx,
	}, nil
}

func (db *DB) GetConn() *pgx.Conn {
	return db.Conn
}

// Stop closes the connection
func (db *DB) Stop() error {
	if db.Conn != nil {
		return db.Conn.Close(db.Ctx)
	}
	return nil
}
