-- name: ListOrdersOfItem :many
-- -- cache : 10s
select * FROM Listings
WHERE
  ItemID = @itemID AND ID > @after
LIMIT @first;

-- name: AddListing :one
INSERT INTO Listings (
  ItemID, MakerID, Price
) VALUES (
  $1, $2, $3
) RETURNING ID;
