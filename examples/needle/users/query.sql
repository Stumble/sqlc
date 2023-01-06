-- name: GetUserByID :one
-- -- cache : 30s
SELECT * FROM Users
WHERE id = $1 LIMIT 1;

-- name: GetUserByName :one
-- -- cache : 5m
SELECT * FROM Users
WHERE Name = $1 LIMIT 1;

-- name: ListUsers :many
SELECT * FROM users
WHERE id > @after
ORDER BY id
LIMIT @first;

-- name: UpdateNameByID :one
UPDATE users
SET
  Name = $1
WHERE
  ID = $2
RETURNING ID;

-- name: UpdateMetaByID :execrows
UPDATE users
SET
  Metadata = $1
WHERE
  ID = $2;

-- name: ListUserNames :many
SELECT id, name FROM users
WHERE id > @after
ORDER BY id
LIMIT @first;

-- name: CreateUser :one
-- -- invalidate : [GetUserByID, GetUserByName]
INSERT INTO Users (
  Name, Metadata, Thumbnail
) VALUES (
  $1, $2, $3
)
RETURNING *;

-- name: DeleteUser :exec
-- -- invalidate : [GetUserByID, GetUserByName]
DELETE FROM Users
WHERE id = $1;

-- name: UpdateUserGrade :execrows
-- -- invalidate : [GetUserByID]
UPDATE users
  SET metadata = jsonb_set(metadata, '{grade}', @grade::text, true)
WHERE
  Name LIKE @name;

-- name: DeleteBadUsers :execresult
-- -- invalidate : [GetUserByID]
DELETE FROM Users
WHERE NAME LIKE $1;

-- name: Complicated :one
-- -- cache : 1m
-- example of sqlc cannot handle recursive query.
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
  	WHERE n < @n::int
	)
SELECT
	x
FROM fibonacci;
