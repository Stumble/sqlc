-- name: GetItemByID :one
-- -- cache : 5m
SELECT * FROM Items
WHERE id = $1 LIMIT 1;

-- name: SearchItems :many
SELECT * FROM Items
WHERE Name LIKE $1;

-- name: ListItems :many
SELECT * FROM Items
WHERE id > @after
ORDER BY id
LIMIT @first;

-- name: ListSomeItems :many
SELECT * FROM Items
WHERE id = ANY(@ids::bigserial[]);

-- name: CreateItems :one
INSERT INTO Items (
  Name, Description, Category, Price, Thumbnail, Metadata
) VALUES (
  $1, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: DeleteItem :exec
-- -- invalidate : [GetItemByID, ListItems]
DELETE FROM Items
WHERE id = $1;
