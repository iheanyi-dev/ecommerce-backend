-- +goose Up

ALTER TABLE products
ADD COLUMN video_storage_reference TEXT;

-- +goose Down

ALTER TABLE products
DROP COLUMN video_storage_reference;
