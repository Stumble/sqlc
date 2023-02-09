// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v2.0.2-1-g38858e5b-dirty-wicked-fork
// source: query.sql

package vitems

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
)

const getTopItems = `-- name: GetTopItems :many
select id, name, description, category, price, thumbnail, qrcode, metadata, createdat, updatedat, totalvolume, last30dvolume, floorprice from v_items
order by
  totalVolume
limit 3
`

func (q *Queries) GetTopItems(ctx context.Context) ([]VItem, error) {
	rows, err := q.db.WQuery(ctx, "vitems.GetTopItems", getTopItems)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []VItem
	for rows.Next() {
		var i *VItem = new(VItem)
		if err := rows.Scan(
			&i.ID,
			&i.Name,
			&i.Description,
			&i.Category,
			&i.Price,
			&i.Thumbnail,
			&i.Qrcode,
			&i.Metadata,
			&i.CreatedAt,
			&i.UpdatedAt,
			&i.Totalvolume,
			&i.Last30dvolume,
			&i.Floorprice,
		); err != nil {
			return nil, err
		}
		items = append(items, *i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, err
}

const refresh = `-- name: Refresh :exec
REFRESH MATERIALIZED VIEW CONCURRENTLY v_items
`

func (q *Queries) Refresh(ctx context.Context) error {
	_, err := q.db.WExec(ctx, "vitems.Refresh", refresh)
	if err != nil {
		return err
	}

	return nil
}

//// auto generated functions

func (q *Queries) Dump(ctx context.Context, beforeDump ...BeforeDump) ([]byte, error) {
	sql := "SELECT id,name,description,category,price,thumbnail,qrcode,metadata,createdat,updatedat,totalvolume,last30dvolume,floorprice FROM v_items ORDER BY id,name,description,thumbnail,qrcode,createdat,updatedat,totalvolume,last30dvolume,floorprice ASC;"
	rows, err := q.db.WQuery(ctx, "vitems.Dump", sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []VItem
	for rows.Next() {
		var v VItem
		if err := rows.Scan(&v.ID, &v.Name, &v.Description, &v.Category, &v.Price, &v.Thumbnail, &v.Qrcode, &v.Metadata, &v.CreatedAt, &v.UpdatedAt, &v.Totalvolume, &v.Last30dvolume, &v.Floorprice); err != nil {
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
	sql := "INSERT INTO v_items (id,name,description,category,price,thumbnail,qrcode,metadata,createdat,updatedat,totalvolume,last30dvolume,floorprice) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13);"
	rows := make([]VItem, 0)
	err := json.Unmarshal(data, &rows)
	if err != nil {
		return err
	}
	for _, row := range rows {
		_, err := q.db.WExec(ctx, "vitems.Load", sql, row.ID, row.Name, row.Description, row.Category, row.Price, row.Thumbnail, row.Qrcode, row.Metadata, row.CreatedAt, row.UpdatedAt, row.Totalvolume, row.Last30dvolume, row.Floorprice)
		if err != nil {
			return err
		}
	}
	return nil
}

func hashIfLong(v string) string {
	if len(v) > 64 {
		hash := sha256.Sum256([]byte(v))
		return "h(" + hex.EncodeToString(hash[:]) + ")"
	}
	return v
}

// eliminate unused error
var _ = log.Logger
var _ = fmt.Sprintf("")
var _ = time.Now()
var _ = json.RawMessage{}
var _ = sha256.Sum256(nil)
var _ = hex.EncodeToString(nil)
