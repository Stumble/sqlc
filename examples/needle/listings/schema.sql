CREATE TABLE IF NOT EXISTS Listings (
   ID        bigserial GENERATED  ALWAYS AS IDENTITY,
   ItemID    INT       references Items(ID) NOT NULL,
   MakerID   INT       references Users(ID) NOT NULL,
   Price     BIGINT    NOT NULL,
   CreatedAt TIMESTAMP NOT NULL DEFAULT NOW(),
   PRIMARY KEY(ID)
);

CREATE INDEX IF NOT EXISTS listings_item_id_id_idx
    ON Listings (ItemID, ID);
