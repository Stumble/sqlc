-- name: GetRevenue :one
-- -- timeout: 250ms
-- -- cache: 1m
SELECT * FROM book_revenues WHERE id = @id;
