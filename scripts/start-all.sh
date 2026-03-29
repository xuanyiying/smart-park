#!/bin/bash

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_dependencies() {
    log_info "Checking dependencies..."
    
    local missing_deps=()
    
    command -v go >/dev/null 2>&1 || missing_deps+=("go")
    command -v docker >/dev/null 2>&1 || missing_deps+=("docker")
    command -v pnpm >/dev/null 2>&1 || missing_deps+=("pnpm")
    
    if [ ${#missing_deps[@]} -gt 0 ]; then
        log_error "Missing dependencies: ${missing_deps[*]}"
        exit 1
    fi
    
    log_success "All dependencies found"
}

build_services() {
    log_info "Building backend services..."
    
    cd "$PROJECT_ROOT"
    
    log_info "Building gateway service..."
    go build -o bin/gateway ./cmd/gateway
    
    log_info "Building vehicle service..."
    go build -o bin/vehicle ./cmd/vehicle
    
    log_info "Building billing service..."
    go build -o bin/billing ./cmd/billing
    
    log_info "Building payment service..."
    go build -o bin/payment ./cmd/payment
    
    log_info "Building admin service..."
    go build -o bin/admin ./cmd/admin
    
    log_success "All services built successfully"
}

start_infrastructure() {
    log_info "Starting infrastructure services..."
    
    cd "$PROJECT_ROOT"
    
    if docker ps | grep -q postgres; then
        log_success "PostgreSQL is already running"
    else
        log_warning "PostgreSQL is not running"
        log_info "Please start PostgreSQL manually or use existing container"
    fi
    
    if docker ps | grep -q redis; then
        log_success "Redis is already running"
    else
        log_warning "Redis is not running"
        log_info "Please start Redis manually or use existing container"
    fi
    
    if docker ps | grep -q etcd; then
        log_success "Etcd is already running"
    else
        log_info "Starting Etcd..."
        docker-compose -f deploy/docker-compose.infra.yml up -d etcd
    fi
    
    if docker ps | grep -q jaeger; then
        log_success "Jaeger is already running"
    else
        log_info "Starting Jaeger..."
        docker-compose -f deploy/docker-compose.infra.yml up -d jaeger
    fi
    
    log_success "Infrastructure services started"
}

start_backend_services() {
    log_info "Starting backend services..."
    
    cd "$PROJECT_ROOT"
    
    mkdir -p logs
    
    if lsof -ti:8001 >/dev/null 2>&1; then
        log_warning "Port 8001 is already in use, skipping vehicle service"
    else
        log_info "Starting vehicle service on port 8001..."
        ./bin/vehicle -conf ./configs/vehicle.yaml > logs/vehicle.log 2>&1 &
        echo $! > /tmp/smart-park-vehicle.pid
        sleep 2
    fi
    
    if lsof -ti:8002 >/dev/null 2>&1; then
        log_warning "Port 8002 is already in use, skipping billing service"
    else
        log_info "Starting billing service on port 8002..."
        ./bin/billing -conf ./configs/billing.yaml > logs/billing.log 2>&1 &
        echo $! > /tmp/smart-park-billing.pid
        sleep 2
    fi
    
    if lsof -ti:8003 >/dev/null 2>&1; then
        log_warning "Port 8003 is already in use, skipping payment service"
    else
        log_info "Starting payment service on port 8003..."
        ./bin/payment -conf ./configs/payment.yaml > logs/payment.log 2>&1 &
        echo $! > /tmp/smart-park-payment.pid
        sleep 2
    fi
    
    if lsof -ti:8004 >/dev/null 2>&1; then
        log_warning "Port 8004 is already in use, skipping admin service"
    else
        log_info "Starting admin service on port 8004..."
        ./bin/admin -conf ./configs/admin.yaml > logs/admin.log 2>&1 &
        echo $! > /tmp/smart-park-admin.pid
        sleep 2
    fi
    
    if lsof -ti:8000 >/dev/null 2>&1; then
        log_warning "Port 8000 is already in use, skipping gateway"
    else
        log_info "Starting gateway on port 8000..."
        ./bin/gateway -conf ./configs/gateway.yaml > logs/gateway.log 2>&1 &
        echo $! > /tmp/smart-park-gateway.pid
        sleep 2
    fi
    
    log_success "Backend services started"
}

start_frontend() {
    log_info "Starting frontend..."
    
    cd "$PROJECT_ROOT/web"
    
    if lsof -ti:3000 >/dev/null 2>&1; then
        log_warning "Port 3000 is already in use, skipping frontend"
    else
        log_info "Starting frontend on port 3000..."
        pnpm run dev > ../logs/frontend.log 2>&1 &
        echo $! > /tmp/smart-park-frontend.pid
        sleep 3
    fi
    
    log_success "Frontend started"
}

show_status() {
    echo ""
    log_info "Checking services status..."
    echo ""
    
    log_info "Infrastructure:"
    docker ps | grep -E "postgres|redis|etcd|jaeger" | awk '{print "  - " $1 ": " $NF}' || log_warning "No infrastructure services found"
    
    echo ""
    log_info "Backend Services:"
    for port in 8000 8001 8002 8003 8004; do
        if lsof -ti:$port >/dev/null 2>&1; then
            log_success "Port $port is active"
        else
            log_error "Port $port is not active"
        fi
    done
    
    echo ""
    log_info "Frontend:"
    if lsof -ti:3000 >/dev/null 2>&1; then
        log_success "Frontend is running on http://localhost:3000"
    else
        log_error "Frontend is not running"
    fi
    
    echo ""
    log_success "========================================="
    log_success "  All Services Started Successfully!"
    log_success "========================================="
    echo ""
    log_info "Service URLs:"
    echo "  - Frontend:    http://localhost:3000"
    echo "  - Gateway:     http://localhost:8000"
    echo "  - Vehicle:     http://localhost:8001"
    echo "  - Billing:     http://localhost:8002"
    echo "  - Payment:     http://localhost:8003"
    echo "  - Admin:       http://localhost:8004"
    echo ""
    log_info "API Endpoints:"
    echo "  - Device API:  http://localhost:8000/api/v1/device"
    echo "  - Billing API: http://localhost:8000/api/v1/billing"
    echo "  - Payment API: http://localhost:8000/api/v1/pay"
    echo "  - Admin API:   http://localhost:8000/api/v1/admin"
    echo ""
    log_info "Logs are available in: $PROJECT_ROOT/logs/"
    echo ""
    log_info "To stop all services, run: ./scripts/stop-all.sh"
    echo ""
}

main() {
    check_dependencies
    
    if [ "$1" = "--skip-build" ]; then
        log_info "Skipping build step..."
    else
        build_services
    fi
    
    start_infrastructure
    start_backend_services
    start_frontend
    
    show_status
}

main "$@"
