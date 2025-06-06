-- +goose Up
-- +goose StatementBegin

CREATE TYPE auction_status AS ENUM ('INACTIVE', 'ACTIVE', 'ENDED', 'CANCELLED');

CREATE TABLE users (
    id VARCHAR(255) PRIMARY KEY,
    user_name VARCHAR(255) NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL UNIQUE,
    hashed_password VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE auctions (
    id VARCHAR(255) PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    title VARCHAR(500) NOT NULL,
    description TEXT NOT NULL,
    starting_price INTEGER NOT NULL,
    current_price INTEGER NOT NULL,
    start_date TIMESTAMPTZ NOT NULL,
    end_date TIMESTAMPTZ NOT NULL,
    status auction_status NOT NULL DEFAULT 'INACTIVE',
    image VARCHAR(1000) NOT NULL,
    categories VARCHAR(255) NOT NULL,
    user_id VARCHAR(255) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE bids (
    id VARCHAR(255) PRIMARY KEY DEFAULT gen_random_uuid()::TEXT,
    amount DECIMAL(12,2) NOT NULL,
    user_id VARCHAR(255) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    auction_id VARCHAR(255) NOT NULL REFERENCES auctions(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_username ON users(user_name);
CREATE INDEX idx_users_created_at ON users(created_at);

CREATE INDEX idx_auctions_user_id ON auctions(user_id);
CREATE INDEX idx_auctions_status ON auctions(status);
CREATE INDEX idx_auctions_end_date ON auctions(end_date);
CREATE INDEX idx_auctions_start_date ON auctions(start_date);
CREATE INDEX idx_auctions_categories ON auctions(categories);
CREATE INDEX idx_auctions_status_end_date ON auctions(status, end_date);
CREATE INDEX idx_auctions_current_price ON auctions(current_price);
CREATE INDEX idx_auctions_created_at ON auctions(created_at DESC);

CREATE INDEX idx_bids_auction_id ON bids(auction_id);
CREATE INDEX idx_bids_user_id ON bids(user_id);
CREATE INDEX idx_bids_amount ON bids(amount DESC);
CREATE INDEX idx_bids_created_at ON bids(created_at DESC);
CREATE INDEX idx_bids_auction_created ON bids(auction_id, created_at DESC);
CREATE INDEX idx_bids_auction_amount ON bids(auction_id, amount DESC);

-- Constraints
ALTER TABLE auctions ADD CONSTRAINT chk_auctions_price_positive 
    CHECK (starting_price > 0 AND current_price > 0);
ALTER TABLE auctions ADD CONSTRAINT chk_auctions_dates 
    CHECK (end_date > start_date);
ALTER TABLE bids ADD CONSTRAINT chk_bids_amount_positive 
    CHECK (amount > 0);

-- Function and triggers
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_users_updated_at 
    BEFORE UPDATE ON users 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_auctions_updated_at 
    BEFORE UPDATE ON auctions 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Views
CREATE VIEW active_auctions AS
SELECT 
    a.*,
    u.user_name as seller_name,
    u.email as seller_email,
    (SELECT COUNT(*) FROM bids b WHERE b.auction_id = a.id) as bid_count,
    (SELECT MAX(amount) FROM bids b WHERE b.auction_id = a.id) as highest_bid
FROM auctions a
JOIN users u ON a.user_id = u.id
WHERE a.status = 'ACTIVE' AND a.end_date > NOW();

CREATE VIEW user_auction_stats AS
SELECT 
    u.id,
    u.user_name,
    u.email,
    COUNT(DISTINCT a.id) as total_auctions,
    COUNT(DISTINCT CASE WHEN a.status = 'ACTIVE' THEN a.id END) as active_auctions,
    COUNT(DISTINCT b.id) as total_bids,
    COALESCE(SUM(DISTINCT b.amount), 0) as total_bid_amount
FROM users u
LEFT JOIN auctions a ON u.id = a.user_id
LEFT JOIN bids b ON u.id = b.user_id
GROUP BY u.id, u.user_name, u.email;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Drop views first
DROP VIEW IF EXISTS user_auction_stats;
DROP VIEW IF EXISTS active_auctions;

-- Drop triggers
DROP TRIGGER IF EXISTS update_auctions_updated_at ON auctions;
DROP TRIGGER IF EXISTS update_users_updated_at ON users;

-- Drop function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop constraints (they will be dropped with tables, but being explicit)
ALTER TABLE auctions DROP CONSTRAINT IF EXISTS chk_auctions_price_positive;
ALTER TABLE auctions DROP CONSTRAINT IF EXISTS chk_auctions_dates;
ALTER TABLE bids DROP CONSTRAINT IF EXISTS chk_bids_amount_positive;

-- Drop indexes (they will be dropped with tables, but some prefer explicit)
DROP INDEX IF EXISTS idx_bids_auction_amount;
DROP INDEX IF EXISTS idx_bids_auction_created;
DROP INDEX IF EXISTS idx_bids_created_at;
DROP INDEX IF EXISTS idx_bids_amount;
DROP INDEX IF EXISTS idx_bids_user_id;
DROP INDEX IF EXISTS idx_bids_auction_id;

DROP INDEX IF EXISTS idx_auctions_created_at;
DROP INDEX IF EXISTS idx_auctions_current_price;
DROP INDEX IF EXISTS idx_auctions_status_end_date;
DROP INDEX IF EXISTS idx_auctions_categories;
DROP INDEX IF EXISTS idx_auctions_start_date;
DROP INDEX IF EXISTS idx_auctions_end_date;
DROP INDEX IF EXISTS idx_auctions_status;
DROP INDEX IF EXISTS idx_auctions_user_id;

DROP INDEX IF EXISTS idx_users_created_at;
DROP INDEX IF EXISTS idx_users_username;
DROP INDEX IF EXISTS idx_users_email;

-- Drop tables in correct order (child tables first)
DROP TABLE IF EXISTS bids;
DROP TABLE IF EXISTS auctions;
DROP TABLE IF EXISTS users;

-- Drop custom types
DROP TYPE IF EXISTS auction_status;

-- +goose StatementEnd