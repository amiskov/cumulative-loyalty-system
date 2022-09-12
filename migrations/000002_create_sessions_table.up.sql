CREATE TABLE IF NOT EXISTS sessions(
  session_id VARCHAR(128) PRIMARY KEY,
  user_id integer references users(id) on delete cascade
);