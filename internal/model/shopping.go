package model

type Merchant struct {
	ID   string
	Name string
}

type Category struct {
	ID   string
	Name string
}

type Product struct {
	ID         string
	Name       string
	Price      int
	Categories []Category
}

type Receipt struct {
	ID            string
	MerchantID    string
	Merchant      Merchant
	Products      []Product
	TotalAmount   int
	ScannedAmount string
}
