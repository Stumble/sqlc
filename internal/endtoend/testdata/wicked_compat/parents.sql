CREATE TABLE parents (
    id bigint NOT NULL,
    external_id uuid NOT NULL,
    optional_uuid uuid,
    name text NOT NULL,
    data jsonb
);
