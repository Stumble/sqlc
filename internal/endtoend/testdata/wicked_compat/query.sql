-- name: GetRecord :one
-- -- timeout: 1s
-- -- cache: 1m
-- -- sql_timeout: 250ms
SELECT * FROM records WHERE id = @id;

-- name: ListRecords :many
-- -- timeout: 1s
-- -- cache: 1m
-- -- sql_timeout: 250ms
SELECT * FROM records ORDER BY id;

-- name: CreateRecord :exec
-- -- timeout: 1s
-- -- invalidate: [GetRecord]
INSERT INTO records (id, external_id, optional_uuid, name, data)
VALUES (@id, @external_id, @optional_uuid, @name, @data)
RETURNING id, name;
