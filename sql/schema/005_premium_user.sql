-- +goose Up
ALTER TABLE users
ADD COLUMN is_premium boolean NOT NULL
DEFAULT false;

-- +goose Down
ALTER TABLE users
DROP COLUMN is_premium;
