package repository

import (
	"database/sql"

	"github.com/cockroachdb/errors"
	"github.com/mmarci96/fin-track/internal/apperr"
	"github.com/mmarci96/fin-track/internal/model"
)

func (db *Database) GetUserByID(id int) (*model.User, error) {
	user := &model.User{}

	err := db.DB.QueryRow(`
		SELECT id, name, email, created_at
		FROM users
		WHERE id = $1
	`, id).Scan(&user.ID, &user.Name, &user.Email, &user.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperr.NotFound("user not found", errors.Wrapf(err, "user id=%d", id))
	}
	if err != nil {
		return nil, errors.Wrapf(err, "query user id=%d", id)
	}

	return user, nil
}

// EnsureDefaultUser guarantees a user with the given id exists, so the app can
// assign unauthenticated uploads to it. Idempotent.
func (db *Database) EnsureDefaultUser(id int) error {
	_, err := db.DB.Exec(`
		INSERT INTO users (id, name)
		VALUES ($1, 'default')
		ON CONFLICT (id) DO NOTHING
	`, id)
	if err != nil {
		return errors.Wrapf(err, "ensure default user id=%d", id)
	}

	return nil
}
