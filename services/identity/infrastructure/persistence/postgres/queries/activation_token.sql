-- name: CreateActivationToken :exec
INSERT INTO activation_tokens (
    id,
    user_id,
    token_hash,
    expires_at,
    consumed_at,
    created_at
)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: FindActivationTokenByHash :one
SELECT
    id,
    user_id,
    token_hash,
    expires_at,
    consumed_at,
    created_at
FROM activation_tokens
WHERE token_hash = $1
LIMIT 1;

-- name: DeleteActivationTokensByUserID :exec
DELETE FROM activation_tokens
WHERE user_id = $1;

-- name: ConsumeActivationToken :execrows
UPDATE activation_tokens
SET consumed_at = $2
WHERE token_hash = $1
  AND consumed_at IS NULL
  AND expires_at > $2;

-- name: ActivatePendingUser :execrows
UPDATE users
SET
    status = 'active',
    updated_at = $2
WHERE id = $1
  AND status = 'pending_verification';
