# Gotel Example Makefile

.PHONY: all build run-server run-client test proto docker-up docker-down clean help

# Default target
all: proto build

# Build all binaries
build: proto
	@echo "🔨 Building server..."
	@cd server && go build -o ../bin/server main.go
	@echo "🔨 Building client..."
	@cd client && go build -o ../bin/client main.go
	@echo "✅ Build complete"

# Generate protobuf files
proto:
	@echo "🔧 Generating protobuf files..."
	@protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/example.proto
	@echo "✅ Protobuf files generated"

# Run server
run-server: build
	@echo "🚀 Starting server..."
	@./bin/server

# Run client
run-client: build
	@echo "🔍 Starting client..."
	@./bin/client

# Run both server and client (server in background)
demo: build
	@echo "🎬 Starting demo..."
	@echo "🚀 Starting server in background..."
	@./bin/server & echo $$! > server.pid
	@echo "⏳ Waiting for server to start..."
	@sleep 3
	@echo "🔍 Running client..."
	@./bin/client
	@echo "🛑 Stopping server..."
	@kill `cat server.pid` && rm server.pid
	@echo "🎉 Demo complete!"

# Start infrastructure (Jaeger, Redis, PostgreSQL)
docker-up:
	@echo "🐳 Starting infrastructure..."
	@docker-compose up -d
	@echo "⏳ Waiting for services to be ready..."
	@sleep 10
	@echo "✅ Infrastructure ready"
	@echo "📊 Jaeger UI: http://localhost:16686"
	@echo "🔴 Redis: localhost:6379"
	@echo "🐘 PostgreSQL: localhost:5432"

# Stop infrastructure
docker-down:
	@echo "🛑 Stopping infrastructure..."
	@docker-compose down
	@echo "✅ Infrastructure stopped"

# Clean up
clean:
	@echo "🧹 Cleaning up..."
	@rm -rf bin/
	@rm -f server.pid
	@echo "✅ Clean complete"

# Test setup
test: docker-up
	@echo "🧪 Running integration test..."
	@sleep 5
	@$(MAKE) demo
	@echo "✅ Integration test complete"

# Install dependencies
deps:
	@echo "📦 Installing dependencies..."
	@cd server && go mod tidy
	@cd client && go mod tidy
	@cd shared && go mod tidy
	@echo "✅ Dependencies installed"

# Development setup
setup: init proto deps docker-up
	@echo "🎉 Development environment setup complete!"
	@echo ""
	@echo "📋 Next steps:"
	@echo "  1. Run 'make demo' to test everything"
	@echo "  2. Run 'make run-server' in one terminal"
	@echo "  3. Run 'make run-client' in another terminal"
	@echo "  4. View traces at http://localhost:16686"
	@echo ""

# Help
help:
	@echo "🔧 Gotel Example - Available commands:"
	@echo ""
	@echo "🏗️  Build & Generate:"
	@echo "  make build      - Build server and client binaries"
	@echo "  make proto      - Generate protobuf Go files"
	@echo "  make init       - Initialize Go modules"
	@echo "  make deps       - Install Go dependencies"
	@echo ""
	@echo "🚀 Run:"
	@echo "  make run-server - Start the server"
	@echo "  make run-client - Start the client"
	@echo "  make demo       - Run complete demo (server + client)"
	@echo ""
	@echo "🐳 Infrastructure:"
	@echo "  make docker-up  - Start Jaeger, Redis, PostgreSQL"
	@echo "  make docker-down- Stop infrastructure"
	@echo ""
	@echo "🧪 Testing:"
	@echo "  make test       - Run integration test"
	@echo "  make setup      - Complete development setup"
	@echo ""
	@echo "🧹 Maintenance:"
	@echo "  make clean      - Clean build artifacts"
	@echo "  make help       - Show this help"
	@echo ""
	@echo "🌐 URLs:"
	@echo "  http://localhost:8080/health - Server health check"
	@echo "  http://localhost:16686       - Jaeger UI"