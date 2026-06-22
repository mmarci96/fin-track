package repository

import (
	"database/sql"
	"net/http"

	"github.com/cockroachdb/errors"
	"github.com/mmarci96/fin-track/internal/apperr"
	"github.com/mmarci96/fin-track/internal/model"
)

func (db *Database) CreateCategory(name string) (*model.Category, error) {
	cat := &model.Category{Name: name}

	err := db.DB.QueryRow(`
		INSERT INTO categories (name)
		VALUES ($1)
		RETURNING id
	`, name).Scan(&cat.ID)
	if pgErrCode(err) == pgUniqueViolation {
		return nil, apperr.Coded("CATEGORY_EXISTS", http.StatusConflict, "category already exists", errors.Wrapf(err, "create category name=%q", name))
	}
	if err != nil {
		return nil, errors.Wrapf(err, "create category name=%q", name)
	}

	return cat, nil
}

func (db *Database) GetAllCurrencies() ([]model.Currency, error) {
	rows, err := db.DB.Query(`
		SELECT id, code 
		FROM currencies 
	`)
	if err != nil {
		return nil, errors.Wrap(err, "query currencies")
	}
	defer rows.Close()
	var currencies []model.Currency
	for rows.Next() {
		var ccy model.Currency
		if err := rows.Scan(&ccy.ID, &ccy.Code); err != nil {
			return nil, errors.Wrap(err, "scan currencies")
		}
		currencies = append(currencies, ccy)
	}
	return currencies, errors.Wrap(rows.Err(), "iterate currencies")
}

func (db *Database) GetAllCategories() ([]model.Category, error) {
	rows, err := db.DB.Query(`
		SELECT id, name
		FROM categories
		ORDER BY name
	`)
	if err != nil {
		return nil, errors.Wrap(err, "query categories")
	}
	defer rows.Close()

	var categories []model.Category
	for rows.Next() {
		var cat model.Category
		if err := rows.Scan(&cat.ID, &cat.Name); err != nil {
			return nil, errors.Wrap(err, "scan category")
		}
		categories = append(categories, cat)
	}

	return categories, errors.Wrap(rows.Err(), "iterate categories")
}

func (db *Database) GetCategoryByID(id int) (*model.Category, error) {
	cat := &model.Category{}

	err := db.DB.QueryRow(`
		SELECT id, name
		FROM categories
		WHERE id = $1
	`, id).Scan(&cat.ID, &cat.Name)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, apperr.NotFound("category not found", errors.Wrapf(err, "category id=%d", id))
	}
	if err != nil {
		return nil, errors.Wrapf(err, "query category id=%d", id)
	}

	return cat, nil
}

func (db *Database) UpdateCategory(id int, name string) (*model.Category, error) {
	res, err := db.DB.Exec(`
		UPDATE categories
		SET name = $1
		WHERE id = $2
	`, name, id)
	if pgErrCode(err) == pgUniqueViolation {
		return nil, apperr.Coded("CATEGORY_EXISTS", http.StatusConflict, "category already exists", errors.Wrapf(err, "update category id=%d", id))
	}
	if err != nil {
		return nil, errors.Wrapf(err, "update category id=%d", id)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil, apperr.NotFound("category not found", errors.Newf("category id=%d", id))
	}

	return &model.Category{ID: id, Name: name}, nil
}

func (db *Database) DeleteCategory(id int) error {
	// product_categories rows are removed by ON DELETE CASCADE.
	res, err := db.DB.Exec(`
		DELETE FROM categories
		WHERE id = $1
	`, id)
	if err != nil {
		return errors.Wrapf(err, "delete category id=%d", id)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return apperr.NotFound("category not found", errors.Newf("category id=%d", id))
	}

	return nil
}

func (db *Database) AssignCategoryToProduct(productID, categoryID int) error {
	_, err := db.DB.Exec(`
		INSERT INTO product_categories (product_id, category_id)
		VALUES ($1, $2)
		ON CONFLICT (product_id, category_id) DO NOTHING
	`, productID, categoryID)
	if pgErrCode(err) == pgForeignKeyViolation {
		return apperr.NotFound("product or category not found", errors.Wrapf(err, "assign category product_id=%d category_id=%d", productID, categoryID))
	}
	if err != nil {
		return errors.Wrapf(err, "assign category product_id=%d category_id=%d", productID, categoryID)
	}

	return nil
}

func (db *Database) UnassignCategoryFromProduct(productID, categoryID int) error {
	res, err := db.DB.Exec(`
		DELETE FROM product_categories
		WHERE product_id = $1 AND category_id = $2
	`, productID, categoryID)
	if err != nil {
		return errors.Wrapf(err, "unassign category product_id=%d category_id=%d", productID, categoryID)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return apperr.NotFound("assignment not found", errors.Newf("product_id=%d category_id=%d", productID, categoryID))
	}

	return nil
}
