package repository

import "github.com/mmarci96/fin-track/internal/model"

func (db *Database) CreateReceipt(receipt *model.Receipt) error {
	// Insert receipt and return the generated ID
	err := db.DB.QueryRow(`
		INSERT INTO receipts (merchant_id, total_amount)
		VALUES ($1, $2)
		RETURNING id
	`, receipt.MerchantID, receipt.TotalAmount).Scan(&receipt.ID)
	if err != nil {
		return err
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
			return err
		}

		// Insert product categories if any
		for _, category := range product.Categories {
			_, err := db.DB.Exec(`
				INSERT INTO product_categories (product_id, category_id)
				VALUES ($1, $2)
			`, productID, category.ID)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (db *Database) GetReceiptByID(id int) (*model.Receipt, error) {
	receipt := &model.Receipt{}

	// Get receipt with merchant info
	err := db.DB.QueryRow(`
		SELECT
			r.id,
			r.total_amount,
			r.merchant_id,
			m.name
		FROM receipts r
		JOIN merchants m ON r.merchant_id = m.id
		WHERE r.id = $1
	`, id).Scan(
		&receipt.ID,
		&receipt.TotalAmount,
		&receipt.Merchant.ID,
		&receipt.Merchant.Name,
	)
	if err != nil {
		return nil, err
	}

	receipt.MerchantID = receipt.Merchant.ID

	// Get products for this receipt
	rows, err := db.DB.Query(`
		SELECT id, name, price
		FROM products
		WHERE receipt_id = $1
	`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		product := model.Product{}
		err := rows.Scan(&product.ID, &product.Name, &product.Price)
		if err != nil {
			return nil, err
		}

		// Get categories for this product
		catRows, err := db.DB.Query(`
			SELECT c.id, c.name
			FROM categories c
			JOIN product_categories pc ON c.id = pc.category_id
			WHERE pc.product_id = $1
		`, product.ID)
		if err != nil {
			return nil, err
		}
		defer catRows.Close()

		for catRows.Next() {
			category := model.Category{}
			err := catRows.Scan(&category.ID, &category.Name)
			if err != nil {
				return nil, err
			}
			product.Categories = append(product.Categories, category)
		}

		receipt.Products = append(receipt.Products, product)
	}

	return receipt, rows.Err()
}

func (db *Database) UpdateReceiptByID(

	id int,
	receipt *model.Receipt,

) error {
	// Update receipt total amount
	_, err := db.DB.Exec(`
		UPDATE receipts
		SET total_amount = $1
		WHERE id = $2
	`, receipt.TotalAmount, id)
	if err != nil {
		return err
	}

	// Delete existing products for this receipt
	_, err = db.DB.Exec(`
		DELETE FROM products
		WHERE receipt_id = $1
	`, id)
	if err != nil {
		return err
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
			return err
		}

		// Insert product categories
		for _, category := range product.Categories {
			_, err := db.DB.Exec(`
				INSERT INTO product_categories (product_id, category_id)
				VALUES ($1, $2)
			`, productID, category.ID)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
func (db *Database) RemoveReceiptByID(id int) error {
	_, err := db.DB.Exec(`
		DELETE FROM receipts
		WHERE id = $1
	`, id)

	return err
}

type ReceiptRepository interface {
	CreateReceipt(receipt *model.Receipt) error
	GetAllReceipts() ([]model.Receipt, error)
	GetReceiptByID(id int) (*model.Receipt, error)
	UpdateReceiptByID(id int, receipt *model.Receipt) error
	RemoveReceiptByID(id int) error
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
