package model

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
	MerchantID    int
	Merchant      Merchant
	Products      []Product
	TotalAmount   int
	ScannedAmount string
}
