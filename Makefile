# ==================================================================================== #
# HELPERS
# ==================================================================================== #

## help: print this help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

.PHONY: confirm
confirm:
	@echo -n 'Are you sure? [y/N] ' && read ans && [ $${ans:-N} = y ]

# ==================================================================================== #
# DEVELOPMENT
# ==================================================================================== #

## run/api: run the cmd/api application
.PHONY: run/api
run/api:
	go run ./cmd/api

## build/api: build the cmd/api application
.PHONY: build/api
build/api:
	@echo 'Building cmd/api...'
	@mkdir -p ./bin/linux_amd64
	go build -ldflags="-s -w -X 'main.buildTime=$$(date -u +"%Y-%m-%d %H:%M:%S %Z")' -X 'main.version=$$(git describe --always --dirty --tags 2>/dev/null || echo "unknown")'" -o=./bin/api ./cmd/api
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X 'main.buildTime=$$(date -u +"%Y-%m-%d %H:%M:%S %Z")' -X 'main.version=$$(git describe --always --dirty --tags 2>/dev/null || echo "unknown")'" -o=./bin/linux_amd64/api ./cmd/api

# ==================================================================================== #
# QUALITY CONTROL
# ==================================================================================== #

## audit: tidy dependencies and format, vet and test all code
.PHONY: audit
audit:
	@echo 'Tidying and verifying module dependencies...'
	go mod tidy
	go mod verify
	@echo 'Formatting code...'
	go fmt ./...
	@echo 'Vetting code...'
	go vet ./...
	staticcheck -checks="all,-U1000" ./...
	@echo 'Running tests...'
	go test -race -cover -vet=off ./...

## vendor: tidy and vendor dependencies
.PHONY: vendor
vendor:
	@echo 'Tidying and verifying module dependencies...'
	go mod tidy
	go mod verify
	@echo 'Vendoring dependencies...'
	go mod vendor

# ==================================================================================== #
# BUILD
# ==================================================================================== #

## build: build the application
.PHONY: build
build: build/api

## run: run the application
.PHONY: run
run: run/api

## test: run all tests
.PHONY: test
test:
	@echo 'Running tests...'
	go test -race -cover -vet=off ./...

# ==================================================================================== #
# DOCKER & INFRASTRUCTURE
# ==================================================================================== #

## docker/up: start all services with docker-compose
.PHONY: docker/up
docker/up:
	@echo 'Starting FizzBuzz stack with Docker Compose...'
	docker compose up -d

## docker/down: stop all services and remove containers
.PHONY: docker/down
docker/down:
	@echo 'Stopping FizzBuzz stack...'
	docker compose down

## docker/logs: show logs for all services
.PHONY: docker/logs
docker/logs:
	docker compose logs -f

## docker/logs/api: show logs for API service only
.PHONY: docker/logs/api
docker/logs/api:
	docker compose logs -f fizzbuzz-api

## docker/logs/db: show logs for PostgreSQL service only
.PHONY: docker/logs/db
docker/logs/db:
	docker compose logs -f postgres

## docker/build: build the Docker image
.PHONY: docker/build
docker/build:
	@echo 'Building Docker image...'
	docker compose build fizzbuzz-api

## docker/rebuild: rebuild and restart the application
.PHONY: docker/rebuild
docker/rebuild: docker/build
	@echo 'Rebuilding and restarting API...'
	docker compose up -d --force-recreate fizzbuzz-api

## docker/clean: remove all containers, volumes and images
.PHONY: docker/clean
docker/clean: confirm
	@echo 'Removing all Docker resources...'
	docker compose down -v --remove-orphans
	docker system prune -af

## docker/db/shell: connect to PostgreSQL shell
.PHONY: docker/db/shell
docker/db/shell:
	@echo 'Connecting to PostgreSQL...'
	docker compose exec postgres psql -U fizzbuzz_user -d fizzbuzz

## docker/dev: start with development tools (includes pgAdmin)
.PHONY: docker/dev
docker/dev:
	@echo 'Starting development environment...'
	docker compose --profile dev-tools up -d

## docker/health: check health status of all services
.PHONY: docker/health
docker/health:
	@echo 'Checking service health...'
	docker compose ps

## docker/reset: completely reset the environment
.PHONY: docker/reset
docker/reset: docker/clean docker/up
	@echo 'Environment reset complete!'

# ==================================================================================== #
# QUICK START
# ==================================================================================== #

## start: quick start - build and run everything
.PHONY: start
start: docker/up
	@echo ''
	@echo 'FizzBuzz API is starting up...'
	@echo 'API will be available at: http://localhost:4000'
	@echo 'Health check: http://localhost:4000/v1/healthcheck'
	@echo 'PostgreSQL: localhost:5432 (fizzbuzz_user/fizzbuzz_pass)'
	@echo ''
	@echo 'Run "make docker/logs" to see startup logs'
	@echo 'Run "make docker/health" to check service status'

## stop: stop all services
.PHONY: stop
stop: docker/down

## dev: start development environment with tools
.PHONY: dev
dev: docker/dev
	@echo ''
	@echo 'Development environment started!'
	@echo 'API: http://localhost:4000'
	@echo 'pgAdmin: http://localhost:5050 (admin@fizzbuzz.local/admin123)'
	@echo ''