package repository

import (
	"github.com/cockroachdb/errors"
	"github.com/mmarci96/fin-track/internal/service/receipt"
)

// GetMerchantAliases returns every learned header→canonical alias, joined to its
// merchant so the parser gets the canonical name directly. The merchant arm of
// the recognition flywheel.
func (db *Database) GetMerchantAliases() ([]receipt.MerchantAlias, error) {
	rows, err := db.DB.Query(`
		SELECT a.normalized_alias, m.name
		FROM merchant_aliases a
		JOIN merchants m ON m.id = a.merchant_id
	`)
	if err != nil {
		return nil, errors.Wrap(err, "query merchant_aliases")
	}
	defer rows.Close()

	var out []receipt.MerchantAlias
	for rows.Next() {
		var a receipt.MerchantAlias
		if err := rows.Scan(&a.Normalized, &a.Canonical); err != nil {
			return nil, errors.Wrap(err, "scan merchant_alias")
		}
		out = append(out, a)
	}
	return out, errors.Wrap(rows.Err(), "iterate merchant_aliases")
}

// CreateMerchantAlias records (or re-points) a normalized header variant to a
// merchant. sourceImageID may be nil. The normalized_alias UNIQUE constraint
// makes re-learning idempotent: the same variant re-points instead of spawning a
// duplicate row.
func (db *Database) CreateMerchantAlias(merchantID int, normalizedAlias string, sourceImageID *int) error {
	_, err := db.DB.Exec(`
		INSERT INTO merchant_aliases (merchant_id, normalized_alias, source_image_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (normalized_alias)
		DO UPDATE SET merchant_id = EXCLUDED.merchant_id,
		              source_image_id = EXCLUDED.source_image_id
	`, merchantID, normalizedAlias, sourceImageID)
	if err != nil {
		return errors.Wrapf(err, "insert merchant_alias %q -> merchant_id=%d", normalizedAlias, merchantID)
	}
	return nil
}
