package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

// DB represents a database to store information about
// the captured attacks
type DB struct {
	pool *pgxpool.Pool
	url  string
	cfg  pgx.ConnConfig
}

// NewDB creates a new DB
func NewDB(url string) DB {
	return DB{url: url}
}

// Start connects to the DB
func (db *DB) Start() error {
	log.Info().Msg("Starting Database")

	cfg, err := pgxpool.ParseConfig(db.url)
	if err != nil {
		return err
	}

	// Do some additional config
	cfg.HealthCheckPeriod = 10 * time.Second
	cfg.MaxConnIdleTime = 10 * time.Minute
	cfg.AfterRelease = func(_ *pgx.Conn) bool {
		log.Debug().Msg("Connection released")
		return true
	}
	cfg.AfterConnect = func(_ context.Context, _ *pgx.Conn) error {
		log.Debug().Msg("Connection established")
		return nil
	}
	cfg.ConnConfig.ConnectTimeout = 10 * time.Second

	db.pool, err = pgxpool.NewWithConfig(context.TODO(), cfg)
	return err
}

// Stop disconnects from the DB
func (db *DB) Stop() error {
	log.Info().Msg("Stopping Database")
	db.pool.Close()
	return nil
}

func (db *DB) BeginTx(f func(pgx.Tx) error) error {
	return pgx.BeginTxFunc(context.Background(), db.pool, pgx.TxOptions{AccessMode: pgx.ReadWrite}, f)
}
