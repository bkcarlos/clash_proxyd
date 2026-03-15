.PHONY: all build build-backend build-frontend install deploy install-service uninstall-service clean test run dev-backend dev-frontend init-db docker-build docker-run mihomo-start mihomo-stop mihomo-restart mihomo-status help

# Variables
BINARY_NAME=proxyd
BUILD_DIR=build
CMD_DIR=cmd/proxyd
WEB_UI_DIR=web-ui
PREFIX=/opt/proxyd
SERVICE_USER=nobody

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
NPM=npm

# API parameters (for mihomo control via proxyd API)
API_BASE?=http://127.0.0.1:8080/api/v1
ADMIN_USERNAME?=admin
ADMIN_PASSWORD?=$$2a$$10$$YourBCryptHashHere

all: build

## build: Build frontend then backend (without embedded web UI)
build: build-frontend build-backend

## build-all: Build frontend + embed it into the binary (-tags webui)
build-all: build-frontend embed-frontend build-backend-web

## build-backend: Build Go backend (no embedded web UI)
build-backend:
	mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) ./$(CMD_DIR)

## build-backend-web: Build Go backend with embedded web UI (-tags webui)
build-backend-web:
	mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 $(GOBUILD) -tags webui -o $(BUILD_DIR)/$(BINARY_NAME) ./$(CMD_DIR)

## copy-assets: Copy mihomo binary and Country.mmdb into internal/assets/ for go:embed
## Requires: build/mihomo and build/data/mihomo/Country.mmdb to exist
copy-assets:
	@test -f $(BUILD_DIR)/mihomo || (echo "ERROR: $(BUILD_DIR)/mihomo not found. Build or download mihomo first." && exit 1)
	@test -f $(BUILD_DIR)/data/mihomo/Country.mmdb || (echo "ERROR: $(BUILD_DIR)/data/mihomo/Country.mmdb not found." && exit 1)
	cp $(BUILD_DIR)/mihomo internal/assets/mihomo
	cp $(BUILD_DIR)/data/mihomo/Country.mmdb internal/assets/Country.mmdb
	@echo "Assets copied to internal/assets/"

## build-bundle: Build fully self-contained binary (web UI + mihomo + MMDB embedded)
build-bundle: build-frontend embed-frontend copy-assets
	mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 $(GOBUILD) -tags webui,bundle -o $(BUILD_DIR)/$(BINARY_NAME) ./$(CMD_DIR)
	@echo "Bundle binary built: $(BUILD_DIR)/$(BINARY_NAME)"

## build-frontend: Build Vue frontend
build-frontend:
	cd $(WEB_UI_DIR) && $(NPM) install
	cd $(WEB_UI_DIR) && $(NPM) run build

## embed-frontend: Copy Vue dist into internal/webui/dist for go:embed
embed-frontend:
	rm -rf internal/webui/dist
	cp -r $(WEB_UI_DIR)/dist internal/webui/dist

## install: Install binary + config to PREFIX (default /opt/proxyd), no service setup
install: build
	install -d $(PREFIX)/bin
	install -d $(PREFIX)/data/db
	install -d $(PREFIX)/data/mihomo
	install -d $(PREFIX)/data/generated
	install -d $(PREFIX)/data/cache
	install -d $(PREFIX)/logs
	install -m 755 $(BUILD_DIR)/$(BINARY_NAME) $(PREFIX)/bin/
	@if [ ! -f $(PREFIX)/config.yaml ]; then \
		cp config.example.yaml $(PREFIX)/config.yaml; \
		echo "Installed config.example.yaml to $(PREFIX)/config.yaml"; \
		echo "Please edit it before running proxyd"; \
	fi

## deploy: Full one-step deploy: build + install binary + set up systemd service (requires root)
deploy: build-all
	sudo INSTALL_DIR=$(PREFIX) scripts/install.sh $(BUILD_DIR)/$(BINARY_NAME)

## clean: Clean build artifacts
clean:
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -rf $(WEB_UI_DIR)/dist
	rm -rf $(WEB_UI_DIR)/node_modules
	rm -rf internal/webui/dist
	rm -f internal/assets/mihomo internal/assets/Country.mmdb

## test: Run tests
test:
	$(GOTEST) -v ./...

## run: Run proxyd in development mode
run:
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) ./$(CMD_DIR)
	$(BUILD_DIR)/$(BINARY_NAME) -c config.example.yaml

## dev: Run in development (requires separate terminal for frontend)
dev-backend:
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) ./$(CMD_DIR)
	$(BUILD_DIR)/$(BINARY_NAME) -c config.example.yaml

## dev-frontend: Run frontend dev server
dev-frontend:
	cd $(WEB_UI_DIR) && $(NPM) run dev

## init-db: Initialize database
init-db:
	$(BUILD_DIR)/$(BINARY_NAME) -c config.example.yaml -init-db

## install-service: Install (or update) only the systemd unit file; requires root
install-service:
	@echo "Installing systemd service..."
	@sed \
		-e "s|/opt/proxyd|$(PREFIX)|g" \
		-e "s|User=nobody|User=$(SERVICE_USER)|g" \
		deployments/systemd/proxyd.service \
		> /tmp/proxyd.service
	install -m 644 /tmp/proxyd.service /etc/systemd/system/proxyd.service
	rm /tmp/proxyd.service
	systemctl daemon-reload
	@echo "Service file installed: /etc/systemd/system/proxyd.service"
	@echo "Enable:  systemctl enable proxyd"
	@echo "Start:   systemctl start proxyd"

## uninstall-service: Stop, disable, and remove the systemd unit file
uninstall-service:
	systemctl stop proxyd    || true
	systemctl disable proxyd || true
	rm -f /etc/systemd/system/proxyd.service
	systemctl daemon-reload
	@echo "Service removed."

## docker-build: Build Docker image
docker-build:
	docker build -t proxyd:latest .

## docker-run: Run Docker container
docker-run:
	docker run -d \
		--name proxyd \
		-p 8080:8080 \
		-v $(PWD)/data:/opt/proxyd/data \
		-v $(PWD)/logs:/opt/proxyd/logs \
		proxyd:latest

## mihomo-start: Start mihomo via proxyd API
mihomo-start:
	@TOKEN=$$(curl -sS -X POST "$(API_BASE)/auth/login" -H "Content-Type: application/json" -d '{"username":"$(ADMIN_USERNAME)","password":"$(ADMIN_PASSWORD)"}' | python3 -c 'import sys,json; d=json.load(sys.stdin); print(d.get("token",""))'); \
	if [ -z "$$TOKEN" ]; then echo "Login failed"; exit 1; fi; \
	curl -sS -X POST "$(API_BASE)/proxy/mihomo/start" -H "Authorization: Bearer $$TOKEN"

## mihomo-stop: Stop mihomo via proxyd API
mihomo-stop:
	@TOKEN=$$(curl -sS -X POST "$(API_BASE)/auth/login" -H "Content-Type: application/json" -d '{"username":"$(ADMIN_USERNAME)","password":"$(ADMIN_PASSWORD)"}' | python3 -c 'import sys,json; d=json.load(sys.stdin); print(d.get("token",""))'); \
	if [ -z "$$TOKEN" ]; then echo "Login failed"; exit 1; fi; \
	curl -sS -X POST "$(API_BASE)/proxy/mihomo/stop" -H "Authorization: Bearer $$TOKEN"

## mihomo-restart: Restart mihomo via proxyd API
mihomo-restart:
	@TOKEN=$$(curl -sS -X POST "$(API_BASE)/auth/login" -H "Content-Type: application/json" -d '{"username":"$(ADMIN_USERNAME)","password":"$(ADMIN_PASSWORD)"}' | python3 -c 'import sys,json; d=json.load(sys.stdin); print(d.get("token",""))'); \
	if [ -z "$$TOKEN" ]; then echo "Login failed"; exit 1; fi; \
	curl -sS -X POST "$(API_BASE)/proxy/mihomo/restart" -H "Authorization: Bearer $$TOKEN"

## mihomo-status: Show mihomo runtime status via proxyd API
mihomo-status:
	@TOKEN=$$(curl -sS -X POST "$(API_BASE)/auth/login" -H "Content-Type: application/json" -d '{"username":"$(ADMIN_USERNAME)","password":"$(ADMIN_PASSWORD)"}' | python3 -c 'import sys,json; d=json.load(sys.stdin); print(d.get("token",""))'); \
	if [ -z "$$TOKEN" ]; then echo "Login failed"; exit 1; fi; \
	curl -sS "$(API_BASE)/system/status" -H "Authorization: Bearer $$TOKEN"

## help: Show this help message
help:
	@echo "Usage: make [target]"
	@echo ""
	@echo "Available targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'
