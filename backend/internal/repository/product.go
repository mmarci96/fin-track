package repository

import (
	"database/sql"

	"github.com/cockroachdb/errors"
	"github.com/mmarci96/fin-track/internal/apperr"
	"github.com/mmarci96/fin-track/internal/model"
)

func (db *Database) GetProductByID(id int) (*model.Product, error) {
	row := db.DB.QueryRow(`
		SELECT id, name, price
		FROM products
		WHERE id = $1
	`, id)
	var p model.Product
	if err := row.Scan(&p.ID, &p.Name, &p.Price); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperr.NotFound("product not found", errors.Wrapf(err, "get product id=%d", id))
		}
		return nil, errors.Wrapf(err, "get product id=%d", id)
	}
	var categories []model.Category
	rows, err := db.DB.Query(`
		SELECT c.id, c.name
		FROM categories c
		JOIN product_categories pc ON pc.category_id = c.id
		WHERE pc.product_id = $1
	`, id)
	if err != nil {
		return nil, errors.Wrapf(err, "get product categories id=%d", id)
	}
	defer rows.Close()
	for rows.Next() {
		var c model.Category
		if err := rows.Scan(&c.ID, &c.Name); err != nil {
			return nil, errors.Wrapf(err, "scan product category id=%d", id)
		}
		categories = append(categories, c)
	}
	p.Categories = categories
	return &p, nil
}

func (db *Database) CreateProduct(p model.Product) (*model.Product, error) {
	var id int
	err := db.DB.QueryRow(`
		INSERT INTO products (name, price)
		VALUES ($1, $2)
		RETURNING id
	`, p.Name, p.Price).Scan(&id)
	if err != nil {
		return nil, errors.Wrap(err, "create product")
	}
	p.ID = id
	return &p, nil
}

func (db *Database) DeleteProduct(id int) error {
	res, err := db.DB.Exec(`
		DELETE FROM products
		WHERE id = $1
	`, id)
	if err != nil {
		return errors.Wrapf(err, "delete product id=%d", id)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return errors.Wrapf(err, "delete product id=%d", id)
	}
	if rowsAffected == 0 {
		return apperr.NotFound("product not found", errors.Errorf("delete product id=%d: no rows affected", id))
	}
	return nil
}

func (db *Database) UpdateProduct(id int, p model.Product) (*model.Product, error) {
	res, err := db.DB.Exec(`
		UPDATE products
		SET name = $1, price = $2
		WHERE id = $3
	`, p.Name, p.Price, id)
	if err != nil {
		return nil, errors.Wrapf(err, "update product id=%d", id)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return nil, errors.Wrapf(err, "update product id=%d", id)
	}
	if rowsAffected == 0 {
		return nil, apperr.NotFound("product not found", errors.Errorf("update product id=%d: no rows affected", id))
	}
	p.ID = id
	return &p, nil
}
