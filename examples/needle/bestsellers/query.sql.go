// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v2.1.0-1-g1f618f69-wicked-fork
// source: query.sql

package bestsellers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
)

const listAll = `-- name: ListAll :many
SELECT id, name, firstbstime FROM BestSellers
`

func (q *Queries) ListAll(ctx context.Context) ([]Bestseller, error) {
	rows, err := q.db.WQuery(ctx, "bestsellers.ListAll", listAll)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Bestseller
	for rows.Next() {
		var i *Bestseller = new(Bestseller)
		if err := rows.Scan(&i.ID, &i.Name, &i.Firstbstime); err != nil {
			return nil, err
		}
		items = append(items, *i)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, err
}

//// auto generated functions

func (q *Queries) Dump(ctx context.Context, beforeDump ...BeforeDump) ([]byte, error) {
	sql := "SELECT id,name,firstbstime FROM bestsellers ORDER BY id,name,firstbstime ASC;"
	rows, err := q.db.WQuery(ctx, "bestsellers.Dump", sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []Bestseller
	for rows.Next() {
		var v Bestseller
		if err := rows.Scan(&v.ID, &v.Name, &v.Firstbstime); err != nil {
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
	sql := "INSERT INTO bestsellers (id,name,firstbstime) VALUES ($1,$2,$3);"
	rows := make([]Bestseller, 0)
	err := json.Unmarshal(data, &rows)
	if err != nil {
		return err
	}
	for _, row := range rows {
		_, err := q.db.WExec(ctx, "bestsellers.Load", sql, row.ID, row.Name, row.Firstbstime)
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
