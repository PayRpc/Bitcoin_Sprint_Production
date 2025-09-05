-- Add enhanced monitoring and alerting capabilities
-- +migrate Up

-- Add monitoring tables to sprint_analytics schema
CREATE TABLE IF NOT EXISTS sprint_analytics.system_health (
    id SERIAL PRIMARY KEY,
    component VARCHAR(100) NOT NULL,
    status VARCHAR(20) NOT NULL,
    cpu_usage NUMERIC(5,2),
    memory_usage NUMERIC(5,2),
    disk_usage NUMERIC(5,2),
    response_time_ms INTEGER,
    error_count INTEGER DEFAULT 0,
    last_error TEXT,
    recorded_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
) PARTITION BY RANGE (recorded_at);

CREATE TABLE IF NOT EXISTS sprint_analytics.alert_rules (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    component VARCHAR(100) NOT NULL,
    metric VARCHAR(100) NOT NULL,
    threshold_value NUMERIC(15,6),
    comparison_operator VARCHAR(10) NOT NULL, -- '>', '<', '>=', '<=', '='
    severity VARCHAR(20) NOT NULL DEFAULT 'medium', -- 'low', 'medium', 'high', 'critical'
    is_active BOOLEAN DEFAULT true,
    notification_channels TEXT[], -- Email, Slack, etc.
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS sprint_analytics.alerts (
    id SERIAL PRIMARY KEY,
    rule_id INTEGER REFERENCES sprint_analytics.alert_rules(id),
    component VARCHAR(100) NOT NULL,
    message TEXT NOT NULL,
    severity VARCHAR(20) NOT NULL,
    metric_value NUMERIC(15,6),
    threshold_value NUMERIC(15,6),
    status VARCHAR(20) DEFAULT 'active', -- 'active', 'acknowledged', 'resolved'
    acknowledged_by VARCHAR(100),
    acknowledged_at TIMESTAMP WITH TIME ZONE,
    resolved_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
) PARTITION BY RANGE (created_at);

-- Add endpoint performance tracking
CREATE TABLE IF NOT EXISTS sprint_analytics.endpoint_metrics (
    id SERIAL PRIMARY KEY,
    endpoint VARCHAR(255) NOT NULL,
    node_id INTEGER REFERENCES sprint_core.nodes(id),
    success_count INTEGER DEFAULT 0,
    error_count INTEGER DEFAULT 0,
    total_response_time_ms BIGINT DEFAULT 0,
    min_response_time_ms INTEGER,
    max_response_time_ms INTEGER,
    avg_response_time_ms NUMERIC(10,2),
    last_success_at TIMESTAMP WITH TIME ZONE,
    last_error_at TIMESTAMP WITH TIME ZONE,
    window_start TIMESTAMP WITH TIME ZONE DEFAULT date_trunc('hour', NOW()),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(endpoint, node_id, window_start)
) PARTITION BY RANGE (window_start);

-- Create partitions for new tables
DO $$
DECLARE
    current_month TEXT;
    next_month TEXT;
    start_date DATE;
    end_date DATE;
    next_start_date DATE;
    next_end_date DATE;
BEGIN
    current_month := to_char(CURRENT_DATE, 'YYYY_MM');
    next_month := to_char(CURRENT_DATE + INTERVAL '1 month', 'YYYY_MM');
    start_date := date_trunc('month', CURRENT_DATE);
    end_date := date_trunc('month', CURRENT_DATE) + INTERVAL '1 month';
    next_start_date := date_trunc('month', CURRENT_DATE) + INTERVAL '1 month';
    next_end_date := date_trunc('month', CURRENT_DATE) + INTERVAL '2 months';
    
    -- System health partitions
    EXECUTE format('CREATE TABLE IF NOT EXISTS sprint_analytics.system_health_%s PARTITION OF sprint_analytics.system_health FOR VALUES FROM (%L) TO (%L)', 
                   current_month, start_date, end_date);
    EXECUTE format('CREATE TABLE IF NOT EXISTS sprint_analytics.system_health_%s PARTITION OF sprint_analytics.system_health FOR VALUES FROM (%L) TO (%L)', 
                   next_month, next_start_date, next_end_date);
    
    -- Alerts partitions
    EXECUTE format('CREATE TABLE IF NOT EXISTS sprint_analytics.alerts_%s PARTITION OF sprint_analytics.alerts FOR VALUES FROM (%L) TO (%L)', 
                   current_month, start_date, end_date);
    EXECUTE format('CREATE TABLE IF NOT EXISTS sprint_analytics.alerts_%s PARTITION OF sprint_analytics.alerts FOR VALUES FROM (%L) TO (%L)', 
                   next_month, next_start_date, next_end_date);
    
    -- Endpoint metrics partitions
    EXECUTE format('CREATE TABLE IF NOT EXISTS sprint_analytics.endpoint_metrics_%s PARTITION OF sprint_analytics.endpoint_metrics FOR VALUES FROM (%L) TO (%L)', 
                   current_month, start_date, end_date);
    EXECUTE format('CREATE TABLE IF NOT EXISTS sprint_analytics.endpoint_metrics_%s PARTITION OF sprint_analytics.endpoint_metrics FOR VALUES FROM (%L) TO (%L)', 
                   next_month, next_start_date, next_end_date);
END $$;

-- Create indexes for optimal performance
CREATE INDEX IF NOT EXISTS idx_system_health_component_recorded ON sprint_analytics.system_health(component, recorded_at);
CREATE INDEX IF NOT EXISTS idx_system_health_status ON sprint_analytics.system_health(status);
CREATE INDEX IF NOT EXISTS idx_alert_rules_component ON sprint_analytics.alert_rules(component);
CREATE INDEX IF NOT EXISTS idx_alert_rules_active ON sprint_analytics.alert_rules(is_active);
CREATE INDEX IF NOT EXISTS idx_alerts_status_created ON sprint_analytics.alerts(status, created_at);
CREATE INDEX IF NOT EXISTS idx_alerts_severity ON sprint_analytics.alerts(severity);
CREATE INDEX IF NOT EXISTS idx_alerts_rule_id ON sprint_analytics.alerts(rule_id);
CREATE INDEX IF NOT EXISTS idx_endpoint_metrics_endpoint_window ON sprint_analytics.endpoint_metrics(endpoint, window_start);
CREATE INDEX IF NOT EXISTS idx_endpoint_metrics_node_window ON sprint_analytics.endpoint_metrics(node_id, window_start);

-- Create function to calculate endpoint performance metrics
CREATE OR REPLACE FUNCTION calculate_endpoint_metrics(
    p_endpoint VARCHAR(255),
    p_node_id INTEGER,
    p_window_start TIMESTAMP WITH TIME ZONE
) RETURNS VOID AS $$
DECLARE
    metrics_record RECORD;
BEGIN
    -- Calculate aggregated metrics from request_stats
    SELECT 
        COUNT(*) FILTER (WHERE status_code BETWEEN 200 AND 299) AS success_count,
        COUNT(*) FILTER (WHERE status_code >= 400) AS error_count,
        SUM(response_time_ms) AS total_response_time,
        MIN(response_time_ms) AS min_response_time,
        MAX(response_time_ms) AS max_response_time,
        AVG(response_time_ms) AS avg_response_time,
        MAX(created_at) FILTER (WHERE status_code BETWEEN 200 AND 299) AS last_success,
        MAX(created_at) FILTER (WHERE status_code >= 400) AS last_error
    INTO metrics_record
    FROM sprint_analytics.request_stats 
    WHERE endpoint = p_endpoint 
    AND created_at >= p_window_start 
    AND created_at < p_window_start + INTERVAL '1 hour';

    -- Insert or update endpoint metrics
    INSERT INTO sprint_analytics.endpoint_metrics (
        endpoint, node_id, success_count, error_count, 
        total_response_time_ms, min_response_time_ms, max_response_time_ms, avg_response_time_ms,
        last_success_at, last_error_at, window_start
    ) VALUES (
        p_endpoint, p_node_id, 
        COALESCE(metrics_record.success_count, 0),
        COALESCE(metrics_record.error_count, 0),
        COALESCE(metrics_record.total_response_time, 0),
        metrics_record.min_response_time,
        metrics_record.max_response_time,
        metrics_record.avg_response_time,
        metrics_record.last_success,
        metrics_record.last_error,
        p_window_start
    ) ON CONFLICT (endpoint, node_id, window_start) DO UPDATE SET
        success_count = EXCLUDED.success_count,
        error_count = EXCLUDED.error_count,
        total_response_time_ms = EXCLUDED.total_response_time_ms,
        min_response_time_ms = EXCLUDED.min_response_time_ms,
        max_response_time_ms = EXCLUDED.max_response_time_ms,
        avg_response_time_ms = EXCLUDED.avg_response_time_ms,
        last_success_at = EXCLUDED.last_success_at,
        last_error_at = EXCLUDED.last_error_at;
END;
$$ LANGUAGE plpgsql;

-- Create function to evaluate alert rules
CREATE OR REPLACE FUNCTION evaluate_alert_rules() RETURNS INTEGER AS $$
DECLARE
    rule_record RECORD;
    metric_value NUMERIC(15,6);
    alert_triggered BOOLEAN;
    alerts_created INTEGER := 0;
BEGIN
    -- Iterate through active alert rules
    FOR rule_record IN 
        SELECT * FROM sprint_analytics.alert_rules WHERE is_active = true
    LOOP
        -- Get the latest metric value for this rule
        CASE rule_record.metric
            WHEN 'cpu_usage' THEN
                SELECT cpu_usage INTO metric_value 
                FROM sprint_analytics.system_health 
                WHERE component = rule_record.component 
                ORDER BY recorded_at DESC LIMIT 1;
            WHEN 'memory_usage' THEN
                SELECT memory_usage INTO metric_value 
                FROM sprint_analytics.system_health 
                WHERE component = rule_record.component 
                ORDER BY recorded_at DESC LIMIT 1;
            WHEN 'response_time_ms' THEN
                SELECT avg_response_time_ms INTO metric_value 
                FROM sprint_analytics.endpoint_metrics 
                WHERE endpoint = rule_record.component 
                ORDER BY window_start DESC LIMIT 1;
            WHEN 'error_rate' THEN
                SELECT 
                    CASE WHEN (success_count + error_count) > 0 
                    THEN (error_count::NUMERIC / (success_count + error_count)) * 100 
                    ELSE 0 END INTO metric_value
                FROM sprint_analytics.endpoint_metrics 
                WHERE endpoint = rule_record.component 
                ORDER BY window_start DESC LIMIT 1;
            ELSE
                CONTINUE; -- Skip unknown metrics
        END CASE;

        -- Skip if no metric value found
        CONTINUE WHEN metric_value IS NULL;

        -- Evaluate condition
        alert_triggered := CASE rule_record.comparison_operator
            WHEN '>' THEN metric_value > rule_record.threshold_value
            WHEN '<' THEN metric_value < rule_record.threshold_value
            WHEN '>=' THEN metric_value >= rule_record.threshold_value
            WHEN '<=' THEN metric_value <= rule_record.threshold_value
            WHEN '=' THEN metric_value = rule_record.threshold_value
            ELSE FALSE
        END;

        -- Create alert if triggered and no active alert exists
        IF alert_triggered THEN
            INSERT INTO sprint_analytics.alerts (
                rule_id, component, message, severity, 
                metric_value, threshold_value, status
            )
            SELECT 
                rule_record.id,
                rule_record.component,
                format('Alert: %s %s %s %s (current: %s)', 
                       rule_record.component, rule_record.metric, 
                       rule_record.comparison_operator, rule_record.threshold_value, metric_value),
                rule_record.severity,
                metric_value,
                rule_record.threshold_value,
                'active'
            WHERE NOT EXISTS (
                SELECT 1 FROM sprint_analytics.alerts 
                WHERE rule_id = rule_record.id 
                AND status = 'active'
                AND created_at > NOW() - INTERVAL '1 hour'
            );
            
            alerts_created := alerts_created + 1;
        END IF;
    END LOOP;

    RETURN alerts_created;
END;
$$ LANGUAGE plpgsql;

-- Insert default alert rules
INSERT INTO sprint_analytics.alert_rules (name, component, metric, threshold_value, comparison_operator, severity, notification_channels) VALUES
('High CPU Usage', 'system', 'cpu_usage', 80.0, '>', 'high', ARRAY['email', 'slack']),
('High Memory Usage', 'system', 'memory_usage', 85.0, '>', 'high', ARRAY['email', 'slack']),
('API Response Time', 'api', 'response_time_ms', 5000.0, '>', 'medium', ARRAY['email']),
('High Error Rate', 'api', 'error_rate', 5.0, '>', 'critical', ARRAY['email', 'slack', 'pager'])
ON CONFLICT (name) DO NOTHING;

-- Update triggers for new tables
CREATE TRIGGER update_alert_rules_updated_at BEFORE UPDATE ON sprint_analytics.alert_rules 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- +migrate Down

-- Drop functions
DROP FUNCTION IF EXISTS evaluate_alert_rules();
DROP FUNCTION IF EXISTS calculate_endpoint_metrics(VARCHAR, INTEGER, TIMESTAMP WITH TIME ZONE);

-- Drop tables
DROP TABLE IF EXISTS sprint_analytics.endpoint_metrics;
DROP TABLE IF EXISTS sprint_analytics.alerts;
DROP TABLE IF EXISTS sprint_analytics.alert_rules;
DROP TABLE IF EXISTS sprint_analytics.system_health;
