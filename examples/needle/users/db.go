// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.16.0-69-g345d6d6a-dirty-wicked-fork

package users

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
type BeforeDump func(m *User)

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
CREATE TABLE IF NOT EXISTS Users (
   ID        INT GENERATED ALWAYS AS IDENTITY,
   Name      VARCHAR(255) NOT NULL,
   Metadata  JSON,
   Thumbnail TEXT NOT NULL,
   CreatedAt TIMESTAMP NOT NULL DEFAULT NOW(),
   PRIMARY KEY(ID)
);

CREATE INDEX IF NOT EXISTS users_created_at_idx
    ON Users (CreatedAt);

CREATE UNIQUE INDEX IF NOT EXISTS users_name_idx
    ON Users ((lower(Name))) INCLUDE (ID);
`
