-- +goose Up
CREATE TABLE auction_email_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    auction_id UUID NOT NULL REFERENCES auctions(id) ON DELETE CASCADE,
    email_sent_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    UNIQUE(auction_id)
);

-- +goose Down
DROP TABLE IF EXISTS auction_email_logs;
