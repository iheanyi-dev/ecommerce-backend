-- name: CreateStore :exec
INSERT INTO stores (
    id,
    owner_id,
    name,
    slug,
    description,
    image_reference,
    status,
    plan,
    created_at,
    updated_at
)
VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7,
    $8,
    $9,
    $10
);

-- name: DeleteStore :exec
DELETE FROM stores
WHERE id = $1;

-- name: GetStoreByID :one
SELECT
    id,
    owner_id,
    name,
    slug,
    description,
    image_reference,
    status,
    plan,
    product_count,
    created_at,
    updated_at
FROM stores
WHERE id = $1;

-- name: GetStoreByOwnerID :one
SELECT
    id,
    owner_id,
    name,
    slug,
    description,
    image_reference,
    status,
    plan,
    product_count,
    created_at,
    updated_at
FROM stores
WHERE owner_id = $1;

-- name: GetStoreBySlug :one
SELECT
    id,
    owner_id,
    name,
    slug,
    description,
    image_reference,
    status,
    plan,
    product_count,
    created_at,
    updated_at
FROM stores
WHERE slug = $1;

-- name: StoreExistsBySlug :one
SELECT EXISTS (
    SELECT 1
    FROM stores
    WHERE slug = $1
) AS exists;

-- name: ListActiveStores :many
SELECT
    id,
    owner_id,
    name,
    slug,
    description,
    image_reference,
    status,
    plan,
    product_count,
    created_at,
    updated_at
FROM stores
WHERE status = 'active'
  AND (
      $1 = ''
      OR name ILIKE '%' || $1 || '%'
      OR slug ILIKE '%' || $1 || '%'
  )
ORDER BY created_at DESC
LIMIT $2
OFFSET $3;

-- name: CountActiveStores :one
SELECT COUNT(*)
FROM stores
WHERE status = 'active'
  AND (
      $1 = ''
      OR name ILIKE '%' || $1 || '%'
      OR slug ILIKE '%' || $1 || '%'
  );

-- name: UpdateStore :exec
UPDATE stores
SET
    name = $2,
    slug = $3,
    description = $4,
    image_reference = $5,
    status = $6,
    plan = $7,
    updated_at = $8
WHERE id = $1;

-- name: ChangeStorePlan :execrows
UPDATE stores
SET
    name = $2,
    slug = $3,
    description = $4,
    image_reference = $5,
    status = $6,
    plan = $7,
    updated_at = $8
WHERE id = $1
  AND product_count <= CASE $7
      WHEN 'basic' THEN 2
      WHEN 'premium' THEN 5
      ELSE 0
  END;

-- name: ChangeStoreStatus :exec
UPDATE stores
SET
    status = $2,
    updated_at = NOW()
WHERE id = $1;


-- Atomically reserve one product slot without exceeding the Store plan limit.
-- A missing row means either the Store does not exist or its capacity is full.
-- name: IncrementStoreProductCount :one
UPDATE stores
SET
    product_count = product_count + 1,
    updated_at = NOW()
WHERE id = $1
  AND product_count < CASE plan
      WHEN 'basic' THEN 2
      WHEN 'premium' THEN 5
      ELSE 0
  END
RETURNING product_count;

-- Compensate for a failed product creation without allowing a negative count.
-- name: DecrementStoreProductCount :one
UPDATE stores
SET
    product_count = product_count - 1,
    updated_at = NOW()
WHERE id = $1
  AND product_count > 0
RETURNING product_count;

-- Return only active Store IDs from the supplied candidate IDs.
-- name: FilterActiveStoreIDs :many
SELECT id
FROM stores
WHERE status = 'active'
  AND id = ANY($1::uuid[]);
