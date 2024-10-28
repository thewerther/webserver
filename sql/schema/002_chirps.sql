-- +goose Up
CREATE TABLE chirps (
  id uuid PRIMARY KEY,
  body TEXT NOT NULL,
  created_at TIMESTAMP NOT NULL,
  updated_at TIMESTAMP NOT NULL,
  user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE chirps;
