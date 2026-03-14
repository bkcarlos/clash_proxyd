.PHONY: all build build-backend build-frontend install clean test run dev-backend dev-frontend init-db install-service uninstall-service docker-build docker-run mihomo-start mihomo-stop mihomo-restart mihomo-status help

# Variables
BINARY_NAME=proxyd
BUILD_DIR=build
CMD_DIR=cmd/proxyd
WEB_UI_DIR=web-ui
PREFIX=/opt/proxyd

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# API parameters (for mihomo control via proxyd API)
API_BASE?=http://127.0.0.1:8080/api/v1
ADMIN_USERNAME?=admin
ADMIN_PASSWORD?=$$2a$$10$$YourBCryptHashHere

all: build

## build: Build both backend and frontend
build: build-backend build-frontend

## build-backend: Build Go backend
build-backend:
	mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) ./$(CMD_DIR)

## build-frontend: Build Vue frontend
build-frontend:
	cd $(WEB_UI_DIR) && $(NPM) install
	cd $(WEB_UI_DIR) && $(NPM) run build

## install: Install proxyd to system
install: build
	install -d $(PREFIX)/bin
	install -d $(PREFIX)/data/db
	install -d $(PREFIX)/logs
	install -d $(PREFIX)/web-ui
	install -m 755 $(BUILD_DIR)/$(BINARY_NAME) $(PREFIX)/bin/
	cp -r $(WEB_UI_DIR)/dist/* $(PREFIX)/web-ui/
	@if [ ! -f $(PREFIX)/config.yaml ]; then \
		cp config.example.yaml $(PREFIX)/config.yaml; \
		echo "Installed config.example.yaml to $(PREFIX)/config.yaml"; \
		echo "Please edit it before running proxyd"; \
	fi

## clean: Clean build artifacts
clean:
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	rm -rf $(WEB_UI_DIR)/dist
	rm -rf $(WEB_UI_DIR)/node_modules

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

## install-service: Install systemd service
install-service:
	@echo "Installing systemd service..."
	@sed "s|/opt/proxyd|$(PREFIX)|g" deployments/systemd/proxyd.service > /tmp/proxyd.service
	install -m 644 /tmp/proxyd.service /etc/systemd/system/
	systemctl daemon-reload
	@echo "Service installed. Enable with: systemctl enable proxyd"
	@echo "Start with: systemctl start proxyd"

## uninstall-service: Uninstall systemd service
uninstall-service:
	systemctl stop proxyd || true
	systemctl disable proxyd || true
	rm -f /etc/systemd/system/proxyd.service
	systemctl daemon-reload

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
