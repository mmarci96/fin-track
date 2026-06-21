package repository

import "github.com/mmarci96/fin-track/internal/model"

// func (db *Database) CreateReceipt(receipt *model.Receipt) error
//
// func (db *Database) GetReceiptByID(id string) (*model.Receipt, error)
//
// func (db *Database) UpdateReceiptByID(
//
//	id string,
//	receipt *model.Receipt,
//
// ) error
func (db *Database) RemoveReceiptByID(id string) error {
	_, err := db.DB.Exec(`
		DELETE FROM receipts
		WHERE id = $1
	`, id)

	return err
}

type ReceiptRepository interface {
	CreateReceipt(receipt *model.Receipt) error
	GetAllReceipts() ([]model.Receipt, error)
	GetReceiptByID(id string) (*model.Receipt, error)
	UpdateReceiptByID(id string, receipt *model.Receipt) error
	RemoveReceiptByID(id string) error
}

func (db *Database) FindMerchants() ([]string, error) {
	rows, err := db.DB.Query(`
		SELECT name
		FROM merchants
		ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var merchants []string

	for rows.Next() {
		var merchant string

		if err := rows.Scan(&merchant); err != nil {
			return nil, err
		}

		merchants = append(merchants, merchant)
	}

	return merchants, rows.Err()
}
func (db *Database) GetAllReceipts() ([]model.Receipt, error) {
	rows, err := db.DB.Query(`
		SELECT
			r.id,
			r.total_amount,
			m.id,
			m.name
		FROM receipts r
		JOIN merchants m
			ON r.merchant_id = m.id
		ORDER BY r.created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var receipts []model.Receipt

	for rows.Next() {
		var receipt model.Receipt

		err := rows.Scan(
			&receipt.ID,
			&receipt.TotalAmount,
			&receipt.Merchant.ID,
			&receipt.Merchant.Name,
		)
		if err != nil {
			return nil, err
		}

		receipts = append(receipts, receipt)
	}

	return receipts, rows.Err()
}
