-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS hashes (
  hash BLOB NOT NULL PRIMARY KEY,
  unix_secs INT NOT NULL
);

CREATE INDEX IF NOT EXISTS unix_secs_idx ON hashes (unix_secs);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS unix_secs_idx;
DROP TABLE IF EXISTS hashes;
-- +goose StatementEnd
