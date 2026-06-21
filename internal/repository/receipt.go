package repository

import (
	"database/sql"

	"github.com/cockroachdb/errors"
	"github.com/mmarci96/fin-track/internal/apperr"
	"github.com/mmarci96/fin-track/internal/model"
)

func (db *Database) CreateReceipt(receipt *model.Receipt) error {
	// Insert receipt and return the generated ID
	err := db.DB.QueryRow(`
		INSERT INTO receipts (user_id, merchant_id, total_amount)
		VALUES ($1, $2, $3)
		RETURNING id
	`, receipt.UserID, receipt.MerchantID, receipt.TotalAmount).Scan(&receipt.ID)
	if err != nil {
		return errors.Wrapf(err, "insert receipt user_id=%d merchant_id=%d", receipt.UserID, receipt.MerchantID)
	}

	// Insert products if any
	for _, product := range receipt.Products {
		var productID int

		err := db.DB.QueryRow(`
			INSERT INTO products (receipt_id, name, price)
			VALUES ($1, $2, $3)
			RETURNING id
		`, receipt.ID, product.Name, product.Price).Scan(&productID)
		if err != nil {
			return errors.Wrapf(err, "insert product receipt_id=%d name=%q", receipt.ID, product.Name)
		}

		// Insert product categories if any
		for _, category := range product.Categories {
			_, err := db.DB.Exec(`
				INSERT INTO product_categories (product_id, category_id)
				VALUES ($1, $2)
			`, productID, category.ID)
			if err != nil {
				return errors.Wrapf(err, "insert product_category product_id=%d category_id=%d", productID, category.ID)
			}
		}
	}

	return nil
}

func (db *Database) GetReceiptByID(id, userID int) (*model.Receipt, error) {
	receipt := &model.Receipt{}

	// Get receipt with merchant info, scoped to the owning user.
	err := db.DB.QueryRow(`
		SELECT
			r.id,
			r.user_id,
			r.total_amount,
			r.merchant_id,
			m.name
		FROM receipts r
		JOIN merchants m ON r.merchant_id = m.id
		WHERE r.id = $1 AND r.user_id = $2
	`, id, userID).Scan(
		&receipt.ID,
		&receipt.UserID,
		&receipt.TotalAmount,
		&receipt.Merchant.ID,
		&receipt.Merchant.Name,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperr.NotFound("receipt not found", errors.Wrapf(err, "receipt id=%d user_id=%d", id, userID))
	}
	if err != nil {
		return nil, errors.Wrapf(err, "query receipt id=%d", id)
	}

	receipt.MerchantID = receipt.Merchant.ID

	// Get products for this receipt
	rows, err := db.DB.Query(`
		SELECT id, name, price
		FROM products
		WHERE receipt_id = $1
	`, id)
	if err != nil {
		return nil, errors.Wrapf(err, "query products receipt_id=%d", id)
	}
	defer rows.Close()

	for rows.Next() {
		product := model.Product{}
		err := rows.Scan(&product.ID, &product.Name, &product.Price)
		if err != nil {
			return nil, errors.Wrapf(err, "scan product receipt_id=%d", id)
		}

		// Get categories for this product
		catRows, err := db.DB.Query(`
			SELECT c.id, c.name
			FROM categories c
			JOIN product_categories pc ON c.id = pc.category_id
			WHERE pc.product_id = $1
		`, product.ID)
		if err != nil {
			return nil, errors.Wrapf(err, "query categories product_id=%d", product.ID)
		}
		defer catRows.Close()

		for catRows.Next() {
			category := model.Category{}
			err := catRows.Scan(&category.ID, &category.Name)
			if err != nil {
				return nil, errors.Wrapf(err, "scan category product_id=%d", product.ID)
			}
			product.Categories = append(product.Categories, category)
		}

		receipt.Products = append(receipt.Products, product)
	}

	return receipt, errors.Wrap(rows.Err(), "iterate products")
}

func (db *Database) UpdateReceiptByID(
	id, userID int,
	receipt *model.Receipt,
) error {
	// Update receipt total amount, scoped to the owning user. A zero row count
	// means the receipt either does not exist or belongs to another user.
	res, err := db.DB.Exec(`
		UPDATE receipts
		SET total_amount = $1
		WHERE id = $2 AND user_id = $3
	`, receipt.TotalAmount, id, userID)
	if err != nil {
		return errors.Wrapf(err, "update receipt id=%d", id)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return apperr.NotFound("receipt not found", errors.Newf("receipt id=%d user_id=%d", id, userID))
	}

	// Delete existing products for this receipt
	_, err = db.DB.Exec(`
		DELETE FROM products
		WHERE receipt_id = $1
	`, id)
	if err != nil {
		return errors.Wrapf(err, "delete products receipt_id=%d", id)
	}

	// Re-insert products
	for _, product := range receipt.Products {
		var productID int

		err := db.DB.QueryRow(`
			INSERT INTO products (receipt_id, name, price)
			VALUES ($1, $2, $3)
			RETURNING id
		`, id, product.Name, product.Price).Scan(&productID)
		if err != nil {
			return errors.Wrapf(err, "reinsert product receipt_id=%d name=%q", id, product.Name)
		}

		// Insert product categories
		for _, category := range product.Categories {
			_, err := db.DB.Exec(`
				INSERT INTO product_categories (product_id, category_id)
				VALUES ($1, $2)
			`, productID, category.ID)
			if err != nil {
				return errors.Wrapf(err, "reinsert product_category product_id=%d category_id=%d", productID, category.ID)
			}
		}
	}

	return nil
}

func (db *Database) RemoveReceiptByID(id, userID int) error {
	res, err := db.DB.Exec(`
		DELETE FROM receipts
		WHERE id = $1 AND user_id = $2
	`, id, userID)
	if err != nil {
		return errors.Wrapf(err, "delete receipt id=%d", id)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return apperr.NotFound("receipt not found", errors.Newf("receipt id=%d user_id=%d", id, userID))
	}

	return nil
}

type ReceiptRepository interface {
	CreateReceipt(receipt *model.Receipt) error
	GetAllReceipts(userID int) ([]model.Receipt, error)
	GetReceiptByID(id, userID int) (*model.Receipt, error)
	UpdateReceiptByID(id, userID int, receipt *model.Receipt) error
	RemoveReceiptByID(id, userID int) error
}

func (db *Database) FindMerchants() ([]string, error) {
	rows, err := db.DB.Query(`
		SELECT name
		FROM merchants
		ORDER BY name
	`)
	if err != nil {
		return nil, errors.Wrap(err, "query merchants")
	}
	defer rows.Close()

	var merchants []string

	for rows.Next() {
		var merchant string

		if err := rows.Scan(&merchant); err != nil {
			return nil, errors.Wrap(err, "scan merchant")
		}

		merchants = append(merchants, merchant)
	}

	return merchants, errors.Wrap(rows.Err(), "iterate merchants")
}

func (db *Database) GetAllReceipts(userID int) ([]model.Receipt, error) {
	rows, err := db.DB.Query(`
		SELECT
			r.id,
			r.user_id,
			r.total_amount,
			m.id,
			m.name
		FROM receipts r
		JOIN merchants m
			ON r.merchant_id = m.id
		WHERE r.user_id = $1
		ORDER BY r.created_at DESC
	`, userID)
	if err != nil {
		return nil, errors.Wrap(err, "query receipts")
	}
	defer rows.Close()

	var receipts []model.Receipt

	for rows.Next() {
		var receipt model.Receipt

		err := rows.Scan(
			&receipt.ID,
			&receipt.UserID,
			&receipt.TotalAmount,
			&receipt.Merchant.ID,
			&receipt.Merchant.Name,
		)
		if err != nil {
			return nil, errors.Wrap(err, "scan receipt")
		}
		receipt.MerchantID = receipt.Merchant.ID

		receipts = append(receipts, receipt)
	}

	return receipts, errors.Wrap(rows.Err(), "iterate receipts")
}
