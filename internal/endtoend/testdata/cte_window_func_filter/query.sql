-- name: ListItemsByJobIDsWithPerJobLimit :many
-- -- timeout: 2s
WITH ranked AS (
    SELECT *,
           COALESCE(published_at, created_at) AS sort_time,
           ROW_NUMBER() OVER (PARTITION BY job_id ORDER BY COALESCE(published_at, created_at) DESC, id DESC) AS rn
    FROM items
    WHERE job_id = ANY(@job_ids::bigint[])
)
SELECT id, job_id, title, published_at, created_at, sort_time
FROM ranked
WHERE rn <= @limit_per_job
ORDER BY sort_time DESC, id DESC;
