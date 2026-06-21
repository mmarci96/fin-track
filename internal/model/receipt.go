package model

type Category string

type Receipt struct {
	Name       string
	Price      int64
	Categories []Category
}
