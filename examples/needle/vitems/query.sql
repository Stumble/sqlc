-- name: GetTopItems :many
select * from v_items
order by
  totalVolume
limit 3;

-- name: Refresh :exec
REFRESH MATERIALIZED VIEW CONCURRENTLY v_items;
