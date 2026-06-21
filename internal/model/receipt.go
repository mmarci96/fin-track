package model

import "time"

type User struct {
	ID        int
	Name      string
	Email     *string
	CreatedAt time.Time
}

type Merchant struct {
	ID   int
	Name string
}

type Category struct {
	ID   int
	Name string
}

type Product struct {
	ID         int
	Name       string
	Price      int
	Categories []Category
}

type Receipt struct {
	ID            int
	UserID        int
	MerchantID    int
	Merchant      Merchant
	Products      []Product
	TotalAmount   int
	ScannedAmount string
}
