-- name: GetBook :one
SELECT * FROM books WHERE id = @id;
