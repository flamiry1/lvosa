-- name: AddRequest :one
INSERT INTO generation_requests (user_id, level, topic, difficulty, focus, size) VALUES ($1,$2,$3,$4,$5,$6) RETURNING id;
-- name: GetRequest :one
SELECT * FROM generation_requests WHERE id = $1 AND user_id = $2;
