-- Bitcoin Sprint Multi-Chain Platform Database Initialization
-- PostgreSQL schema for enterprise features and multi-chain data

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "pg_stat_statements";

-- Create schemas for different components
CREATE SCHEMA IF NOT EXISTS sprint_core;
CREATE SCHEMA IF NOT EXISTS sprint_enterprise;
CREATE SCHEMA IF NOT EXISTS sprint_chains;
CREATE SCHEMA IF NOT EXISTS sprint_analytics;

-- ===== CORE TABLES =====

-- API key management
CREATE TABLE sprint_core.api_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    key_hash VARCHAR(128) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    tier VARCHAR(50) NOT NULL DEFAULT 'basic',
    rate_limit INTEGER NOT NULL DEFAULT 1000,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE,
    is_active BOOLEAN DEFAULT true,
    metadata JSONB DEFAULT '{}'
);

-- Request logs for analytics
CREATE TABLE sprint_core.request_logs (
    id BIGSERIAL PRIMARY KEY,
    api_key_id UUID REFERENCES sprint_core.api_keys(id),
    chain VARCHAR(50) NOT NULL,
    method VARCHAR(100) NOT NULL,
    endpoint VARCHAR(255) NOT NULL,
    request_size INTEGER,
    response_size INTEGER,
    response_time_ms INTEGER,
    status_code INTEGER,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create partitioned table for request logs (by month)
CREATE TABLE sprint_core.request_logs_y2024m01 PARTITION OF sprint_core.request_logs
    FOR VALUES FROM ('2024-01-01') TO ('2024-02-01');

-- ===== ENTERPRISE SECURITY TABLES =====

-- Audit logs for enterprise features
CREATE TABLE sprint_enterprise.audit_logs (
    id BIGSERIAL PRIMARY KEY,
    event_type VARCHAR(100) NOT NULL,
    user_id VARCHAR(255),
    resource_type VARCHAR(100),
    resource_id VARCHAR(255),
    action VARCHAR(100) NOT NULL,
    details JSONB,
    ip_address INET,
    user_agent TEXT,
    security_level VARCHAR(50),
    risk_score INTEGER,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Security sessions
CREATE TABLE sprint_enterprise.security_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    session_token VARCHAR(255) UNIQUE NOT NULL,
    api_key_id UUID REFERENCES sprint_core.api_keys(id),
    security_level VARCHAR(50) NOT NULL,
    fingerprint_hash VARCHAR(128),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_activity TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    is_active BOOLEAN DEFAULT true
);

-- Entropy generation tracking
CREATE TABLE sprint_enterprise.entropy_usage (
    id BIGSERIAL PRIMARY KEY,
    session_id UUID REFERENCES sprint_enterprise.security_sessions(id),
    entropy_type VARCHAR(50) NOT NULL, -- 'fast', 'hybrid', 'quantum'
    bytes_generated INTEGER NOT NULL,
    generation_time_ms INTEGER,
    quality_score DECIMAL(5,2),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- ===== MULTI-CHAIN DATA TABLES =====

-- Chain status tracking
CREATE TABLE sprint_chains.chain_status (
    id SERIAL PRIMARY KEY,
    chain_name VARCHAR(50) UNIQUE NOT NULL,
    rpc_endpoint VARCHAR(255) NOT NULL,
    websocket_endpoint VARCHAR(255),
    block_height BIGINT,
    is_synced BOOLEAN DEFAULT false,
    last_block_time TIMESTAMP WITH TIME ZONE,
    avg_block_time DECIMAL(10,2),
    peer_count INTEGER,
    health_score DECIMAL(3,1),
    last_updated TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Insert initial chain configurations
INSERT INTO sprint_chains.chain_status (chain_name, rpc_endpoint, websocket_endpoint) VALUES
('bitcoin', 'http://bitcoin-core:8332', 'tcp://bitcoin-core:28332'),
('ethereum', 'http://geth:8545', 'ws://geth:8546'),
('solana', 'http://solana-validator:8899', 'ws://solana-validator:8900'),
('cosmos', 'http://cosmos-hub:1317', 'ws://cosmos-hub:26657'),
('polkadot', 'ws://polkadot-node:9944', 'ws://polkadot-node:9944'),
('avalanche', 'http://avalanche-node:9650', 'ws://avalanche-node:9651'),
('polygon', 'http://polygon-node:8545', 'ws://polygon-node:8546'),
('cardano', 'http://cardano-node:3001', 'ws://cardano-node:3002');

-- Transaction cache for fast lookups
CREATE TABLE sprint_chains.transaction_cache (
    id BIGSERIAL PRIMARY KEY,
    chain VARCHAR(50) NOT NULL,
    tx_hash VARCHAR(128) NOT NULL,
    block_height BIGINT,
    block_hash VARCHAR(128),
    tx_data JSONB NOT NULL,
    confirmations INTEGER DEFAULT 0,
    cached_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE,
    UNIQUE(chain, tx_hash)
);

-- Block data cache
CREATE TABLE sprint_chains.block_cache (
    id BIGSERIAL PRIMARY KEY,
    chain VARCHAR(50) NOT NULL,
    block_height BIGINT NOT NULL,
    block_hash VARCHAR(128) UNIQUE NOT NULL,
    block_data JSONB NOT NULL,
    tx_count INTEGER,
    cached_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(chain, block_height)
);

-- ===== ANALYTICS TABLES =====

-- Chain performance metrics
CREATE TABLE sprint_analytics.chain_metrics (
    id BIGSERIAL PRIMARY KEY,
    chain VARCHAR(50) NOT NULL,
    metric_name VARCHAR(100) NOT NULL,
    metric_value DECIMAL(20,6),
    metric_unit VARCHAR(50),
    recorded_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- API performance metrics
CREATE TABLE sprint_analytics.api_metrics (
    id BIGSERIAL PRIMARY KEY,
    endpoint VARCHAR(255) NOT NULL,
    method VARCHAR(10) NOT NULL,
    avg_response_time_ms DECIMAL(10,3),
    min_response_time_ms DECIMAL(10,3),
    max_response_time_ms DECIMAL(10,3),
    request_count BIGINT,
    error_count BIGINT,
    recorded_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- ===== INDEXES FOR PERFORMANCE =====

-- Request logs indexes
CREATE INDEX idx_request_logs_api_key ON sprint_core.request_logs(api_key_id);
CREATE INDEX idx_request_logs_chain ON sprint_core.request_logs(chain);
CREATE INDEX idx_request_logs_created_at ON sprint_core.request_logs(created_at);
CREATE INDEX idx_request_logs_method ON sprint_core.request_logs(method);

-- Audit logs indexes
CREATE INDEX idx_audit_logs_event_type ON sprint_enterprise.audit_logs(event_type);
CREATE INDEX idx_audit_logs_user_id ON sprint_enterprise.audit_logs(user_id);
CREATE INDEX idx_audit_logs_created_at ON sprint_enterprise.audit_logs(created_at);

-- Chain data indexes
CREATE INDEX idx_transaction_cache_chain_hash ON sprint_chains.transaction_cache(chain, tx_hash);
CREATE INDEX idx_block_cache_chain_height ON sprint_chains.block_cache(chain, block_height);
CREATE INDEX idx_chain_metrics_chain_recorded ON sprint_analytics.chain_metrics(chain, recorded_at);

-- ===== FUNCTIONS AND TRIGGERS =====

-- Update timestamp function
CREATE OR REPLACE FUNCTION update_modified_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Add update triggers
CREATE TRIGGER update_api_keys_modtime 
    BEFORE UPDATE ON sprint_core.api_keys 
    FOR EACH ROW EXECUTE FUNCTION update_modified_column();

-- Cache cleanup function
CREATE OR REPLACE FUNCTION cleanup_expired_cache()
RETURNS void AS $$
BEGIN
    DELETE FROM sprint_chains.transaction_cache WHERE expires_at < NOW();
    DELETE FROM sprint_enterprise.security_sessions WHERE expires_at < NOW();
END;
$$ LANGUAGE plpgsql;

-- Create cleanup job (requires pg_cron extension in production)
-- SELECT cron.schedule('cache-cleanup', '0 */6 * * *', 'SELECT cleanup_expired_cache();');

-- ===== VIEWS FOR ANALYTICS =====

-- Chain health dashboard view
CREATE VIEW sprint_analytics.chain_health AS
SELECT 
    cs.chain_name,
    cs.is_synced,
    cs.block_height,
    cs.health_score,
    cs.peer_count,
    cs.last_updated,
    COUNT(rl.id) as request_count_24h,
    AVG(rl.response_time_ms) as avg_response_time_24h
FROM sprint_chains.chain_status cs
LEFT JOIN sprint_core.request_logs rl ON rl.chain = cs.chain_name 
    AND rl.created_at > NOW() - INTERVAL '24 hours'
GROUP BY cs.chain_name, cs.is_synced, cs.block_height, cs.health_score, cs.peer_count, cs.last_updated;

-- API usage summary view
CREATE VIEW sprint_analytics.api_usage_summary AS
SELECT 
    ak.name as api_key_name,
    ak.tier,
    COUNT(rl.id) as request_count,
    COUNT(DISTINCT rl.chain) as chains_used,
    AVG(rl.response_time_ms) as avg_response_time,
    SUM(CASE WHEN rl.status_code >= 400 THEN 1 ELSE 0 END) as error_count
FROM sprint_core.api_keys ak
LEFT JOIN sprint_core.request_logs rl ON rl.api_key_id = ak.id
    AND rl.created_at > NOW() - INTERVAL '24 hours'
WHERE ak.is_active = true
GROUP BY ak.id, ak.name, ak.tier;

-- Grant permissions
GRANT USAGE ON SCHEMA sprint_core TO sprint;
GRANT USAGE ON SCHEMA sprint_enterprise TO sprint;
GRANT USAGE ON SCHEMA sprint_chains TO sprint;
GRANT USAGE ON SCHEMA sprint_analytics TO sprint;

GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA sprint_core TO sprint;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA sprint_enterprise TO sprint;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA sprint_chains TO sprint;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA sprint_analytics TO sprint;

GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA sprint_core TO sprint;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA sprint_enterprise TO sprint;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA sprint_chains TO sprint;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA sprint_analytics TO sprint;

-- Create initial admin API key
INSERT INTO sprint_core.api_keys (key_hash, name, tier, rate_limit) VALUES
(encode(digest('sprint-admin-key-2024', 'sha256'), 'hex'), 'Admin Key', 'enterprise', 10000);
