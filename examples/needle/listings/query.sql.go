// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.16.0-69-g345d6d6a-dirty-wicked-fork
// source: query.sql

package listings

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"
)

const addListing = `-- name: AddListing :one
INSERT INTO Listings (
  ItemID, MakerID, Price
) VALUES (
  $1, $2, $3
) RETURNING ID
`

type AddListingParams struct {
	Itemid  int32
	Makerid int32
	Price   int64
}

func (q *Queries) AddListing(ctx context.Context, arg AddListingParams) (*int64, error) {
	row := q.db.WQueryRow(ctx, "AddListing", addListing, arg.Itemid, arg.Makerid, arg.Price)
	var id *int64 = new(int64)
	err := row.Scan(id)
	if err == pgx.ErrNoRows {
		return (*int64)(nil), nil
	} else if err != nil {
		return nil, err
	}

	return id, err
}

const listOrdersOfItem = `-- name: ListOrdersOfItem :many
select id, itemid, makerid, price, createdat FROM Listings
WHERE
  ItemID = $1 AND ID > $2
LIMIT $3
`

type ListOrdersOfItemParams struct {
	Itemid int32
	After  int64
	First  int32
}

// CacheKey - cache key
func (arg ListOrdersOfItemParams) CacheKey() string {
	prefix := "listings:ListOrdersOfItem:"
	return prefix + fmt.Sprintf("%+v,%+v,%+v", arg.Itemid, arg.After, arg.First)
}

// -- cache : 10s
func (q *Queries) ListOrdersOfItem(ctx context.Context, arg ListOrdersOfItemParams) ([]Listing, error) {
	dbRead := func() (any, time.Duration, error) {
		cacheDuration := time.Duration(time.Millisecond * 10000)
		rows, err := q.db.WQuery(ctx, "ListOrdersOfItem", listOrdersOfItem, arg.Itemid, arg.After, arg.First)
		if err != nil {
			return nil, 0, err
		}
		defer rows.Close()
		var items []Listing
		for rows.Next() {
			var i *Listing = new(Listing)
			if err := rows.Scan(
				&i.ID,
				&i.Itemid,
				&i.Makerid,
				&i.Price,
				&i.CreatedAt,
			); err != nil {
				return nil, 0, err
			}
			items = append(items, *i)
		}
		if err := rows.Err(); err != nil {
			return nil, 0, err
		}
		return items, cacheDuration, nil
	}
	if q.cache == nil {
		items, _, err := dbRead()
		return items.([]Listing), err
	}
	var items []Listing
	err := q.cache.GetWithTtl(ctx, arg.CacheKey(), &items, dbRead, false, false)
	if err != nil {
		return nil, err
	}

	return items, err
}

//// auto generated functions

func (q *Queries) Dump(ctx context.Context, beforeDump ...BeforeDump) ([]byte, error) {
	sql := "SELECT id,itemid,makerid,price,createdat FROM listings ORDER BY id,itemid,makerid,price,createdat ASC;"
	rows, err := q.db.WQuery(ctx, "Dump", sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Listing
	for rows.Next() {
		var v Listing
		if err := rows.Scan(&v.ID, &v.Itemid, &v.Makerid, &v.Price, &v.CreatedAt); err != nil {
			return nil, err
		}
		for _, applyBeforeDump := range beforeDump {
			applyBeforeDump(&v)
		}
		items = append(items, v)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	bytes, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return nil, err
	}
	return bytes, nil
}

func (q *Queries) Load(ctx context.Context, data []byte) error {
	sql := "INSERT INTO listings (id,itemid,makerid,price,createdat) VALUES ($1,$2,$3,$4,$5);"
	rows := make([]Listing, 0)
	err := json.Unmarshal(data, &rows)
	if err != nil {
		return err
	}
	for _, row := range rows {
		_, err := q.db.WExec(ctx, "Load", sql, row.ID, row.Itemid, row.Makerid, row.Price, row.CreatedAt)
		if err != nil {
			return err
		}
	}
	return nil
}

// eliminate unused error
var _ = log.Logger
var _ = fmt.Sprintf("")
var _ = time.Now()
var _ = json.RawMessage{}
