package model

import "time"

type Currency struct {
	ID   int
	Code string
}

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
	Currency      Currency
}

// ReceiptImage is a persisted debug upload: the original image (on disk at
// StoredPath, under config.ImageStoreDir) plus the raw OCR transcript and the
// parser output, optionally linked to the receipt it produced. CleanText holds
// a later human-corrected transcript — the ground truth the recognition
// flywheel learns from.
type ReceiptImage struct {
	ID           int        `json:"id"`
	ReceiptID    *int       `json:"receipt_id"`
	UserID       int        `json:"user_id"`
	StoredPath   string     `json:"-"` // server-side path, never exposed
	OriginalName string     `json:"original_name"`
	ContentType  string     `json:"content_type"`
	OCRText      string     `json:"ocr_text"`
	CleanText    *string    `json:"clean_text"`
	ParseJSON    []byte     `json:"parse,omitempty"`       // raw receipt.Result JSON
	CleanParseJSON []byte   `json:"clean_parse,omitempty"` // receipt.Result of re-parsing clean_text
	ApprovedAt   *time.Time `json:"approved_at,omitempty"`
	// Approved mirrors "approved_at IS NOT NULL". A concrete field (not a method)
	// so it serializes when a ReceiptImage is marshaled directly (the list
	// endpoint). Repository reads set it alongside ApprovedAt.
	Approved bool `json:"approved"`

	CreatedAt time.Time `json:"created_at"`
}
