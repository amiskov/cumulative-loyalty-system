CREATE TABLE IF NOT EXISTS users(
  id SERIAL PRIMARY KEY,
  login VARCHAR(128) NOT NULL UNIQUE,
  password BYTEA NOT NULL,
  balance NUMERIC(8, 2) NOT NULL DEFAULT 0,
  withdrawn NUMERIC(8, 2) NOT NULL DEFAULT 0
);
