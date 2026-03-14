-- Proxyd Database Schema
-- SQLite database for mihomo proxy management system

-- Sources table: subscription configuration sources
CREATE TABLE IF NOT EXISTS sources (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    type TEXT NOT NULL CHECK(type IN ('http', 'file', 'local')),
    url TEXT,
    path TEXT,
    update_interval INTEGER DEFAULT 3600,
    update_cron TEXT,
    enabled BOOLEAN DEFAULT 1,
    priority INTEGER DEFAULT 0,
    config_override TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Settings table: system configuration
CREATE TABLE IF NOT EXISTS settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    description TEXT,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Revisions table: configuration version history
CREATE TABLE IF NOT EXISTS revisions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    version TEXT NOT NULL UNIQUE,
    content TEXT NOT NULL,
    source_hash TEXT,
    created_by TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Runtime table: mihomo runtime state
CREATE TABLE IF NOT EXISTS runtime (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    pid INTEGER,
    port INTEGER,
    config_path TEXT,
    status TEXT CHECK(status IN ('running', 'stopped', 'error')),
    uptime INTEGER DEFAULT 0,
    memory_usage INTEGER DEFAULT 0,
    last_check DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Audit log table
CREATE TABLE IF NOT EXISTS audit_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user TEXT,
    action TEXT NOT NULL,
    resource TEXT,
    details TEXT,
    ip_address TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_sources_enabled ON sources(enabled);
CREATE INDEX IF NOT EXISTS idx_sources_priority ON sources(priority);
CREATE INDEX IF NOT EXISTS idx_revisions_created_at ON revisions(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_runtime_status ON runtime(status);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created_at ON audit_logs(created_at DESC);

-- Insert default settings
INSERT OR IGNORE INTO settings (key, value, description) VALUES
    ('mihomo_path', '/usr/local/bin/mihomo', 'Path to mihomo binary'),
    ('mihomo_config_dir', '/etc/mihomo', 'Directory for mihomo configs'),
    ('listen_port', '9090', 'API listen port'),
    ('jwt_secret', 'change-me-in-production', 'JWT signing secret'),
    ('session_timeout', '86400', 'Session timeout in seconds'),
    ('log_level', 'info', 'Log level: debug, info, warn, error'),
    ('enable_cors', 'true', 'Enable CORS for API'),
    ('max_config_size', '10485760', 'Max config size in bytes (10MB)');

-- Insert default admin user (password: admin, change immediately)
INSERT OR IGNORE INTO settings (key, value, description) VALUES
    ('admin_username', 'admin', 'Admin username'),
    ('admin_password', 'admin', 'Admin password');

-- Insert initial runtime state
INSERT OR IGNORE INTO runtime (pid, port, config_path, status) VALUES
    (NULL, 7890, '', 'stopped');
