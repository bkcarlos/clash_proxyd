# API Documentation

This document describes the REST API endpoints provided by proxyd.

## Base URL

```
http://localhost:8080/api/v1
```

## Authentication

Most endpoints require authentication using a JWT bearer token.

### Login

Authenticate and receive a JWT token.

**Request:**
```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "username": "admin",
  "password": "admin"
}
```

**Response:**
```json
{
  "token": "eyJhbGciOiJIUzI1NiIs...",
  "expires_at": 1640995200
}
```

### Using the Token

Include the token in the Authorization header:
```http
Authorization: Bearer <token>
```

## Endpoints

### Authentication

#### POST /auth/login
Login and receive JWT token.

**Request Body:**
```json
{
  "username": "string",
  "password": "string"
}
```

**Response (200):**
```json
{
  "token": "string",
  "expires_at": 1640995200
}
```

#### POST /auth/logout
Logout (client-side token removal).

**Headers:** `Authorization: Bearer <token>`

**Response (200):**
```json
{
  "message": "Logged out successfully"
}
```

#### POST /auth/refresh
Refresh JWT token.

**Headers:** `Authorization: Bearer <token>`

**Response (200):**
```json
{
  "token": "string",
  "expires_at": 1640995200
}
```

#### GET /auth/profile
Get current user profile.

**Headers:** `Authorization: Bearer <token>`

**Response (200):**
```json
{
  "username": "admin",
  "role": "admin"
}
```

#### PUT /auth/password
Update password.

**Headers:** `Authorization: Bearer <token>`

**Request Body:**
```json
{
  "old_password": "string",
  "new_password": "string"
}
```

### System

#### GET /system/info
Get system information.

**Response (200):**
```json
{
  "version": "1.0.0",
  "go_version": "go1.22.0",
  "uptime": 3600,
  "mihomo_status": "running",
  "database": "sqlite"
}
```

#### GET /system/status
Get detailed system status.

**Response (200):**
```json
{
  "uptime": "1h0m0s",
  "goroutines": 10,
  "memory_alloc": 1024000,
  "mihomo_status": "running",
  "mihomo_pid": 12345
}
```

#### GET /system/settings
Get all system settings.

**Response (200):**
```json
[
  {
    "key": "mihomo_path",
    "value": "/usr/local/bin/mihomo",
    "description": "Path to mihomo binary"
  }
]
```

#### PUT /system/settings
Update a system setting.

**Request Body:**
```json
{
  "key": "mihomo_path",
  "value": "/usr/local/bin/mihomo",
  "description": "Path to mihomo binary"
}
```

#### PUT /system/settings/batch
Batch update system settings.

**Request Body:**
```json
{
  "settings": [
    {
      "key": "log_level",
      "value": "debug",
      "description": "Log level"
    },
    {
      "key": "enable_cors",
      "value": "false",
      "description": "Enable CORS"
    }
  ]
}
```

**Response (200):**
```json
{
  "message": "Settings updated successfully"
}
```

### Sources

#### GET /sources
List all sources.

**Response (200):**
```json
[
  {
    "id": 1,
    "name": "My Subscription",
    "type": "http",
    "url": "https://example.com/sub",
    "enabled": true,
    "priority": 0,
    "update_interval": 3600
  }
]
```

#### POST /sources
Create a new source.

**Request Body:**
```json
{
  "name": "My Subscription",
  "type": "http",
  "url": "https://example.com/sub",
  "enabled": true,
  "priority": 0,
  "update_interval": 3600
}
```

**Response (201):** Returns the created source object.

#### GET /sources/:id
Get a specific source.

**Response (200):** Returns source object.

#### PUT /sources/:id
Update a source.

**Request Body:** Same as POST /sources

**Response (200):** Returns updated source object.

#### DELETE /sources/:id
Delete a source.

**Response (200):**
```json
{
  "message": "Source deleted successfully"
}
```

#### POST /sources/:id/test
Test a source connection.

**Response (200):**
```json
{
  "success": true,
  "latency": 150,
  "size": 1024
}
```

#### POST /sources/:id/fetch
Fetch source content immediately.

**Response (200):**
```json
{
  "content": "proxies: ...",
  "size": 1024,
  "hash": "abc123..."
}
```

### Configuration

#### POST /config/generate
Generate mihomo configuration from sources.

**Request Body:**
```json
{
  "source_ids": [1, 2, 3]
}
```

**Response (200):**
```json
{
  "config": "proxies: ...",
  "hash": "abc123..."
}
```

#### GET /config
Get current mihomo configuration.

**Response (200):**
```json
{
  "config": "proxies: ...",
  "version": "1"
}
```

#### POST /config/save
Save configuration to file (path must be under configured `mihomo_config_dir`).

**Request Body:**
```json
{
  "config": "proxies: ...",
  "path": "/etc/mihomo/runtime.yaml"
}
```

#### POST /config/apply
Apply config content and reload runtime.

If `config` is empty, latest revision content is used.
If `path` is empty, default runtime path is used.

**Request Body:**
```json
{
  "config": "proxies: ...",
  "path": "/etc/mihomo/runtime.yaml"
}
```

**Response (200):**
```json
{
  "message": "Configuration applied successfully"
}
```

#### GET /config/revisions
List configuration revisions.

**Query Parameters:**
- `limit` (optional): Number of revisions to return (default: 50)

**Response (200):**
```json
[
  {
    "id": 1,
    "version": "v1.0.0",
    "created_at": "2024-01-01T00:00:00Z"
  }
]
```

#### GET /config/revisions/:id
Get a specific revision.

**Response (200):** Returns revision object with content.

#### DELETE /config/revisions/:id
Delete a revision.

### Policy

#### POST /policy/groups
Generate proxy groups.

**Request Body:**
```json
{
  "proxy_names": ["proxy1", "proxy2"]
}
```

**Response (200):**
```json
{
  "groups": [...]
}
```

#### POST /policy/rules
Generate rules.

**Request Body:**
```json
{
  "custom_rules": ["DOMAIN-SUFFIX,example.com,DIRECT"]
}
```

**Response (200):**
```json
{
  "rules": [...]
}
```

#### POST /policy/validate-rule
Validate a rule.

**Request Body:**
```json
{
  "rule": "DOMAIN-SUFFIX,example.com,DIRECT"
}
```

**Response (200):**
```json
{
  "message": "Rule is valid"
}
```

#### POST /policy/custom-group
Create a custom proxy group.

**Request Body:**
```json
{
  "name": "MyGroup",
  "type": "select",
  "proxies": ["proxy1", "proxy2"]
}
```

### Proxies

#### GET /proxy/proxies
Get all proxies from mihomo.

**Response (200):**
```json
{
  "proxies": {
    "proxy1": {...},
    "proxy2": {...}
  }
}
```

#### GET /proxy/proxies/:name
Get a specific proxy.

**Response (200):** Returns proxy object.

#### POST /proxy/proxies/:name/test
Test proxy delay.

Supports query parameters and JSON body.

**Query Parameters:**
- `url` (optional): Test URL (default: http://www.gstatic.com/generate_204)
- `timeout` (optional): Timeout in ms (default: 3000)

**JSON Body (optional):**
```json
{
  "url": "http://www.gstatic.com/generate_204",
  "timeout": 3000
}
```

**Response (200):
```json
{
  "name": "proxy1",
  "delay": 150,
  "url": "http://www.gstatic.com/generate_204",
  "timeout": 3000
}
```

#### PUT /proxy/groups/:group
Switch proxy in a group.

**Request Body:**
```json
{
  "proxy": "proxy1"
}
```

#### GET /proxy/groups
Get proxy groups derived from mihomo proxy data.

**Response (200):**
```json
{
  "groups": [
    {
      "name": "Proxy",
      "type": "Selector",
      "now": "proxy1",
      "proxies": ["proxy1", "proxy2"]
    }
  ]
}
```

#### GET /proxy/rules
Get active rules.

**Response (200):**
```json
{
  "rules": [...]
}
```

#### GET /proxy/traffic
Get traffic statistics.

**Response (200):**
```json
{
  "up": 1024000,
  "down": 2048000
}
```

#### GET /proxy/memory
Get memory usage.

**Response (200):**
```json
{
  "inuse": 10485760
}
```

#### POST /proxy/mihomo/:action
Control mihomo process.

**URL Parameters:**
- `action`: `start`, `stop`, or `restart`

**Response (200):**
```json
{
  "message": "Mihomo started successfully"
}
```

## Error Responses

All endpoints may return error responses:

```json
{
  "error": "Error message"
}
```

Common HTTP status codes:
- `400` Bad Request - Invalid request parameters
- `401` Unauthorized - Missing or invalid token
- `403` Forbidden - Insufficient permissions
- `404` Not Found - Resource not found
- `500` Internal Server Error - Server error

## Rate Limiting

Currently not implemented, but may be added in future versions.

## WebSocket

WebSocket is supported for real-time system status updates.

**Endpoint**: `GET /api/v1/system/ws`

**Authentication**: Pass the JWT token either as a query parameter (`?token=<jwt>`) or via the `Authorization: Bearer <jwt>` header.

**Push interval**: The server sends a snapshot message every 3 seconds and a ping frame every 15 seconds (read deadline is reset on pong).

**Message format** (JSON):

```json
{
  "type": "snapshot",
  "at": "2024-01-01T00:00:00Z",
  "status": {
    "mihomo_status": "running",
    "mihomo_pid": 1234,
    "last_auto_update": { "action": "...", "details": "...", "at": "..." },
    "last_alert":       { "action": "...", "details": "...", "at": "..." }
  },
  "traffic": { "up": 1024, "down": 2048 },
  "mihomo_error": ""
}
```

The `traffic` field is omitted and `mihomo_error` is set when the mihomo API is unreachable.
