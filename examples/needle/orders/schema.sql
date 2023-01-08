CREATE TABLE IF NOT EXISTS Orders (
   ID        INT GENERATED ALWAYS AS IDENTITY,
   UserID    INT references Users(ID) NOT NULL,
   ItemID    INT references Items(ID) NOT NULL,
   Price     BIGINT NOT NULL,
   CreatedAt TIMESTAMP NOT NULL DEFAULT NOW(),
   IsDeleted BOOLEAN NOT NULL,
   PRIMARY KEY(ID)
);

CREATE INDEX orders_item_id_idx ON orders (ItemID);
