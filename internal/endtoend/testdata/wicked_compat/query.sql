-- name: GetRecord :one
-- -- timeout: 1s
-- -- cache: 1m
SELECT * FROM records WHERE id = @id;

-- name: CreateRecord :exec
-- -- timeout: 1s
-- -- invalidate: [GetRecord]
INSERT INTO records (id, external_id, optional_uuid, name, data)
VALUES (@id, @external_id, @optional_uuid, @name, @data)
RETURNING id, name;
