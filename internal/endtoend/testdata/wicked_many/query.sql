-- name: ListBooks :many
-- -- timeout: 1s
SELECT id, value FROM books WHERE id >= @min_id ORDER BY id;

-- name: ListCachedBooks :many
-- -- timeout: 1s
-- -- cache: 5s
SELECT id, value FROM books ORDER BY id;

-- name: ListPrimaryBooks :many
-- -- timeout: 1s
-- -- allow_replica: false
-- -- count_intent: false
SELECT id, value FROM books ORDER BY id;

-- name: ListIDs :many
-- -- timeout: 1s
SELECT id FROM books ORDER BY id;

-- name: PutBooks :many
-- -- timeout: 1s
-- -- invalidate: [ListCachedBooks]
INSERT INTO books (id, value) VALUES (@id, @value)
ON CONFLICT (id) DO UPDATE SET value = EXCLUDED.value
RETURNING id, value;
