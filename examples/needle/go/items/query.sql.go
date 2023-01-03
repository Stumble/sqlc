// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.16.0
// source: query.sql

package items

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v5"
)

const createItems = `-- name: CreateItems :one
INSERT INTO Items (
  Name, Description, Category, Price, Thumbnail, Metadata
) VALUES (
  $1, $2, $3, $4, $5, $6
)
RETURNING id, name, description, category, price, thumbnail, metadata, createdat, updatedat
`

type CreateItemsParams struct {
	Name        string
	Description string
	Category    Itemcategory
	Price       pgtype.Numeric
	Thumbnail   string
	Metadata    []byte
}

// CacheKey - cache key
func (arg CreateItemsParams) CacheKey() string {
	prefix := "CreateItems:"
	return prefix + fmt.Sprintf("%+v,%+v,%+v,%+v,%+v,%+v",
		arg.Name,
		arg.Description,
		arg.Category,
		arg.Price,
		arg.Thumbnail,
		arg.Metadata,
	)
}

func (q *Queries) CreateItems(ctx context.Context, arg CreateItemsParams) (*Item, error) {
	// TODO(mustRevalidate, noStore)
	dbRead := func() (any, time.Duration, error) {
		cacheDuration := time.Duration(time.Millisecond * 0)
		row := q.db.WQueryRow(ctx, "CreateItems", createItems,

			arg.Name,
			arg.Description,
			arg.Category,
			arg.Price,
			arg.Thumbnail,
			arg.Metadata,
		)
		i := &Item{}
		err := row.Scan(
			&i.ID,
			&i.Name,
			&i.Description,
			&i.Category,
			&i.Price,
			&i.Thumbnail,
			&i.Metadata,
			&i.CreatedAt,
			&i.UpdatedAt,
		)
		if err == pgx.ErrNoRows {
			return nil, cacheDuration, nil
		}
		return i, cacheDuration, err
	}
	if q.cache == nil {
		rv, _, err := dbRead()
		return rv.(*Item), err
	}

	var rv *Item
	err := q.cache.GetWithTtl(ctx, arg.CacheKey(), &rv, dbRead, false, false)
	if err != nil {
		return nil, err
	}

	return rv, err
}

const deleteItem = `-- name: DeleteItem :exec
DELETE FROM Items
WHERE id = $1
`

// -- invalidate : [GetItemByID, ListItems]
func (q *Queries) DeleteItem(ctx context.Context, id int64, getItemByID *int64, listItems *ListItemsParams) error {
	_, err := q.db.WExec(ctx, "DeleteItem", deleteItem, id)
	if err != nil {
		return err
	}

	// invalidate
	invalidateErr := q.db.PostExec(func() error {
		var anyErr error
		if getItemByID != nil {
			err = q.cache.Invalidate(ctx, fmt.Sprintf("GetItemByID:%+v", *getItemByID))
			if err != nil {
				anyErr = err
			}
		}
		if listItems != nil {
			err = q.cache.Invalidate(ctx, listItems.CacheKey())
			if err != nil {
				anyErr = err
			}
		}
		return anyErr
	})
	if invalidateErr != nil {
		// invalidateErr is ignored for now.
	}

	return nil
}

const getItemByID = `-- name: GetItemByID :one
SELECT id, name, description, category, price, thumbnail, metadata, createdat, updatedat FROM Items
WHERE id = $1 LIMIT 1
`

// -- cache : 5m
func (q *Queries) GetItemByID(ctx context.Context, id int64) (*Item, error) {
	// TODO(mustRevalidate, noStore)
	dbRead := func() (any, time.Duration, error) {
		cacheDuration := time.Duration(time.Millisecond * 300000)
		row := q.db.WQueryRow(ctx, "GetItemByID", getItemByID,
			id)
		i := &Item{}
		err := row.Scan(
			&i.ID,
			&i.Name,
			&i.Description,
			&i.Category,
			&i.Price,
			&i.Thumbnail,
			&i.Metadata,
			&i.CreatedAt,
			&i.UpdatedAt,
		)
		if err == pgx.ErrNoRows {
			return nil, cacheDuration, nil
		}
		return i, cacheDuration, err
	}
	if q.cache == nil {
		rv, _, err := dbRead()
		return rv.(*Item), err
	}

	var rv *Item
	err := q.cache.GetWithTtl(ctx, fmt.Sprintf("GetItemByID:%+v", id), &rv, dbRead, false, false)
	if err != nil {
		return nil, err
	}

	return rv, err
}

const listItems = `-- name: ListItems :many
SELECT id, name, description, category, price, thumbnail, metadata, createdat, updatedat FROM Items
WHERE id > $1
ORDER BY id
LIMIT $2
`

type ListItemsParams struct {
	After int64
	First int32
}

// CacheKey - cache key
func (arg ListItemsParams) CacheKey() string {
	prefix := "ListItems:"
	return prefix + fmt.Sprintf("%+v,%+v", arg.After, arg.First)
}

func (q *Queries) ListItems(ctx context.Context, arg ListItemsParams) ([]Item, error) {
	dbRead := func() (any, time.Duration, error) {
		cacheDuration := time.Duration(time.Millisecond * 0)
		rows, err := q.db.WQuery(ctx, "ListItems", listItems, arg.After, arg.First)
		if err != nil {
			return nil, 0, err
		}
		defer rows.Close()
		var items []Item
		for rows.Next() {
			var i Item
			if err := rows.Scan(
				&i.ID,
				&i.Name,
				&i.Description,
				&i.Category,
				&i.Price,
				&i.Thumbnail,
				&i.Metadata,
				&i.CreatedAt,
				&i.UpdatedAt,
			); err != nil {
				return nil, 0, err
			}
			items = append(items, i)
		}
		if err := rows.Err(); err != nil {
			return nil, 0, err
		}
		return items, cacheDuration, nil
	}
	if q.cache == nil {
		items, _, err := dbRead()
		return items.([]Item), err
	}

	var items []Item
	err := q.cache.GetWithTtl(ctx, arg.CacheKey(), &items, dbRead, false, false)
	if err != nil {
		return nil, err
	}

	return items, err
}

const listSomeItems = `-- name: ListSomeItems :many
SELECT id, name, description, category, price, thumbnail, metadata, createdat, updatedat FROM Items
WHERE id = ANY($1::bigserial[])
`

func (q *Queries) ListSomeItems(ctx context.Context, ids []int64) ([]Item, error) {
	dbRead := func() (any, time.Duration, error) {
		cacheDuration := time.Duration(time.Millisecond * 0)
		rows, err := q.db.WQuery(ctx, "ListSomeItems", listSomeItems, ids)
		if err != nil {
			return nil, 0, err
		}
		defer rows.Close()
		var items []Item
		for rows.Next() {
			var i Item
			if err := rows.Scan(
				&i.ID,
				&i.Name,
				&i.Description,
				&i.Category,
				&i.Price,
				&i.Thumbnail,
				&i.Metadata,
				&i.CreatedAt,
				&i.UpdatedAt,
			); err != nil {
				return nil, 0, err
			}
			items = append(items, i)
		}
		if err := rows.Err(); err != nil {
			return nil, 0, err
		}
		return items, cacheDuration, nil
	}
	if q.cache == nil {
		items, _, err := dbRead()
		return items.([]Item), err
	}

	var items []Item
	err := q.cache.GetWithTtl(ctx, fmt.Sprintf("ListSomeItems:%+v", ids), &items, dbRead, false, false)
	if err != nil {
		return nil, err
	}

	return items, err
}

const searchItems = `-- name: SearchItems :many
SELECT id, name, description, category, price, thumbnail, metadata, createdat, updatedat FROM Items
WHERE Name LIKE $1
`

func (q *Queries) SearchItems(ctx context.Context, name string) ([]Item, error) {
	dbRead := func() (any, time.Duration, error) {
		cacheDuration := time.Duration(time.Millisecond * 0)
		rows, err := q.db.WQuery(ctx, "SearchItems", searchItems, name)
		if err != nil {
			return nil, 0, err
		}
		defer rows.Close()
		var items []Item
		for rows.Next() {
			var i Item
			if err := rows.Scan(
				&i.ID,
				&i.Name,
				&i.Description,
				&i.Category,
				&i.Price,
				&i.Thumbnail,
				&i.Metadata,
				&i.CreatedAt,
				&i.UpdatedAt,
			); err != nil {
				return nil, 0, err
			}
			items = append(items, i)
		}
		if err := rows.Err(); err != nil {
			return nil, 0, err
		}
		return items, cacheDuration, nil
	}
	if q.cache == nil {
		items, _, err := dbRead()
		return items.([]Item), err
	}

	var items []Item
	err := q.cache.GetWithTtl(ctx, fmt.Sprintf("SearchItems:%+v", name), &items, dbRead, false, false)
	if err != nil {
		return nil, err
	}

	return items, err
}

//// auto generated functions

func (q *Queries) Dump(ctx context.Context, beforeDump ...BeforeDump) ([]byte, error) {
	sql := "SELECT id,name,description,category,price,thumbnail,metadata,createdat,updatedat FROM items ORDER BY id,name,description,category,price,thumbnail,metadata,createdat,updatedat ASC;"
	rows, err := q.db.WQuery(ctx, "Dump", sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Item
	for rows.Next() {
		var v Item
		if err := rows.Scan(&v.ID, &v.Name, &v.Description, &v.Category, &v.Price, &v.Thumbnail, &v.Metadata, &v.CreatedAt, &v.UpdatedAt); err != nil {
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
	sql := "INSERT INTO items (id,name,description,category,price,thumbnail,metadata,createdat,updatedat) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9);"
	rows := make([]Item, 0)
	err := json.Unmarshal(data, &rows)
	if err != nil {
		return err
	}
	for _, row := range rows {
		_, err := q.db.WExec(ctx, "Load", sql, row.ID, row.Name, row.Description, row.Category, row.Price, row.Thumbnail, row.Metadata, row.CreatedAt, row.UpdatedAt)
		if err != nil {
			return err
		}
	}
	return nil
}
