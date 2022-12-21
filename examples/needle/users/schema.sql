CREATE TABLE IF NOT EXISTS Users (
   ID        INT GENERATED ALWAYS AS IDENTITY,
   Name      VARCHAR(255) NOT NULL,
   Metadata  JSON,
   Thumbnail TEXT NOT NULL,
   CreatedAt TIMESTAMP NOT NULL DEFAULT NOW(),
   PRIMARY KEY(ID)
);

CREATE INDEX IF NOT EXISTS users_created_at_idx
    ON Users (CreatedAt);

CREATE UNIQUE INDEX IF NOT EXISTS users_name_idx
    ON Users ((lower(Name))) INCLUDE (ID);
