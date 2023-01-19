// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.16.0-70-g644434f9-wicked-fork

package vitems

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stumble/wpgx"
)

type WGConn interface {
	WQuery(
		ctx context.Context, name string, unprepared string, args ...interface{}) (pgx.Rows, error)
	WQueryRow(
		ctx context.Context, name string, unprepared string, args ...interface{}) pgx.Row
	WExec(
		ctx context.Context, name string, unprepared string, args ...interface{}) (pgconn.CommandTag, error)

	PostExec(f wpgx.PostExecFunc) error
}

type ReadWithTtlFunc = func() (any, time.Duration, error)

// BeforeDump allows you to edit result before dump.
type BeforeDump func(m *VItem)

type Cache interface {
	GetWithTtl(
		ctx context.Context, key string, target any,
		readWithTtl ReadWithTtlFunc, noCache bool, noStore bool) error
	Set(ctx context.Context, key string, val any, ttl time.Duration) error
	Invalidate(ctx context.Context, key string) error
}

func New(db WGConn, cache Cache) *Queries {
	return &Queries{db: db, cache: cache}
}

type Queries struct {
	db    WGConn
	cache Cache
}

func (q *Queries) WithTx(tx *wpgx.WTx) *Queries {
	return &Queries{
		db:    tx,
		cache: q.cache,
	}
}

func (q *Queries) WithCache(cache Cache) *Queries {
	return &Queries{
		db:    q.db,
		cache: cache,
	}
}

var Schema = `
CREATE MATERIALIZED VIEW IF NOT EXISTS v_items AS
  SELECT
    items.ID,
    items.Name,
    items.Description,
    items.Category,
    items.Price,
    items.Thumbnail,
    items.QRCode,
    items.Metadata,
    items.CreatedAt,
    items.UpdatedAt,
    sum(orders.Price) AS totalVolume,
    sum(
      CASE WHEN (orders.CreatedAt > now() - interval '30 day') THEN orders.Price ELSE 0 END
    ) AS last30dVolume,
    max(listings.Price)::bigint AS floorPrice
  FROM
    items
    LEFT JOIN Orders ON items.ID = orders.ItemID
    LEFT JOIN Listings ON items.ID = listings.ItemID
  GROUP BY
      items.ID;

CREATE UNIQUE INDEX v_items_id_unique_idx
  ON v_items (ID);

CREATE UNIQUE INDEX v_items_floor_price_idx
  ON v_items (floorPrice);

CREATE UNIQUE INDEX v_items_total_volume_idx
  ON v_items (totalVolume);

CREATE UNIQUE INDEX v_items_total_last_30d_volume_idx
  ON v_items (last30dVolume);
`
