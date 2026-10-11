-- +goose Up

-- Knowledge metadata belongs to Store. The file bytes are owned by Storage.
CREATE TABLE knowledge (
    id UUID PRIMARY KEY,
    store_id UUID NOT NULL,
    filename TEXT NOT NULL,
    content_type TEXT NOT NULL,
    storage_reference TEXT NOT NULL UNIQUE
);

CREATE INDEX knowledge_store_id_idx ON knowledge (store_id);

-- +goose Down

DROP TABLE IF EXISTS knowledge;
