-- +goose Up
-- Create necessary extensions
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Create auction status enum
CREATE TYPE auction_status AS ENUM ('INACTIVE', 'ACTIVE', 'ENDED', 'CANCELLED');

-- Create users table
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_name TEXT UNIQUE NOT NULL,
    email TEXT UNIQUE NOT NULL,
    hashed_password TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create users indexes
CREATE INDEX idx_users_created_at ON users(created_at);
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_user_name ON users(user_name);

-- Create categories table
CREATE TABLE categories (
    id SERIAL PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Create categories indexes
CREATE INDEX idx_categories_name ON categories(name);

-- Create auctions table
CREATE TABLE auctions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    starting_price NUMERIC(10,2) NOT NULL,
    current_price NUMERIC(10,2) NOT NULL,
    start_date TIMESTAMPTZ NOT NULL,
    end_date TIMESTAMPTZ NOT NULL,
    status auction_status NOT NULL DEFAULT 'INACTIVE',
    image TEXT NOT NULL,
    user_id UUID NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT fk_auction_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT check_end_date_after_start CHECK (end_date > start_date),
    CONSTRAINT check_positive_prices CHECK (starting_price > 0 AND current_price >= starting_price)
);

-- Create auctions indexes
CREATE INDEX idx_auctions_created_at ON auctions(created_at);
CREATE INDEX idx_auctions_status ON auctions(status);
CREATE INDEX idx_auctions_user_id ON auctions(user_id);
CREATE INDEX idx_auctions_end_date ON auctions(end_date);
CREATE INDEX idx_auctions_start_date ON auctions(start_date);
CREATE INDEX idx_auctions_current_price ON auctions(current_price);

-- Create auction_categories junction table
CREATE TABLE auction_categories (
    auction_id UUID NOT NULL,
    category_id INT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    PRIMARY KEY (auction_id, category_id),
    FOREIGN KEY (auction_id) REFERENCES auctions(id) ON DELETE CASCADE,
    FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE CASCADE
);

-- Create auction_categories indexes
CREATE INDEX idx_auction_categories_auction_id ON auction_categories(auction_id);
CREATE INDEX idx_auction_categories_category_id ON auction_categories(category_id);

-- Create bids table
CREATE TABLE bids (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    amount NUMERIC(10,2) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    user_id UUID NOT NULL,
    auction_id UUID NOT NULL,
    CONSTRAINT fk_bid_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_bid_auction FOREIGN KEY (auction_id) REFERENCES auctions(id) ON DELETE CASCADE,
    CONSTRAINT check_positive_bid_amount CHECK (amount > 0)
);

-- Create bids indexes
CREATE INDEX idx_bids_created_at ON bids(created_at);
CREATE INDEX idx_bids_user_id ON bids(user_id);
CREATE INDEX idx_bids_auction_id ON bids(auction_id);
CREATE INDEX idx_bids_amount ON bids(amount);
CREATE INDEX idx_bids_auction_amount_desc ON bids(auction_id, amount DESC, created_at DESC);

-- Create trigger function for updated_at timestamps
CREATE OR REPLACE FUNCTION update_updated_at_column() RETURNS TRIGGER AS 'BEGIN NEW.updated_at = NOW(); RETURN NEW; END;' LANGUAGE plpgsql;

-- Create triggers for automatic updated_at updates
CREATE TRIGGER update_users_updated_at 
    BEFORE UPDATE ON users 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_auctions_updated_at 
    BEFORE UPDATE ON auctions 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- +goose Down
-- Drop triggers
DROP TRIGGER IF EXISTS update_users_updated_at ON users;
DROP TRIGGER IF EXISTS update_auctions_updated_at ON auctions;

-- Drop trigger function
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop tables in reverse order (due to foreign key constraints)
DROP TABLE IF EXISTS bids CASCADE;
DROP TABLE IF EXISTS auction_categories CASCADE;
DROP TABLE IF EXISTS auctions CASCADE;
DROP TABLE IF EXISTS categories CASCADE;
DROP TABLE IF EXISTS users CASCADE;

-- Drop custom types
DROP TYPE IF EXISTS auction_status;

-- Drop extensions
DROP EXTENSION IF EXISTS "pgcrypto";