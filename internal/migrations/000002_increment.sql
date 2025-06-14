-- +goose Up
ALTER TABLE auctions ADD COLUMN increment NUMERIC(10,2) NOT NULL DEFAULT 100;

-- +goose Down
ALTER TABLE auctions DROP COLUMN increment;
