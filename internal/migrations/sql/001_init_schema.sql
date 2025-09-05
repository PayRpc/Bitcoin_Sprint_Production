-- Initial Bitcoin Sprint database schema
-- +migrate Up

-- Core sprint schema
CREATE SCHEMA IF NOT EXISTS sprint_core;

-- Enterprise features schema
CREATE SCHEMA IF NOT EXISTS sprint_enterprise;

-- Blockchain data schema
CREATE SCHEMA IF NOT EXISTS sprint_chains;

-- Analytics and reporting schema
CREATE SCHEMA IF NOT EXISTS sprint_analytics;

-- Core tables in sprint_core schema
CREATE TABLE IF NOT EXISTS sprint_core.nodes (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    endpoint VARCHAR(512) NOT NULL,
    node_type VARCHAR(50) NOT NULL DEFAULT 'bitcoin',
    status VARCHAR(20) NOT NULL DEFAULT 'unknown',
    last_seen TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(name, endpoint)
);

CREATE TABLE IF NOT EXISTS sprint_core.blocks (
    id SERIAL PRIMARY KEY,
    hash VARCHAR(64) NOT NULL UNIQUE,
    height BIGINT NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE,
    size_bytes INTEGER,
    tx_count INTEGER DEFAULT 0,
    node_id INTEGER REFERENCES sprint_core.nodes(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(hash)
);

CREATE TABLE IF NOT EXISTS sprint_core.transactions (
    id SERIAL PRIMARY KEY,
    txid VARCHAR(64) NOT NULL,
    block_id INTEGER REFERENCES sprint_core.blocks(id),
    size_bytes INTEGER,
    fee_satoshis BIGINT,
    input_count INTEGER DEFAULT 0,
    output_count INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(txid)
);

-- Enterprise tables in sprint_enterprise schema
CREATE TABLE IF NOT EXISTS sprint_enterprise.api_keys (
    id SERIAL PRIMARY KEY,
    key_hash VARCHAR(128) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    tier VARCHAR(20) NOT NULL DEFAULT 'FREE',
    rate_limit INTEGER DEFAULT 100,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE,
    is_active BOOLEAN DEFAULT true
);

CREATE TABLE IF NOT EXISTS sprint_enterprise.rate_limits (
    id SERIAL PRIMARY KEY,
    api_key_id INTEGER REFERENCES sprint_enterprise.api_keys(id),
    endpoint VARCHAR(255) NOT NULL,
    requests_count INTEGER DEFAULT 0,
    window_start TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(api_key_id, endpoint, window_start)
);

CREATE TABLE IF NOT EXISTS sprint_enterprise.audit_log (
    id SERIAL PRIMARY KEY,
    api_key_id INTEGER REFERENCES sprint_enterprise.api_keys(id),
    action VARCHAR(100) NOT NULL,
    resource VARCHAR(255),
    details JSONB,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Chain-specific tables in sprint_chains schema
CREATE TABLE IF NOT EXISTS sprint_chains.bitcoin_blocks (
    id SERIAL PRIMARY KEY,
    hash VARCHAR(64) NOT NULL UNIQUE,
    height BIGINT NOT NULL,
    version INTEGER,
    merkle_root VARCHAR(64),
    timestamp TIMESTAMP WITH TIME ZONE,
    nonce BIGINT,
    difficulty NUMERIC(20,8),
    chain_work VARCHAR(64),
    size_bytes INTEGER,
    weight INTEGER,
    tx_count INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
) PARTITION BY RANGE (height);

CREATE TABLE IF NOT EXISTS sprint_chains.ethereum_blocks (
    id SERIAL PRIMARY KEY,
    hash VARCHAR(66) NOT NULL UNIQUE,
    number BIGINT NOT NULL,
    parent_hash VARCHAR(66),
    timestamp TIMESTAMP WITH TIME ZONE,
    gas_limit BIGINT,
    gas_used BIGINT,
    size_bytes INTEGER,
    tx_count INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
) PARTITION BY RANGE (number);

-- Analytics tables in sprint_analytics schema
CREATE TABLE IF NOT EXISTS sprint_analytics.performance_metrics (
    id SERIAL PRIMARY KEY,
    metric_name VARCHAR(100) NOT NULL,
    metric_value NUMERIC(15,6),
    node_id INTEGER REFERENCES sprint_core.nodes(id),
    endpoint VARCHAR(255),
    recorded_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
) PARTITION BY RANGE (recorded_at);

CREATE TABLE IF NOT EXISTS sprint_analytics.request_stats (
    id SERIAL PRIMARY KEY,
    endpoint VARCHAR(255) NOT NULL,
    method VARCHAR(10) NOT NULL,
    status_code INTEGER,
    response_time_ms INTEGER,
    request_size_bytes INTEGER,
    response_size_bytes INTEGER,
    api_key_id INTEGER REFERENCES sprint_enterprise.api_keys(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
) PARTITION BY RANGE (created_at);

-- Create partitions for current and next month
DO $$
DECLARE
    current_month TEXT;
    next_month TEXT;
BEGIN
    current_month := to_char(CURRENT_DATE, 'YYYY_MM');
    next_month := to_char(CURRENT_DATE + INTERVAL '1 month', 'YYYY_MM');
    
    -- Bitcoin blocks partitions (by height ranges)
    EXECUTE format('CREATE TABLE IF NOT EXISTS sprint_chains.bitcoin_blocks_%s PARTITION OF sprint_chains.bitcoin_blocks FOR VALUES FROM (0) TO (100000)', current_month);
    EXECUTE format('CREATE TABLE IF NOT EXISTS sprint_chains.bitcoin_blocks_%s_high PARTITION OF sprint_chains.bitcoin_blocks FOR VALUES FROM (100000) TO (MAXVALUE)', current_month);
    
    -- Ethereum blocks partitions
    EXECUTE format('CREATE TABLE IF NOT EXISTS sprint_chains.ethereum_blocks_%s PARTITION OF sprint_chains.ethereum_blocks FOR VALUES FROM (0) TO (20000000)', current_month);
    EXECUTE format('CREATE TABLE IF NOT EXISTS sprint_chains.ethereum_blocks_%s_high PARTITION OF sprint_chains.ethereum_blocks FOR VALUES FROM (20000000) TO (MAXVALUE)', current_month);
    
    -- Performance metrics partitions
    EXECUTE format('CREATE TABLE IF NOT EXISTS sprint_analytics.performance_metrics_%s PARTITION OF sprint_analytics.performance_metrics FOR VALUES FROM (%L) TO (%L)', 
                   current_month, 
                   date_trunc('month', CURRENT_DATE), 
                   date_trunc('month', CURRENT_DATE) + INTERVAL '1 month');
    
    EXECUTE format('CREATE TABLE IF NOT EXISTS sprint_analytics.performance_metrics_%s PARTITION OF sprint_analytics.performance_metrics FOR VALUES FROM (%L) TO (%L)', 
                   next_month, 
                   date_trunc('month', CURRENT_DATE) + INTERVAL '1 month', 
                   date_trunc('month', CURRENT_DATE) + INTERVAL '2 months');
    
    -- Request stats partitions
    EXECUTE format('CREATE TABLE IF NOT EXISTS sprint_analytics.request_stats_%s PARTITION OF sprint_analytics.request_stats FOR VALUES FROM (%L) TO (%L)', 
                   current_month, 
                   date_trunc('month', CURRENT_DATE), 
                   date_trunc('month', CURRENT_DATE) + INTERVAL '1 month');
    
    EXECUTE format('CREATE TABLE IF NOT EXISTS sprint_analytics.request_stats_%s PARTITION OF sprint_analytics.request_stats FOR VALUES FROM (%L) TO (%L)', 
                   next_month, 
                   date_trunc('month', CURRENT_DATE) + INTERVAL '1 month', 
                   date_trunc('month', CURRENT_DATE) + INTERVAL '2 months');
END $$;

-- Create indexes for optimal performance
CREATE INDEX IF NOT EXISTS idx_nodes_status ON sprint_core.nodes(status);
CREATE INDEX IF NOT EXISTS idx_nodes_type_status ON sprint_core.nodes(node_type, status);
CREATE INDEX IF NOT EXISTS idx_blocks_height ON sprint_core.blocks(height);
CREATE INDEX IF NOT EXISTS idx_blocks_timestamp ON sprint_core.blocks(timestamp);
CREATE INDEX IF NOT EXISTS idx_transactions_block_id ON sprint_core.transactions(block_id);

CREATE INDEX IF NOT EXISTS idx_api_keys_hash ON sprint_enterprise.api_keys(key_hash);
CREATE INDEX IF NOT EXISTS idx_api_keys_tier_active ON sprint_enterprise.api_keys(tier, is_active);
CREATE INDEX IF NOT EXISTS idx_rate_limits_key_endpoint ON sprint_enterprise.rate_limits(api_key_id, endpoint);
CREATE INDEX IF NOT EXISTS idx_audit_log_created_at ON sprint_enterprise.audit_log(created_at);
CREATE INDEX IF NOT EXISTS idx_audit_log_api_key ON sprint_enterprise.audit_log(api_key_id);

CREATE INDEX IF NOT EXISTS idx_bitcoin_blocks_height ON sprint_chains.bitcoin_blocks(height);
CREATE INDEX IF NOT EXISTS idx_bitcoin_blocks_timestamp ON sprint_chains.bitcoin_blocks(timestamp);
CREATE INDEX IF NOT EXISTS idx_ethereum_blocks_number ON sprint_chains.ethereum_blocks(number);
CREATE INDEX IF NOT EXISTS idx_ethereum_blocks_timestamp ON sprint_chains.ethereum_blocks(timestamp);

CREATE INDEX IF NOT EXISTS idx_performance_metrics_name_recorded ON sprint_analytics.performance_metrics(metric_name, recorded_at);
CREATE INDEX IF NOT EXISTS idx_request_stats_endpoint_created ON sprint_analytics.request_stats(endpoint, created_at);

-- Create functions for automatic timestamp updates
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create triggers for automatic timestamp updates
CREATE TRIGGER update_nodes_updated_at BEFORE UPDATE ON sprint_core.nodes 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Create function for partition management
CREATE OR REPLACE FUNCTION create_monthly_partitions()
RETURNS void AS $$
DECLARE
    next_month TEXT;
    start_date DATE;
    end_date DATE;
BEGIN
    next_month := to_char(CURRENT_DATE + INTERVAL '2 months', 'YYYY_MM');
    start_date := date_trunc('month', CURRENT_DATE) + INTERVAL '2 months';
    end_date := date_trunc('month', CURRENT_DATE) + INTERVAL '3 months';
    
    -- Create next month's partitions
    EXECUTE format('CREATE TABLE IF NOT EXISTS sprint_analytics.performance_metrics_%s PARTITION OF sprint_analytics.performance_metrics FOR VALUES FROM (%L) TO (%L)', 
                   next_month, start_date, end_date);
    
    EXECUTE format('CREATE TABLE IF NOT EXISTS sprint_analytics.request_stats_%s PARTITION OF sprint_analytics.request_stats FOR VALUES FROM (%L) TO (%L)', 
                   next_month, start_date, end_date);
END;
$$ LANGUAGE plpgsql;

-- +migrate Down

-- Drop functions
DROP FUNCTION IF EXISTS create_monthly_partitions();
DROP FUNCTION IF EXISTS update_updated_at_column() CASCADE;

-- Drop schemas (CASCADE will drop all tables)
DROP SCHEMA IF EXISTS sprint_analytics CASCADE;
DROP SCHEMA IF EXISTS sprint_chains CASCADE;
DROP SCHEMA IF EXISTS sprint_enterprise CASCADE;
DROP SCHEMA IF EXISTS sprint_core CASCADE;
