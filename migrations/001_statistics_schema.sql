-- FizzBuzz Statistics Database Schema
-- Version: 1.0
-- Description: PostgreSQL schema for persistent statistics storage

-- Create statistics table with optimized structure
CREATE TABLE fizzbuzz_statistics (
    id BIGSERIAL PRIMARY KEY,
    parameters_hash VARCHAR(64) NOT NULL UNIQUE, -- SHA256 hash for collision-free keys
    
    -- FizzBuzz parameters (denormalized for query performance)
    int1 INTEGER NOT NULL,
    int2 INTEGER NOT NULL,
    limit_value INTEGER NOT NULL, -- 'limit' is reserved keyword
    str1 VARCHAR(255) NOT NULL,
    str2 VARCHAR(255) NOT NULL,
    
    -- Statistics tracking
    hits BIGINT NOT NULL DEFAULT 1,
    
    -- Audit timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Optimized indexes for query patterns
CREATE INDEX idx_statistics_hits_desc ON fizzbuzz_statistics (hits DESC);
CREATE INDEX idx_statistics_created_at ON fizzbuzz_statistics (created_at);
CREATE INDEX idx_statistics_parameters ON fizzbuzz_statistics (int1, int2, limit_value);
CREATE INDEX idx_statistics_updated_at ON fizzbuzz_statistics (updated_at);

-- Atomic increment function for thread-safe hit counting
CREATE OR REPLACE FUNCTION increment_statistics(
    p_hash VARCHAR(64),
    p_int1 INTEGER,
    p_int2 INTEGER,
    p_limit INTEGER,
    p_str1 VARCHAR(255),
    p_str2 VARCHAR(255)
) RETURNS BIGINT AS $$
DECLARE
    current_hits BIGINT;
BEGIN
    -- Atomic upsert: insert new or increment existing
    INSERT INTO fizzbuzz_statistics 
    (parameters_hash, int1, int2, limit_value, str1, str2, hits)
    VALUES (p_hash, p_int1, p_int2, p_limit, p_str1, p_str2, 1)
    ON CONFLICT (parameters_hash) 
    DO UPDATE SET 
        hits = fizzbuzz_statistics.hits + 1,
        updated_at = NOW()
    RETURNING hits INTO current_hits;
    
    RETURN current_hits;
END;
$$ LANGUAGE plpgsql;

-- Function to get most frequent request (optimized for cache refresh)
CREATE OR REPLACE FUNCTION get_most_frequent_request()
RETURNS TABLE(
    parameters_hash VARCHAR(64),
    int1 INTEGER,
    int2 INTEGER,
    limit_value INTEGER,
    str1 VARCHAR(255),
    str2 VARCHAR(255),
    hits BIGINT,
    created_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        s.parameters_hash,
        s.int1,
        s.int2, 
        s.limit_value,
        s.str1,
        s.str2,
        s.hits,
        s.created_at,
        s.updated_at
    FROM fizzbuzz_statistics s
    ORDER BY s.hits DESC, s.created_at ASC
    LIMIT 1;
END;
$$ LANGUAGE plpgsql;

-- Function to get top N requests for analytics
CREATE OR REPLACE FUNCTION get_top_requests(n INTEGER)
RETURNS TABLE(
    parameters_hash VARCHAR(64),
    int1 INTEGER,
    int2 INTEGER,
    limit_value INTEGER,
    str1 VARCHAR(255),
    str2 VARCHAR(255),
    hits BIGINT,
    created_at TIMESTAMP WITH TIME ZONE,
    updated_at TIMESTAMP WITH TIME ZONE
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        s.parameters_hash,
        s.int1,
        s.int2,
        s.limit_value,
        s.str1,
        s.str2,
        s.hits,
        s.created_at,
        s.updated_at
    FROM fizzbuzz_statistics s
    ORDER BY s.hits DESC, s.created_at ASC
    LIMIT n;
END;
$$ LANGUAGE plpgsql;

-- Note: Sample data removed for production deployment
-- For development, you can manually insert test data if needed

-- Grant permissions to application user
GRANT SELECT, INSERT, UPDATE ON fizzbuzz_statistics TO fizzbuzz_user;
GRANT USAGE, SELECT ON SEQUENCE fizzbuzz_statistics_id_seq TO fizzbuzz_user;
GRANT EXECUTE ON FUNCTION increment_statistics(VARCHAR(64), INTEGER, INTEGER, INTEGER, VARCHAR(255), VARCHAR(255)) TO fizzbuzz_user;
GRANT EXECUTE ON FUNCTION get_most_frequent_request() TO fizzbuzz_user;
GRANT EXECUTE ON FUNCTION get_top_requests(INTEGER) TO fizzbuzz_user;

-- Performance monitoring views (for operational insights)
CREATE VIEW v_statistics_summary AS
SELECT 
    COUNT(*) as total_unique_requests,
    SUM(hits) as total_requests,
    AVG(hits) as avg_hits_per_unique_request,
    MAX(hits) as max_hits,
    MIN(created_at) as first_request_time,
    MAX(updated_at) as last_request_time
FROM fizzbuzz_statistics;

GRANT SELECT ON v_statistics_summary TO fizzbuzz_user;

-- Database setup complete
SELECT 'FizzBuzz statistics database schema initialized successfully' AS status;