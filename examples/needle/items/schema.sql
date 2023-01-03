CREATE TYPE ItemCategory AS ENUM (
    'ALCOHOL ',
    'DRUG',
    'DRINK',
    'FRUIT',
    'VEGETABLE'
);

CREATE TABLE IF NOT EXISTS Items (
   ID           BIGSERIAL GENERATED ALWAYS AS IDENTITY,
   Name         VARCHAR(255)        NOT NULL,
   Description  VARCHAR(255)        NOT NULL,
   Category     ItemCategory        NOT NULL,
   Price        DECIMAL(10,2)       NOT NULL,
   Thumbnail    TEXT                NOT NULL,
   Metadata     JSON,
   CreatedAt    TIMESTAMP           NOT NULL DEFAULT NOW(),
   UpdatedAt    TIMESTAMP           NOT NULL DEFAULT NOW(),
   PRIMARY KEY(ID)
) PARTITION BY RANGE (ID);

CREATE TABLE IF NOT EXISTS items_id_le_1000
PARTITION OF Items FOR VALUES
FROM (0) TO (1000);

-- local index on (Name)
CREATE INDEX IF NOT EXISTS items_id_le_1000_name_idx
ON items_id_le_1000 (Name);

-- global index on created_at
CREATE INDEX IF NOT EXISTS items_created_at_idx
    ON Items (CreatedAt);
