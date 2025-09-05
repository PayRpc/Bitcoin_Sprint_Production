-- Bitcoin Sprint: Database Initialization Script
-- Place this file in the db/ folder for active use in CI/CD and local dev

-- Example table: blocks
CREATE TABLE IF NOT EXISTS blocks (
    id SERIAL PRIMARY KEY,
    hash VARCHAR(64) NOT NULL,
    height INTEGER NOT NULL,
    timestamp TIMESTAMP NOT NULL
);

-- Example table: transactions
CREATE TABLE IF NOT EXISTS transactions (
    id SERIAL PRIMARY KEY,
    block_id INTEGER REFERENCES blocks(id),
    tx_hash VARCHAR(64) NOT NULL,
    amount NUMERIC(18,8) NOT NULL,
    sender VARCHAR(64),
    receiver VARCHAR(64),
    timestamp TIMESTAMP NOT NULL
);

-- Add more tables and indexes as needed for your application
