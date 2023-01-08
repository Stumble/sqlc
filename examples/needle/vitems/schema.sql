CREATE MATERIALIZED VIEW IF NOT EXISTS v_items AS
  SELECT
    items.ID,
    items.Name,
    items.Description,
    items.Category,
    items.Price,
    items.Thumbnail,
    items.QRCode,
    items.Metadata,
    items.CreatedAt,
    items.UpdatedAt,
    sum(orders.Price) AS totalVolume,
    sum(
      CASE WHEN (orders.CreatedAt > now() - interval '30 day') THEN orders.Price ELSE 0 END
    ) AS last30dVolume,
    max(listings.Price)::bigint AS floorPrice
  FROM
    items
    LEFT JOIN Orders ON items.ID = orders.ItemID
    LEFT JOIN Listings ON items.ID = listings.ItemID
  GROUP BY
      items.ID;

CREATE UNIQUE INDEX v_items_id_unique_idx
  ON v_items (ID);

CREATE UNIQUE INDEX v_items_floor_price_idx
  ON v_items (floorPrice);

CREATE UNIQUE INDEX v_items_total_volume_idx
  ON v_items (totalVolume);

CREATE UNIQUE INDEX v_items_total_last_30d_volume_idx
  ON v_items (last30dVolume);
