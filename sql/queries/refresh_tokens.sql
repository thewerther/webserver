-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (token, created_at, updated_at, user_id, expires_at, revoked_at)
VALUES (
  $1
  NOW(),
  NOW(),
  $2,
  $3,
  $4
)
RETURNING *;

-- name: GetToken :one
SELECT * FROM refresh_tokens
WHERE token = $1;

-- name: GetUserFromRefreshToken :one
SELECT U.*
FROM users U
INNER JOIN refresh_tokens R ON R.user_id = U.user_id
WHERE token = $1;
