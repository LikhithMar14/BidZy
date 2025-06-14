-- +goose Up
ALTER TABLE auctions ADD COLUMN client_count INT NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE auctions DROP COLUMN client_count;
