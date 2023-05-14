CREATE TABLE IF NOT EXISTS hashes (
  hash BLOB NOT NULL PRIMARY KEY,
  unix_secs INT NOT NULL
);

CREATE INDEX IF NOT EXISTS unix_secs_idx ON hashes (unix_secs);
