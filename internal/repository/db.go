package repository

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
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

func (db *Database) GetOrCreateMerchant(name string) (string, error) {
	var id string

	err := db.DB.QueryRow(`
		INSERT INTO merchants (id, name)
		VALUES ($1, $2)
		ON CONFLICT (name)
		DO UPDATE SET name = EXCLUDED.name
		RETURNING id
	`,
		uuid.NewString(),
		name,
	).Scan(&id)

	return id, err
}

func (db *Database) getOrCreateMerchantTx(
	tx *sql.Tx,
	name string,
) (string, error) {

	var id string

	err := tx.QueryRow(`
		SELECT id
		FROM merchants
		WHERE name = $1
	`, name).Scan(&id)

	if err == nil {
		return id, nil
	}

	id = uuid.NewString()

	_, err = tx.Exec(`
		INSERT INTO merchants (id, name)
		VALUES ($1, $2)
		ON CONFLICT (name)
		DO UPDATE SET name = EXCLUDED.name
	`, id, name)

	if err != nil {
		return "", err
	}

	return id, nil
}
