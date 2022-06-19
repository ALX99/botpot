package db

import (
	"github.com/jackc/pgx"
	"github.com/rs/zerolog/log"
)

// DB represents a database to store information about
// the captured attacks
type DB struct {
	conn *pgx.Conn
	cfg  pgx.ConnConfig
}

// NewDB creates a new DB
func NewDB(cfg pgx.ConnConfig) DB {
	log.Info().Msg("Starting Database")
	return DB{cfg: cfg}
}

// Start connects to the DB
func (db *DB) Start() error {
	var err error
	db.conn, err = pgx.Connect(db.cfg)
	return err
}

// Stop disconnects from the DB
func (db *DB) Stop() error {
	return db.conn.Close()
}
