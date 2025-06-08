-- +goose Up
-- +goose StatementBegin

-- Enable UUID extension for UUID generation
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create custom types for enums
CREATE TYPE auction_status AS ENUM ('created', 'started', 'ended');
CREATE TYPE auction_category AS ENUM ('electronics', 'fashion', 'art', 'collectibles', 'other');

-- Users table - foundational table
CREATE TABLE users (
    user_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username VARCHAR(50) NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL UNIQUE,
    hashed_password VARCHAR(255) NOT NULL,
    image_url TEXT,
    winnings INTEGER NOT NULL DEFAULT 0,
    prev_bids INTEGER NOT NULL DEFAULT 0,
    prev_auctions INTEGER NOT NULL DEFAULT 0,
    total_auctions INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Auctions table - main auction entity
CREATE TABLE auctions (
    auction_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    starting_price DECIMAL(12,2) NOT NULL CHECK (starting_price > 0),
    max_bid_price DECIMAL(12,2) NOT NULL DEFAULT 0,
    minimum_increment DECIMAL(12,2) NOT NULL CHECK (minimum_increment > 0),
    status auction_status NOT NULL DEFAULT 'created',
    winner_id UUID,
    starting_at TIMESTAMPTZ NOT NULL,
    ending_at TIMESTAMPTZ NOT NULL CHECK (ending_at > starting_at),
    bid_validity_in_seconds INTEGER NOT NULL DEFAULT 300 CHECK (bid_validity_in_seconds > 0),
    image_urls TEXT[] NOT NULL DEFAULT '{}',
    creator_id UUID NOT NULL,
    category auction_category NOT NULL DEFAULT 'other',
    bid_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Foreign key constraints
    CONSTRAINT fk_auction_creator FOREIGN KEY (creator_id) REFERENCES users(user_id) ON DELETE CASCADE,
    CONSTRAINT fk_auction_winner FOREIGN KEY (winner_id) REFERENCES users(user_id) ON DELETE SET NULL
);

-- Bids table - high-frequency insert table
CREATE TABLE bids (
    bid_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    auction_id UUID NOT NULL,
    bidder_id UUID NOT NULL,
    price DECIMAL(12,2) NOT NULL CHECK (price > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Foreign key constraints
    CONSTRAINT fk_bid_auction FOREIGN KEY (auction_id) REFERENCES auctions(auction_id) ON DELETE CASCADE,
    CONSTRAINT fk_bid_bidder FOREIGN KEY (bidder_id) REFERENCES users(user_id) ON DELETE CASCADE
);

-- Performance indexes for Users table
CREATE INDEX idx_users_username ON users USING btree (username);
CREATE INDEX idx_users_email ON users USING btree (email);
CREATE INDEX idx_users_created_at ON users USING btree (created_at);

-- Performance indexes for Auctions table
CREATE INDEX idx_auctions_creator_id ON auctions USING btree (creator_id);
CREATE INDEX idx_auctions_status ON auctions USING btree (status);
CREATE INDEX idx_auctions_category ON auctions USING btree (category);
CREATE INDEX idx_auctions_starting_at ON auctions USING btree (starting_at);
CREATE INDEX idx_auctions_ending_at ON auctions USING btree (ending_at);
CREATE INDEX idx_auctions_status_ending_at ON auctions USING btree (status, ending_at);
CREATE INDEX idx_auctions_category_status ON auctions USING btree (category, status);
CREATE INDEX idx_auctions_max_bid_price ON auctions USING btree (max_bid_price DESC);

-- Performance indexes for Bids table (critical for high-frequency operations)
CREATE INDEX idx_bids_auction_id ON bids USING btree (auction_id);
CREATE INDEX idx_bids_bidder_id ON bids USING btree (bidder_id);
CREATE INDEX idx_bids_auction_price ON bids USING btree (auction_id, price DESC);
CREATE INDEX idx_bids_auction_created_at ON bids USING btree (auction_id, created_at DESC);
CREATE INDEX idx_bids_created_at ON bids USING btree (created_at);

-- Composite indexes for complex queries
CREATE INDEX idx_auctions_active_by_category ON auctions 
    USING btree (category, status, ending_at) 
    WHERE status IN ('created', 'started');

CREATE INDEX idx_bids_latest_by_auction ON bids 
    USING btree (auction_id, created_at DESC, price DESC);

-- Partial indexes for performance optimization
CREATE INDEX idx_auctions_active ON auctions 
    USING btree (ending_at, status) 
    WHERE status IN ('created', 'started');

CREATE INDEX idx_auctions_ended ON auctions 
    USING btree (ending_at DESC) 
    WHERE status = 'ended';

-- Function to update timestamps
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Triggers for automatic timestamp updates
CREATE TRIGGER update_users_updated_at 
    BEFORE UPDATE ON users 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_auctions_updated_at 
    BEFORE UPDATE ON auctions 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

-- Function to update auction bid statistics
CREATE OR REPLACE FUNCTION update_auction_bid_stats()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        -- Update auction with new bid info
        UPDATE auctions 
        SET 
            max_bid_price = GREATEST(max_bid_price, NEW.price),
            bid_count = bid_count + 1,
            updated_at = NOW()
        WHERE auction_id = NEW.auction_id;
        
        -- Update bidder statistics
        UPDATE users 
        SET 
            prev_bids = prev_bids + 1,
            updated_at = NOW()
        WHERE user_id = NEW.bidder_id;
        
        RETURN NEW;
    END IF;
    
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- Trigger to automatically update auction stats when bid is placed
CREATE TRIGGER update_auction_stats_on_bid
    AFTER INSERT ON bids
    FOR EACH ROW
    EXECUTE FUNCTION update_auction_bid_stats();

-- Function to handle auction completion
CREATE OR REPLACE FUNCTION finalize_auction()
RETURNS TRIGGER AS $$
DECLARE
    highest_bidder_id UUID;
BEGIN
    -- Only process when status changes to 'ended'
    IF OLD.status != 'ended' AND NEW.status = 'ended' THEN
        -- Find the highest bidder
        SELECT bidder_id INTO highest_bidder_id
        FROM bids 
        WHERE auction_id = NEW.auction_id 
        ORDER BY price DESC, created_at ASC 
        LIMIT 1;
        
        -- Update auction with winner
        IF highest_bidder_id IS NOT NULL THEN
            NEW.winner_id := highest_bidder_id;
            
            -- Update winner's statistics
            UPDATE users 
            SET 
                winnings = winnings + 1,
                updated_at = NOW()
            WHERE user_id = highest_bidder_id;
        END IF;
        
        -- Update creator's total auctions
        UPDATE users 
        SET 
            total_auctions = total_auctions + 1,
            updated_at = NOW()
        WHERE user_id = NEW.creator_id;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to handle auction finalization
CREATE TRIGGER finalize_auction_on_end
    BEFORE UPDATE ON auctions
    FOR EACH ROW
    EXECUTE FUNCTION finalize_auction();

-- Security: Row Level Security policies
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE auctions ENABLE ROW LEVEL SECURITY;
ALTER TABLE bids ENABLE ROW LEVEL SECURITY;

-- Create roles for different access levels
DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'auction_read') THEN
        CREATE ROLE auction_read;
    END IF;
    
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'auction_write') THEN
        CREATE ROLE auction_write;
    END IF;
    
    IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'auction_admin') THEN
        CREATE ROLE auction_admin;
    END IF;
END
$$;

-- Grant permissions
GRANT SELECT ON ALL TABLES IN SCHEMA public TO auction_read;
GRANT SELECT, INSERT, UPDATE ON ALL TABLES IN SCHEMA public TO auction_write;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO auction_admin;
GRANT USAGE ON ALL SEQUENCES IN SCHEMA public TO auction_write, auction_admin;

-- Optimize table storage and performance
ALTER TABLE bids SET (fillfactor = 90);
ALTER TABLE auctions SET (fillfactor = 95);
ALTER TABLE users SET (fillfactor = 95);

-- Enable parallel query execution for large scans
ALTER TABLE auctions SET (parallel_workers = 4);
ALTER TABLE bids SET (parallel_workers = 6);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Drop triggers
DROP TRIGGER IF EXISTS finalize_auction_on_end ON auctions;
DROP TRIGGER IF EXISTS update_auction_stats_on_bid ON bids;
DROP TRIGGER IF EXISTS update_auctions_updated_at ON auctions;
DROP TRIGGER IF EXISTS update_users_updated_at ON users;

-- Drop functions
DROP FUNCTION IF EXISTS finalize_auction();
DROP FUNCTION IF EXISTS update_auction_bid_stats();
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop indexes
DROP INDEX IF EXISTS idx_bids_latest_by_auction;
DROP INDEX IF EXISTS idx_auctions_active_by_category;
DROP INDEX IF EXISTS idx_auctions_ended;
DROP INDEX IF EXISTS idx_auctions_active;
DROP INDEX IF EXISTS idx_bids_created_at;
DROP INDEX IF EXISTS idx_bids_auction_created_at;
DROP INDEX IF EXISTS idx_bids_auction_price;
DROP INDEX IF EXISTS idx_bids_bidder_id;
DROP INDEX IF EXISTS idx_bids_auction_id;
DROP INDEX IF EXISTS idx_auctions_max_bid_price;
DROP INDEX IF EXISTS idx_auctions_category_status;
DROP INDEX IF EXISTS idx_auctions_status_ending_at;
DROP INDEX IF EXISTS idx_auctions_ending_at;
DROP INDEX IF EXISTS idx_auctions_starting_at;
DROP INDEX IF EXISTS idx_auctions_category;
DROP INDEX IF EXISTS idx_auctions_status;
DROP INDEX IF EXISTS idx_auctions_creator_id;
DROP INDEX IF EXISTS idx_users_created_at;
DROP INDEX IF EXISTS idx_users_email;
DROP INDEX IF EXISTS idx_users_username;

-- Drop tables
DROP TABLE IF EXISTS bids;
DROP TABLE IF EXISTS auctions;
DROP TABLE IF EXISTS users;

-- Drop custom types
DROP TYPE IF EXISTS auction_category;
DROP TYPE IF EXISTS auction_status;

-- Drop roles
DROP ROLE IF EXISTS auction_admin;
DROP ROLE IF EXISTS auction_write;
DROP ROLE IF EXISTS auction_read;

-- +goose StatementEnd