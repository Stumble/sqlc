-- name: GetTopItems :many
select * from v_items
order by
  totalVolume
limit 3;
