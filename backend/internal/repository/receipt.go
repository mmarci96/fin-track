package repository

import (
	"database/sql"

	"github.com/cockroachdb/errors"
	"github.com/mmarci96/fin-track/internal/apperr"
	"github.com/mmarci96/fin-track/internal/model"
)

// resolveCurrency looks up a currency by its code, defaulting to HUF when no
// code is supplied (e.g. the OCR create path, which never detects a currency).
func (db *Database) resolveCurrency(code string) (model.Currency, error) {
	if code == "" {
		code = "HUF"
	}

	cur := model.Currency{Code: code}
	err := db.DB.QueryRow(`SELECT id FROM currencies WHERE code = $1`, code).Scan(&cur.ID)
	if errors.Is(err, sql.ErrNoRows) {
		return cur, apperr.BadRequest("unknown currency: "+code, errors.Wrapf(err, "currency code=%q", code))
	}
	return cur, errors.Wrapf(err, "query currency code=%q", code)
}

func (db *Database) CreateReceipt(receipt *model.Receipt) error {
	cur, err := db.resolveCurrency(receipt.Currency.Code)
	if err != nil {
		return err
	}
	receipt.Currency = cur

	// Insert receipt and return the generated ID. Currency lives on the
	// receipt; products inherit it.
	err = db.DB.QueryRow(`
		INSERT INTO receipts (user_id, merchant_id, currency_id, total_amount)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, receipt.UserID, receipt.MerchantID, cur.ID, receipt.TotalAmount).Scan(&receipt.ID)
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
			m.name,
			cur.id,
			cur.code
		FROM receipts r
		JOIN merchants m ON r.merchant_id = m.id
		JOIN currencies cur ON r.currency_id = cur.id
		WHERE r.id = $1 AND r.user_id = $2
	`, id, userID).Scan(
		&receipt.ID,
		&receipt.UserID,
		&receipt.TotalAmount,
		&receipt.Merchant.ID,
		&receipt.Merchant.Name,
		&receipt.Currency.ID,
		&receipt.Currency.Code,
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
	cur, err := db.resolveCurrency(receipt.Currency.Code)
	if err != nil {
		return err
	}
	receipt.Currency = cur

	// Update receipt total amount and currency, scoped to the owning user. A
	// zero row count means the receipt either does not exist or belongs to
	// another user.
	res, err := db.DB.Exec(`
		UPDATE receipts
		SET total_amount = $1, currency_id = $2
		WHERE id = $3 AND user_id = $4
	`, receipt.TotalAmount, cur.ID, id, userID)
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
	return db.findMerchantNames("")
}

// FindVerifiedMerchants returns only the names of verified (human-curated)
// merchants. The parser matches headers against these alone, so legacy
// auto-created junk rows and the UNKNOWN sentinel can't poison detection.
func (db *Database) FindVerifiedMerchants() ([]string, error) {
	return db.findMerchantNames("WHERE verified")
}

func (db *Database) findMerchantNames(where string) ([]string, error) {
	rows, err := db.DB.Query(`SELECT name FROM merchants ` + where + ` ORDER BY name`)
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
			m.name,
			cur.id,
			cur.code
		FROM receipts r
		JOIN merchants m
			ON r.merchant_id = m.id
		JOIN currencies cur
			ON r.currency_id = cur.id
		WHERE r.user_id = $1
		ORDER BY r.created_at DESC
	`, userID)
	if err != nil {
		return nil, errors.Wrap(err, "query receipts")
	}
	defer rows.Close()

	var receipts []model.Receipt
	index := make(map[int]int) // receipt id -> position in receipts

	for rows.Next() {
		var receipt model.Receipt

		err := rows.Scan(
			&receipt.ID,
			&receipt.UserID,
			&receipt.TotalAmount,
			&receipt.Merchant.ID,
			&receipt.Merchant.Name,
			&receipt.Currency.ID,
			&receipt.Currency.Code,
		)
		if err != nil {
			return nil, errors.Wrap(err, "scan receipt")
		}
		receipt.MerchantID = receipt.Merchant.ID

		index[receipt.ID] = len(receipts)
		receipts = append(receipts, receipt)
	}
	if err := rows.Err(); err != nil {
		return nil, errors.Wrap(err, "iterate receipts")
	}
	rows.Close()

	if len(receipts) == 0 {
		return receipts, nil
	}

	// Attach products for all the user's receipts in one query so the list view
	// can show item counts (categories are omitted; the detail view loads those).
	prodRows, err := db.DB.Query(`
		SELECT p.receipt_id, p.id, p.name, p.price
		FROM products p
		JOIN receipts r ON p.receipt_id = r.id
		WHERE r.user_id = $1
	`, userID)
	if err != nil {
		return nil, errors.Wrap(err, "query receipt products")
	}
	defer prodRows.Close()

	for prodRows.Next() {
		var receiptID int
		var product model.Product
		if err := prodRows.Scan(&receiptID, &product.ID, &product.Name, &product.Price); err != nil {
			return nil, errors.Wrap(err, "scan receipt product")
		}
		if i, ok := index[receiptID]; ok {
			receipts[i].Products = append(receipts[i].Products, product)
		}
	}

	return receipts, errors.Wrap(prodRows.Err(), "iterate receipt products")
}
