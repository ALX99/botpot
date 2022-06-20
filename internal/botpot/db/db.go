package db

import (
	"context"

	"github.com/jackc/pgx/v4"
	"github.com/rs/zerolog/log"
)

// DB represents a database to store information about
// the captured attacks
type DB struct {
	conn *pgx.Conn
	cfg  pgx.ConnConfig
	url  string
}

// NewDB creates a new DB
func NewDB(url string) DB {
	return DB{url: url}
}

// Start connects to the DB
func (db *DB) Start() error {
	log.Info().Msg("Starting Database")
	var err error
	db.conn, err = pgx.Connect(context.TODO(), db.url)
	return err
}

// Stop disconnects from the DB
func (db *DB) Stop() error {
	log.Info().Msg("Stopping Database")
	return db.conn.Close(context.TODO())
}

func (db *DB) BeginTx(f func(pgx.Tx) error) error {
	return db.conn.BeginTxFunc(context.Background(), pgx.TxOptions{}, f)
}
