package repository

import (
	"database/sql"

	"github.com/cockroachdb/errors"
	"github.com/mmarci96/fin-track/internal/apperr"
	"github.com/mmarci96/fin-track/internal/model"
)

// CreateReceiptImage persists a debug upload record and fills in the generated
// id and created_at. parse_json is stored as jsonb (NULL when empty).
func (db *Database) CreateReceiptImage(img *model.ReceiptImage) error {
	var parse any
	if len(img.ParseJSON) > 0 {
		parse = string(img.ParseJSON)
	}

	err := db.DB.QueryRow(`
		INSERT INTO receipt_images
			(receipt_id, user_id, stored_path, original_name, content_type, ocr_text, parse_json)
		VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb)
		RETURNING id, created_at
	`,
		img.ReceiptID, img.UserID, img.StoredPath, img.OriginalName,
		img.ContentType, img.OCRText, parse,
	).Scan(&img.ID, &img.CreatedAt)
	if err != nil {
		return errors.Wrapf(err, "insert receipt_image user_id=%d", img.UserID)
	}
	return nil
}

// GetReceiptImageByID returns one debug upload (full detail), scoped to the
// owning user so one user cannot read another's captures.
func (db *Database) GetReceiptImageByID(id, userID int) (*model.ReceiptImage, error) {
	img := &model.ReceiptImage{}
	var parse []byte

	err := db.DB.QueryRow(`
		SELECT id, receipt_id, user_id, stored_path, original_name, content_type,
		       ocr_text, clean_text, parse_json, created_at
		FROM receipt_images
		WHERE id = $1 AND user_id = $2
	`, id, userID).Scan(
		&img.ID, &img.ReceiptID, &img.UserID, &img.StoredPath, &img.OriginalName,
		&img.ContentType, &img.CleanText, &parse, &img.CreatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperr.NotFound("receipt image not found", errors.Wrapf(err, "receipt_image id=%d user_id=%d", id, userID))
	}
	if err != nil {
		return nil, errors.Wrapf(err, "query receipt_image id=%d", id)
	}
	img.ParseJSON = parse
	return img, nil
}

// ListReceiptImages returns the user's debug uploads newest-first. Heavy text
// columns (ocr_text, clean_text, parse_json) are omitted to keep the list cheap;
// the detail endpoint loads those.
func (db *Database) ListReceiptImages(userID int) ([]model.ReceiptImage, error) {
	rows, err := db.DB.Query(`
		SELECT id, receipt_id, user_id, original_name, content_type, created_at
		FROM receipt_images
		WHERE user_id = $1
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, errors.Wrapf(err, "list receipt_images user_id=%d", userID)
	}
	defer rows.Close()

	var out []model.ReceiptImage
	for rows.Next() {
		var img model.ReceiptImage
		if err := rows.Scan(
			&img.ID, &img.ReceiptID, &img.UserID, &img.OriginalName,
			&img.ContentType, &img.CreatedAt,
		); err != nil {
			return nil, errors.Wrap(err, "scan receipt_image")
		}
		out = append(out, img)
	}
	return out, errors.Wrap(rows.Err(), "iterate receipt_images")
}

// SetReceiptImageCleanText stores a human-corrected transcript for a capture,
// scoped to the owning user. This is the ground-truth label for the flywheel.
func (db *Database) SetReceiptImageCleanText(id, userID int, clean string) error {
	res, err := db.DB.Exec(`
		UPDATE receipt_images
		SET clean_text = $1
		WHERE id = $2 AND user_id = $3
	`, clean, id, userID)
	if err != nil {
		return errors.Wrapf(err, "update receipt_image clean_text id=%d", id)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return apperr.NotFound("receipt image not found", errors.Newf("receipt_image id=%d user_id=%d", id, userID))
	}
	return nil
}
