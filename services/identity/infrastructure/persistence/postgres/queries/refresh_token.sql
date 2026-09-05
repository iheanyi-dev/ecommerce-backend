-- name: CreateRefreshToken :exec
INSERT INTO refresh_tokens (
    id,
    user_id,
    token_hash,
    expires_at,
    revoked_at,
    created_at
)
VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6
);


-- name: FindRefreshTokenByHash :one
SELECT
    id,
    user_id,
    token_hash,
    expires_at,
    revoked_at,
    created_at
FROM refresh_tokens
WHERE token_hash = $1
LIMIT 1;


-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens
SET revoked_at = $2
WHERE id = $1;