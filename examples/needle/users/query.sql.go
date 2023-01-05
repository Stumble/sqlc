// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.16.0
// source: query.sql

package users

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

const complicated = `-- name: Complicated :one
WITH RECURSIVE fibonacci(n,x,y) AS (
	SELECT
    	1 AS n ,
  		0 :: int AS x,
    	1 :: int AS y
  	UNION ALL
  	SELECT
    	n + 1 AS n,
  		y AS x,
    	x + y AS y
  	FROM fibonacci
  	WHERE n < $1::int
	)
SELECT
	x
FROM fibonacci
`

// -- cache : 1m
// example of sqlc cannot handle recursive query.
func (q *Queries) Complicated(ctx context.Context, n int32) (*int32, error) {
	// TODO(mustRevalidate, noStore)
	dbRead := func() (any, time.Duration, error) {
		cacheDuration := time.Duration(time.Millisecond * 60000)
		row := q.db.WQueryRow(ctx, "Complicated", complicated,
			n)
		var x int32
		err := row.Scan(&x)
		if err == pgx.ErrNoRows {
			return nil, cacheDuration, nil
		}
		return x, cacheDuration, err
	}
	if q.cache == nil {
		rv, _, err := dbRead()
		return rv.(*int32), err
	}

	var rv *int32
	err := q.cache.GetWithTtl(ctx, fmt.Sprintf("Complicated:%+v", n), &rv, dbRead, false, false)
	if err != nil {
		return nil, err
	}

	return rv, err
}

const createAuthor = `-- name: CreateAuthor :one
INSERT INTO Users (
  Name, Metadata, Thumbnail
) VALUES (
  $1, $2, $3
)
RETURNING id, name, metadata, thumbnail, createdat
`

type CreateAuthorParams struct {
	Name      string
	Metadata  []byte
	Thumbnail string
}

// CacheKey - cache key
func (arg CreateAuthorParams) CacheKey() string {
	prefix := "CreateAuthor:"
	return prefix + fmt.Sprintf("%+v,%+v,%+v", arg.Name, arg.Metadata, arg.Thumbnail)
}

// -- invalidate : [GetUserByID, GetUserByName]
func (q *Queries) CreateAuthor(ctx context.Context, arg CreateAuthorParams, getUserByID *int32, getUserByName *string) (*User, error) {
	// TODO(mustRevalidate, noStore)
	dbRead := func() (any, time.Duration, error) {
		cacheDuration := time.Duration(time.Millisecond * 0)
		row := q.db.WQueryRow(ctx, "CreateAuthor", createAuthor,
			arg.Name, arg.Metadata, arg.Thumbnail)
		var i User
		err := row.Scan(
			&i.ID,
			&i.Name,
			&i.Metadata,
			&i.Thumbnail,
			&i.CreatedAt,
		)
		if err == pgx.ErrNoRows {
			return nil, cacheDuration, nil
		}
		return i, cacheDuration, err
	}
	if q.cache == nil {
		rv, _, err := dbRead()
		return rv.(*User), err
	}

	var rv *User
	err := q.cache.GetWithTtl(ctx, arg.CacheKey(), &rv, dbRead, false, false)
	if err != nil {
		return nil, err
	}

	// invalidate
	invalidateErr := q.db.PostExec(func() error {
		var anyErr error
		if getUserByID != nil {
			err = q.cache.Invalidate(ctx, fmt.Sprintf("GetUserByID:%+v", *getUserByID))
			if err != nil {
				anyErr = err
			}
		}
		if getUserByName != nil {
			err = q.cache.Invalidate(ctx, fmt.Sprintf("GetUserByName:%+v", *getUserByName))
			if err != nil {
				anyErr = err
			}
		}
		return anyErr
	})
	if invalidateErr != nil {
		// invalidateErr is ignored for now.
	}

	return rv, err
}

const deleteAuthor = `-- name: DeleteAuthor :exec
DELETE FROM Users
WHERE id = $1
`

// -- invalidate : [GetUserByID, ListUsers]
func (q *Queries) DeleteAuthor(ctx context.Context, id int32, getUserByID1 *int32, listUsers *ListUsersParams) error {
	_, err := q.db.WExec(ctx, "DeleteAuthor", deleteAuthor, id)
	if err != nil {
		return err
	}

	// invalidate
	invalidateErr := q.db.PostExec(func() error {
		var anyErr error
		if getUserByID1 != nil {
			err = q.cache.Invalidate(ctx, fmt.Sprintf("GetUserByID:%+v", *getUserByID1))
			if err != nil {
				anyErr = err
			}
		}
		if listUsers != nil {
			err = q.cache.Invalidate(ctx, listUsers.CacheKey())
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

const getUserByID = `-- name: GetUserByID :one
SELECT id, name, metadata, thumbnail, createdat FROM Users
WHERE id = $1 LIMIT 1
`

// -- cache : 30s
func (q *Queries) GetUserByID(ctx context.Context, id int32) (*User, error) {
	// TODO(mustRevalidate, noStore)
	dbRead := func() (any, time.Duration, error) {
		cacheDuration := time.Duration(time.Millisecond * 30000)
		row := q.db.WQueryRow(ctx, "GetUserByID", getUserByID,
			id)
		var i User
		err := row.Scan(
			&i.ID,
			&i.Name,
			&i.Metadata,
			&i.Thumbnail,
			&i.CreatedAt,
		)
		if err == pgx.ErrNoRows {
			return nil, cacheDuration, nil
		}
		return i, cacheDuration, err
	}
	if q.cache == nil {
		rv, _, err := dbRead()
		return rv.(*User), err
	}

	var rv *User
	err := q.cache.GetWithTtl(ctx, fmt.Sprintf("GetUserByID:%+v", id), &rv, dbRead, false, false)
	if err != nil {
		return nil, err
	}

	return rv, err
}

const getUserByName = `-- name: GetUserByName :one
SELECT id, name, metadata, thumbnail, createdat FROM Users
WHERE Name = $1 LIMIT 1
`

// -- cache : 5m
func (q *Queries) GetUserByName(ctx context.Context, name string) (*User, error) {
	// TODO(mustRevalidate, noStore)
	dbRead := func() (any, time.Duration, error) {
		cacheDuration := time.Duration(time.Millisecond * 300000)
		row := q.db.WQueryRow(ctx, "GetUserByName", getUserByName,
			name)
		var i User
		err := row.Scan(
			&i.ID,
			&i.Name,
			&i.Metadata,
			&i.Thumbnail,
			&i.CreatedAt,
		)
		if err == pgx.ErrNoRows {
			return nil, cacheDuration, nil
		}
		return i, cacheDuration, err
	}
	if q.cache == nil {
		rv, _, err := dbRead()
		return rv.(*User), err
	}

	var rv *User
	err := q.cache.GetWithTtl(ctx, fmt.Sprintf("GetUserByName:%+v", name), &rv, dbRead, false, false)
	if err != nil {
		return nil, err
	}

	return rv, err
}

const listUserNames = `-- name: ListUserNames :many
SELECT id, name FROM users
WHERE id > $1
ORDER BY id
LIMIT $2
`

type ListUserNamesParams struct {
	After int32
	First int32
}

// CacheKey - cache key
func (arg ListUserNamesParams) CacheKey() string {
	prefix := "ListUserNames:"
	return prefix + fmt.Sprintf("%+v,%+v", arg.After, arg.First)
}

type ListUserNamesRow struct {
	ID   int32
	Name string
}

func (q *Queries) ListUserNames(ctx context.Context, arg ListUserNamesParams) ([]ListUserNamesRow, error) {
	dbRead := func() (any, time.Duration, error) {
		cacheDuration := time.Duration(time.Millisecond * 0)
		rows, err := q.db.WQuery(ctx, "ListUserNames", listUserNames, arg.After, arg.First)
		if err != nil {
			return nil, 0, err
		}
		defer rows.Close()
		var items []ListUserNamesRow
		for rows.Next() {
			var i ListUserNamesRow
			if err := rows.Scan(&i.ID, &i.Name); err != nil {
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
		return items.([]ListUserNamesRow), err
	}

	var items []ListUserNamesRow
	err := q.cache.GetWithTtl(ctx, arg.CacheKey(), &items, dbRead, false, false)
	if err != nil {
		return nil, err
	}

	return items, err
}

const listUsers = `-- name: ListUsers :many
SELECT id, name, metadata, thumbnail, createdat FROM users
WHERE id > $1
ORDER BY id
LIMIT $2
`

type ListUsersParams struct {
	After int32
	First int32
}

// CacheKey - cache key
func (arg ListUsersParams) CacheKey() string {
	prefix := "ListUsers:"
	return prefix + fmt.Sprintf("%+v,%+v", arg.After, arg.First)
}

func (q *Queries) ListUsers(ctx context.Context, arg ListUsersParams) ([]User, error) {
	dbRead := func() (any, time.Duration, error) {
		cacheDuration := time.Duration(time.Millisecond * 0)
		rows, err := q.db.WQuery(ctx, "ListUsers", listUsers, arg.After, arg.First)
		if err != nil {
			return nil, 0, err
		}
		defer rows.Close()
		var items []User
		for rows.Next() {
			var i User
			if err := rows.Scan(
				&i.ID,
				&i.Name,
				&i.Metadata,
				&i.Thumbnail,
				&i.CreatedAt,
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
		return items.([]User), err
	}

	var items []User
	err := q.cache.GetWithTtl(ctx, arg.CacheKey(), &items, dbRead, false, false)
	if err != nil {
		return nil, err
	}

	return items, err
}

//// auto generated functions

func (q *Queries) Dump(ctx context.Context, beforeDump ...BeforeDump) ([]byte, error) {
	sql := "SELECT id,name,metadata,thumbnail,createdat FROM users ORDER BY id,name,metadata,thumbnail,createdat ASC;"
	rows, err := q.db.WQuery(ctx, "Dump", sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []User
	for rows.Next() {
		var v User
		if err := rows.Scan(&v.ID, &v.Name, &v.Metadata, &v.Thumbnail, &v.CreatedAt); err != nil {
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
	sql := "INSERT INTO users (id,name,metadata,thumbnail,createdat) VALUES ($1,$2,$3,$4,$5);"
	rows := make([]User, 0)
	err := json.Unmarshal(data, &rows)
	if err != nil {
		return err
	}
	for _, row := range rows {
		_, err := q.db.WExec(ctx, "Load", sql, row.ID, row.Name, row.Metadata, row.Thumbnail, row.CreatedAt)
		if err != nil {
			return err
		}
	}
	return nil
}
