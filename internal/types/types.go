package types

import "time"

// Source represents a subscription source
type Source struct {
	ID              int       `json:"id"`
	Name            string    `json:"name" binding:"required"`
	Type            string    `json:"type" binding:"required,oneof=http file local"`
	URL             string    `json:"url,omitempty"`
	Path            string    `json:"path,omitempty"`
	UpdateInterval  int       `json:"update_interval"`
	UpdateCron      string    `json:"update_cron,omitempty"`
	Enabled         bool      `json:"enabled"`
	Priority        int       `json:"priority"`
	ConfigOverride  string    `json:"config_override,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// Setting represents a system setting
type Setting struct {
	Key         string    `json:"key" binding:"required"`
	Value       string    `json:"value" binding:"required"`
	Description string    `json:"description,omitempty"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Revision represents a configuration version
type Revision struct {
	ID          int       `json:"id"`
	Version     string    `json:"version" binding:"required"`
	Content     string    `json:"content" binding:"required"`
	SourceHash  string    `json:"source_hash,omitempty"`
	CreatedBy   string    `json:"created_by,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// Runtime represents mihomo runtime state
type Runtime struct {
	ID          int       `json:"id"`
	PID         int       `json:"pid,omitempty"`
	Port        int       `json:"port"`
	ConfigPath  string    `json:"config_path"`
	Status      string    `json:"status"`
	Uptime      int       `json:"uptime"`
	MemoryUsage int       `json:"memory_usage"`
	LastCheck   time.Time `json:"last_check"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// AuditLog represents an audit log entry
type AuditLog struct {
	ID        int       `json:"id"`
	User      string    `json:"user,omitempty"`
	Action    string    `json:"action" binding:"required"`
	Resource  string    `json:"resource,omitempty"`
	Details   string    `json:"details,omitempty"`
	IPAddress string    `json:"ip_address,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// MihomoConfig represents the mihomo configuration structure
type MihomoConfig struct {
	Port             int                    `json:"port"`
	SocksPort        int                    `json:"socks-port,omitempty"`
	AllowLan         bool                   `json:"allow-lan"`
	BindAddress      string                 `json:"bind-address,omitempty"`
	Mode             string                 `json:"mode"`
	LogLevel         string                 `json:"log-level"`
	IPv6             bool                   `json:"ipv6"`
	ExternalController string              `json:"external-controller"`
	ExternalUI       string                 `json:"external-ui,omitempty"`
	Secret           string                 `json:"secret,omitempty"`
	Proxies          []interface{}          `json:"proxies"`
	ProxyGroups      []interface{}          `json:"proxy-groups"`
	Rules            []interface{}          `json:"rules"`
	DNS              map[string]interface{} `json:"dns,omitempty"`
}

// MihomoTraffic represents mihomo traffic statistics
type MihomoTraffic struct {
	Up   int `json:"up"`
	Down int `json:"down"`
}

// MihomoMemory represents mihomo memory usage
type MihomoMemory struct {
	Inuse   int `json:"inuse"`
	Maxlimit int `json:"maxlimit,omitempty"`
}

// ProxyInfo represents information about a proxy
type ProxyInfo struct {
	Name   string      `json:"name"`
	Type   string      `json:"type"`
	UDP    bool        `json:"udp,omitempty"`
	History []ProxyHistory `json:"history,omitempty"`
	Alive  bool        `json:"alive,omitempty"`
	Delay  int         `json:"delay,omitempty"`
}

// ProxyHistory represents proxy history
type ProxyHistory struct {
	Time time.Time `json:"time"`
	Delay int      `json:"delay"`
}

// GroupDelay represents proxy group delay info
type GroupDelay struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Delay map[string]int `json:"delay,omitempty"`
}

// LoginRequest represents login request
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse represents login response
type LoginResponse struct {
	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at"`
}

// SystemInfo represents system information
type SystemInfo struct {
	Version    string    `json:"version"`
	GoVersion  string    `json:"go_version"`
	Uptime     int64     `json:"uptime"`
	MihomoStatus string  `json:"mihomo_status"`
	Database   string    `json:"database"`
}

// ConfigRequest represents config generation request
type ConfigRequest struct {
	SourceIDs []int `json:"source_ids"`
	PolicyID  *int  `json:"policy_id,omitempty"`
}

// TestResult represents subscription test result
type TestResult struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
	Latency int    `json:"latency,omitempty"`
	Size    int    `json:"size,omitempty"`
}

// HealthStatus represents health check status
type HealthStatus struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Services  map[string]string `json:"services"`
}
