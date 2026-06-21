package repository

import (
	"database/sql"
	"time"

	_ "github.com/lib/pq"
)

type Database struct {
	DB *sql.DB
}

func NewDatabase(connectionString string) (*Database, error) {
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return &Database{DB: db}, nil
}

func (db *Database) GetOrCreateMerchant(name string) (int, error) {
	var id int

	err := db.DB.QueryRow(`
		INSERT INTO merchants (name)
		VALUES ($1)
		ON CONFLICT (name)
		DO UPDATE SET name = EXCLUDED.name
		RETURNING id
	`,
		name,
	).Scan(&id)

	return id, err
}

func (db *Database) getOrCreateMerchantTx(
	tx *sql.Tx,
	name string,
) (int, error) {
	var id int

	err := tx.QueryRow(`
		SELECT id
		FROM merchants
		WHERE name = $1
	`, name).Scan(&id)

	if err == nil {
		return id, nil
	}

	err = tx.QueryRow(`
		INSERT INTO merchants (name)
		VALUES ($1)
		ON CONFLICT (name)
		DO UPDATE SET name = EXCLUDED.name
		RETURNING id
	`, name).Scan(&id)

	if err != nil {
		return 0, err
	}

	return id, nil
}
