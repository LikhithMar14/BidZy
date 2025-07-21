-- +goose Up
CREATE INDEX IF NOT EXISTS idx_auction_email_logs_auction_id ON auction_email_logs(auction_id);
CREATE INDEX IF NOT EXISTS idx_auction_email_logs_email_sent_at ON auction_email_logs(email_sent_at);

-- +goose Down
DROP INDEX IF EXISTS idx_auction_email_logs_auction_id;
DROP INDEX IF EXISTS idx_auction_email_logs_email_sent_at;
