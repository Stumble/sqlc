-- name: ListBooks :many
-- -- timeout: 1s
-- -- each: true
SELECT id, value FROM books WHERE id >= @visit ORDER BY id;

-- name: ListCachedBooks :many
-- -- timeout: 1s
-- -- cache: 5s
-- -- each: true
SELECT id, value FROM books ORDER BY id;

-- name: ListIDs :many
-- -- timeout: 1s
-- -- each: true
SELECT id FROM books ORDER BY id;

-- name: ListNamedScalar :many
-- -- timeout: 1s
-- -- each: true
SELECT id AS more FROM books ORDER BY id;

-- name: ListPrimaryBooks :many
-- -- timeout: 1s
-- -- allow_replica: false
-- -- count_intent: false
-- -- each: true
SELECT id, value FROM books ORDER BY id;

-- name: ListRange :many
-- -- timeout: 1s
-- -- each: true
SELECT id, value FROM books WHERE id BETWEEN @from_id AND @to_id ORDER BY id;
