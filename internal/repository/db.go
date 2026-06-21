package repository

import (
	"database/sql"
	"time"

	"github.com/cockroachdb/errors"
	_ "github.com/lib/pq"
)

type Database struct {
	DB *sql.DB
}

func NewDatabase(connectionString string) (*Database, error) {
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, errors.Wrap(err, "open postgres connection")
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, errors.Wrap(err, "ping postgres")
	}

	return &Database{DB: db}, nil
}

// Merchant resolution lives in merchant.go (ResolveMerchant), which de-dupes by
// the normalized name.
