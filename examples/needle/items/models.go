// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.16.0

package items

import (
	"database/sql/driver"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type Itemcategory string

const (
	ItemcategoryALCOHOL   Itemcategory = "ALCOHOL "
	ItemcategoryDRUG      Itemcategory = "DRUG"
	ItemcategoryDRINK     Itemcategory = "DRINK"
	ItemcategoryFRUIT     Itemcategory = "FRUIT"
	ItemcategoryVEGETABLE Itemcategory = "VEGETABLE"
)

func (e *Itemcategory) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = Itemcategory(s)
	case string:
		*e = Itemcategory(s)
	default:
		return fmt.Errorf("unsupported scan type for Itemcategory: %T", src)
	}
	return nil
}

type NullItemcategory struct {
	Itemcategory Itemcategory
	Valid        bool // Valid is true if Itemcategory is not NULL
}

// Scan implements the Scanner interface.
func (ns *NullItemcategory) Scan(value interface{}) error {
	if value == nil {
		ns.Itemcategory, ns.Valid = "", false
		return nil
	}
	ns.Valid = true
	return ns.Itemcategory.Scan(value)
}

// Value implements the driver Valuer interface.
func (ns NullItemcategory) Value() (driver.Value, error) {
	if !ns.Valid {
		return nil, nil
	}
	return string(ns.Itemcategory), nil
}

func (e Itemcategory) Valid() bool {
	switch e {
	case ItemcategoryALCOHOL,
		ItemcategoryDRUG,
		ItemcategoryDRINK,
		ItemcategoryFRUIT,
		ItemcategoryVEGETABLE:
		return true
	}
	return false
}

func AllItemcategoryValues() []Itemcategory {
	return []Itemcategory{
		ItemcategoryALCOHOL,
		ItemcategoryDRUG,
		ItemcategoryDRINK,
		ItemcategoryFRUIT,
		ItemcategoryVEGETABLE,
	}
}

type Item struct {
	ID          int64          `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Category    Itemcategory   `json:"category"`
	Price       pgtype.Numeric `json:"price"`
	Thumbnail   string         `json:"thumbnail"`
	Qrcode      *string        `json:"qrcode"`
	Metadata    []byte         `json:"metadata"`
	CreatedAt   time.Time      `json:"createdat"`
	UpdatedAt   time.Time      `json:"updatedat"`
}