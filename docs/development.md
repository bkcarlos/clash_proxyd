# Development Guide

This guide covers how to set up a development environment and contribute to the proxyd project.

## Prerequisites

### Backend
- Go 1.22 or higher
- GCC/Clang (for CGO and sqlite3)
- SQLite3 development headers
- Make (optional, for using Makefile)

### Frontend
- Node.js 20 or higher
- npm or yarn

## Setting Up Development Environment

### 1. Clone the Repository

```bash
git clone https://github.com/clash-proxyd/proxyd.git
cd proxyd
```

### 2. Backend Setup

```bash
# Install Go dependencies
go mod download

# Initialize database
go run cmd/proxyd/main.go -init-db -c config.example.yaml

# Run backend
go run cmd/proxyd/main.go -c config.example.yaml
```

The backend will start on `http://localhost:8080`.

### 3. Frontend Setup

```bash
cd web-ui

# Install dependencies
npm install

# Run development server
npm run dev
```

The frontend will start on `http://localhost:5173`.

## Project Structure

```
proxyd/
├── cmd/proxyd/          # Application entry point
├── internal/            # Private application code
│   ├── app/            # Application initialization
│   ├── api/            # HTTP API handlers
│   ├── auth/           # Authentication & authorization
│   ├── core/           # mihomo process management
│   ├── health/         # Health checks
│   ├── logx/           # Logging utilities
│   ├── parser/         # YAML parsing
│   ├── policy/         # Policy generation
│   ├── renderer/       # Configuration rendering
│   ├── scheduler/      # Scheduled tasks
│   ├── source/         # Subscription fetching
│   ├── store/          # Database operations
│   └── types/          # Common types
├── pkg/                # Public packages
│   ├── config/         # Configuration management
│   └── util/           # Utilities
├── web-ui/             # Vue frontend
│   ├── src/
│   │   ├── api/        # API clients
│   │   ├── components/ # Vue components
│   │   ├── layouts/    # Layout components
│   │   ├── router/     # Vue Router
│   │   ├── stores/     # Pinia stores
│   │   └── views/      # Page components
│   ├── package.json
│   └── vite.config.ts
├── deployments/        # Deployment configurations
├── scripts/            # Utility scripts
├── config.example.yaml # Example configuration
├── Makefile            # Build automation
├── Dockerfile          # Docker image
└── go.mod              # Go dependencies
```

## Development Workflow

### Making Changes

1. **Backend Changes**
   - Edit code in `internal/` or `pkg/`
   - Restart the Go process
   - Changes are reflected immediately

2. **Frontend Changes**
   - Edit code in `web-ui/src/`
   - Vite hot-reloads automatically
   - Browser updates instantly

### Running Tests

```bash
# Backend tests
go test ./...

# Frontend tests (when implemented)
cd web-ui
npm test
```

### Building

```bash
# Build everything
make build

# Build backend only
make build-backend

# Build frontend only
make build-frontend
```

## Code Style

### Go
- Follow standard Go conventions
- Use `gofmt` for formatting
- Document exported functions
- Keep functions small and focused

### Vue/TypeScript
- Use Composition API with `<script setup>`
- Prefer Pinia for state management
- Use TypeScript for type safety
- Follow Vue 3 best practices

## API Development

### Adding New Endpoints

1. Define the handler in `internal/api/`
2. Register the route in `internal/api/router.go`
3. Add TypeScript types in `web-ui/src/api/`
4. Create store methods in `web-ui/src/stores/`

### Example

```go
// internal/api/example.go
func (h *Handler) GetExample(c *gin.Context) {
    h.respondJSON(c, http.StatusOK, gin.H{"message": "example"})
}
```

```typescript
// web-ui/src/api/example.ts
export const getExample = (): Promise<any> => {
  return request({
    url: '/example',
    method: 'GET'
  })
}
```

## Database Migrations

When modifying the database schema:

1. Update `internal/store/schema.sql`
2. Update corresponding store methods in `internal/store/`
3. Test with `-init-db` flag

## Debugging

### Backend

```bash
# Enable debug logging
# Set log_level: debug in config.yaml

# Use Delve debugger
dlv debug cmd/proxyd/main.go -- -c config.example.yaml
```

### Frontend

```bash
# Vue DevTools browser extension is recommended
# Logs appear in browser console
```

## Common Issues

### CGO Errors
```bash
# Ensure GCC and sqlite3-dev are installed
sudo apt install build-essential libsqlite3-dev
```

### Port Already in Use
```bash
# Find process using port 8080
lsof -i :8080
# Kill it
kill -9 <PID>
```

### Frontend Proxy Issues
- Check `vite.config.ts` proxy configuration
- Verify backend is running on port 8080

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Write tests if applicable
5. Submit a pull request

## Useful Commands

```bash
# Format Go code
go fmt ./...

# Run Go vet
go vet ./...

# Tidy Go modules
go mod tidy

# Update frontend dependencies
cd web-ui && npm update

# Clean build artifacts
make clean
```
