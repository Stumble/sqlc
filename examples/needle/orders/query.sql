-- name: GetOrderByID :one
-- -- cache : 10m
SELECT
  Orders.ID,
  Orders.UserID,
  Orders.ItemID,
  Orders.CreatedAt,
  Users.Name AS UserName,
  Users.Thumbnail AS UserThumbnail,
  Items.Name AS ItemName,
  Items.Description As ItemDesc,
  Items.Price As ItemPrice,
  Items.Thumbnail As ItemThumbnail,
  Items.Metadata As ItemMetadata
FROM
  Orders
  INNER JOIN Items ON Orders.ItemID = Items.ID
  INNER JOIN Users ON Orders.UserID = Users.ID
WHERE
  Orders.IsDeleted = FALSE;

-- name: ListOrdersByUser :many
SELECT * FROM Orders
WHERE
  UserID = @userID AND CreatedAt < @after
ORDER BY CreatedAt DESC
LIMIT @first;

-- name: CreateAuthor :one
INSERT INTO Orders (
  UserID, ItemID, IsDeleted
) VALUES (
  $1, $2, FALSE
)
RETURNING *;

-- name: DeleteOrder :exec
UPDATE Orders
SET
  IsDeleted = TRUE
WHERE
  id = $1;

-- name: ListOrdersByGender :many
-- -- cache : 1m
-- This is just an example for using type annotation for JSON field and 'with clause'.
WITH UsersByGender AS (
  SELECT * FROM Users WHERE Users.Metadata->>'gender' = @gender::text
)
SELECT * FROM Orders
WHERE
  UserId IN (SELECT id FROM UsersByGender) AND Orders.ID > @after
LIMIT @first;
