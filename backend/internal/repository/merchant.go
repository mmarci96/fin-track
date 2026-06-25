package repository

import (
	"database/sql"
	"net/http"

	"github.com/cockroachdb/errors"
	"github.com/lib/pq"
	"github.com/mmarci96/fin-track/internal/apperr"
	"github.com/mmarci96/fin-track/internal/model"
	"github.com/mmarci96/fin-track/internal/service/receipt"
)

// ResolveMerchant returns the id of the merchant matching displayName, creating
// it if necessary. De-duplication is by the normalized key, so OCR variants
// ("ALUL MAGYAROROZAG…") collapse onto the canonical row instead of spawning new
// merchants (the old self-poisoning bug).
func (db *Database) ResolveMerchant(displayName string) (int, error) {
	normalized := receipt.NormalizeName(displayName)

	var id int
	err := db.DB.QueryRow(`
		INSERT INTO merchants (name, normalized_name)
		VALUES ($1, $2)
		ON CONFLICT (normalized_name) DO UPDATE SET normalized_name = EXCLUDED.normalized_name
		RETURNING id
	`, displayName, normalized).Scan(&id)
	if err != nil {
		return 0, errors.Wrapf(err, "resolve merchant name=%q", displayName)
	}

	return id, nil
}

const (
	pgUniqueViolation     = "23505"
	pgForeignKeyViolation = "23503"
)

func pgErrCode(err error) string {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		return string(pqErr.Code)
	}
	return ""
}

func (db *Database) CreateMerchant(name string) (*model.Merchant, error) {
	m := &model.Merchant{Name: name}

	// Human-created merchants are trusted, so they are verified (and thus matched
	// against during parsing).
	err := db.DB.QueryRow(`
		INSERT INTO merchants (name, normalized_name, verified)
		VALUES ($1, $2, true)
		RETURNING id
	`, name, receipt.NormalizeName(name)).Scan(&m.ID)
	if pgErrCode(err) == pgUniqueViolation {
		return nil, apperr.Coded("MERCHANT_EXISTS", http.StatusConflict, "merchant already exists", errors.Wrapf(err, "create merchant name=%q", name))
	}
	if err != nil {
		return nil, errors.Wrapf(err, "create merchant name=%q", name)
	}

	return m, nil
}

func (db *Database) GetAllMerchants() ([]model.Merchant, error) {
	rows, err := db.DB.Query(`
		SELECT id, name
		FROM merchants
		ORDER BY name
	`)
	if err != nil {
		return nil, errors.Wrap(err, "query merchants")
	}
	defer rows.Close()

	var merchants []model.Merchant
	for rows.Next() {
		var m model.Merchant
		if err := rows.Scan(&m.ID, &m.Name); err != nil {
			return nil, errors.Wrap(err, "scan merchant")
		}
		merchants = append(merchants, m)
	}

	return merchants, errors.Wrap(rows.Err(), "iterate merchants")
}

func (db *Database) GetMerchantByID(id int) (*model.Merchant, error) {
	m := &model.Merchant{}

	err := db.DB.QueryRow(`
		SELECT id, name
		FROM merchants
		WHERE id = $1
	`, id).Scan(&m.ID, &m.Name)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperr.NotFound("merchant not found", errors.Wrapf(err, "merchant id=%d", id))
	}
	if err != nil {
		return nil, errors.Wrapf(err, "query merchant id=%d", id)
	}

	return m, nil
}

func (db *Database) UpdateMerchant(id int, name string) (*model.Merchant, error) {
	res, err := db.DB.Exec(`
		UPDATE merchants
		SET name = $1, normalized_name = $2
		WHERE id = $3
	`, name, receipt.NormalizeName(name), id)
	if pgErrCode(err) == pgUniqueViolation {
		return nil, apperr.Coded("MERCHANT_EXISTS", http.StatusConflict, "merchant already exists", errors.Wrapf(err, "update merchant id=%d", id))
	}
	if err != nil {
		return nil, errors.Wrapf(err, "update merchant id=%d", id)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, apperr.NotFound("merchant not found", errors.Newf("merchant id=%d", id))
	}

	return &model.Merchant{ID: id, Name: name}, nil
}

func (db *Database) DeleteMerchant(id int) error {
	res, err := db.DB.Exec(`
		DELETE FROM merchants
		WHERE id = $1
	`, id)
	if pgErrCode(err) == pgForeignKeyViolation {
		return apperr.Coded("MERCHANT_IN_USE", http.StatusConflict, "merchant has receipts and cannot be deleted", errors.Wrapf(err, "delete merchant id=%d", id))
	}
	if err != nil {
		return errors.Wrapf(err, "delete merchant id=%d", id)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return apperr.NotFound("merchant not found", errors.Newf("merchant id=%d", id))
	}

	return nil
}
