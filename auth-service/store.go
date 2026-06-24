package main

import (
	"database/sql"
	"errors"
	"fmt"

	_ "github.com/lib/pq"
	"golang.org/x/crypto/bcrypt"
)

// Store is the auth database (dedicated auth_uat / auth_dev database).
type Store struct {
	db *sql.DB
}

type User struct {
	ID           int64
	Email        string
	PasswordHash string
	AppUserID    int
	IsActive     bool
}

var errInvalidCredentials = errors.New("invalid credentials")

func openStore(dsn string) (*Store, error) {
	fmt.Println("dsn: ", dsn)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("open auth db: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping auth db: %w", err)
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

// Authenticate verifies the email/password pair and returns the user on success.
// It returns errInvalidCredentials for both unknown email and bad password so
// callers cannot distinguish the two.
func (s *Store) Authenticate(email, password string) (*User, error) {
	var u User
	err := s.db.QueryRow(`
		SELECT id, email, password_hash, app_user_id, is_active
		FROM auth_users
		WHERE email = $1
	`, email).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.AppUserID, &u.IsActive)
	if errors.Is(err, sql.ErrNoRows) {
		// Run a dummy compare to keep timing roughly constant for unknown users.
		_ = bcrypt.CompareHashAndPassword([]byte("$2a$10$invalidinvalidinvalidinvalidinvalidinvalidinvalidinvalidiu"), []byte(password))
		return nil, errInvalidCredentials
	}
	if err != nil {
		return nil, fmt.Errorf("query auth user: %w", err)
	}
	if !u.IsActive {
		return nil, errInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return nil, errInvalidCredentials
	}
	return &u, nil
}

// CreateUser inserts (or updates the password/app_user_id of) a user. Used by
// the `-createuser` admin command so plaintext passwords never touch SQL files.
func (s *Store) CreateUser(email, password string, appUserID int) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	_, err = s.db.Exec(`
		INSERT INTO auth_users (email, password_hash, app_user_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (email) DO UPDATE
		SET password_hash = EXCLUDED.password_hash,
		    app_user_id   = EXCLUDED.app_user_id,
		    is_active     = TRUE
	`, email, string(hash), appUserID)
	if err != nil {
		return fmt.Errorf("upsert auth user: %w", err)
	}
	return nil
}
