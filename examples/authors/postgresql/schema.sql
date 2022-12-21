CREATE TABLE authors (
          id   BIGSERIAL PRIMARY KEY,
          name text      NOT NULL,
          created_at     timestamp NOT NULL,
          bio  text
);
