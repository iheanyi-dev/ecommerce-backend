-- name: GetKnowledgeByStoreID :many
SELECT
    id,
    store_id,
    filename,
    content_type,
    storage_reference
FROM knowledge
WHERE store_id = $1
ORDER BY filename ASC, id ASC;

-- name: SaveKnowledge :exec
INSERT INTO knowledge (
    id,
    store_id,
    filename,
    content_type,
    storage_reference
)
VALUES ($1, $2, $3, $4, $5);

-- name: DeleteKnowledgeByID :exec
DELETE FROM knowledge
WHERE id = $1;
