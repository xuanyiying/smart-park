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

start_infrastructure() {
    log_info "Starting infrastructure services..."
    
    cd "$PROJECT_ROOT"
    
    if docker ps | grep -q postgres; then
        log_success "PostgreSQL is already running"
    else
        log_warning "PostgreSQL is not running"
        log_info "Please start PostgreSQL manually"
    fi
    
    if docker ps | grep -q redis; then
        log_success "Redis is already running"
    else
        log_warning "Redis is not running"
        log_info "Please start Redis manually"
    fi
    
    if docker ps | grep -q etcd; then
        log_success "Etcd is already running"
    else
        log_info "Starting Etcd..."
        docker-compose -f deploy/docker-compose.infra.yml up -d etcd 2>/dev/null || log_warning "Etcd startup skipped"
    fi
    
    if docker ps | grep -q jaeger; then
        log_success "Jaeger is already running"
    else
        log_info "Starting Jaeger..."
        docker-compose -f deploy/docker-compose.infra.yml up -d jaeger 2>/dev/null || log_warning "Jaeger startup skipped"
    fi
    
    log_success "Infrastructure services started"
}

build_all_in_one() {
    log_info "Building all-in-one service..."
    
    cd "$PROJECT_ROOT"
    
    mkdir -p bin
    
    if [ -f "bin/all-in-one" ]; then
        log_success "All-in-one binary already exists, skipping build"
        log_info "Use --rebuild to force rebuild"
        return
    fi
    
    go build -o bin/all-in-one ./cmd/all-in-one
    
    log_success "All-in-one service built successfully"
}

rebuild_all_in_one() {
    log_info "Rebuilding all-in-one service..."
    
    cd "$PROJECT_ROOT"
    
    rm -f bin/all-in-one
    go build -o bin/all-in-one ./cmd/all-in-one
    
    log_success "All-in-one service rebuilt successfully"
}

start_all_in_one() {
    log_info "Starting all-in-one service..."
    
    cd "$PROJECT_ROOT"
    
    mkdir -p logs
    
    if lsof -ti:8000 >/dev/null 2>&1; then
        log_warning "Port 8000 is already in use"
        read -p "Kill existing process and restart? (y/N): " kill_existing
        if [[ $kill_existing =~ ^[Yy]$ ]]; then
            lsof -ti:8000 | xargs kill -9 2>/dev/null
            sleep 2
        else
            log_error "Cannot start all-in-one service, port 8000 is occupied"
            exit 1
        fi
    fi
    
    ./bin/all-in-one -conf ./configs/dev.yaml > logs/all-in-one.log 2>&1 &
    echo $! > /tmp/smart-park-all-in-one.pid
    
    sleep 3
    
    if lsof -ti:8000 >/dev/null 2>&1; then
        log_success "All-in-one service started on port 8000"
    else
        log_error "Failed to start all-in-one service"
        log_info "Check logs at: logs/all-in-one.log"
        tail -20 logs/all-in-one.log
        exit 1
    fi
}

start_frontend() {
    log_info "Starting frontend..."
    
    cd "$PROJECT_ROOT/web"
    
    if lsof -ti:3000 >/dev/null 2>&1; then
        log_warning "Port 3000 is already in use, frontend may already be running"
        return
    fi
    
    log_info "Starting frontend on port 3000..."
    pnpm run dev > ../logs/frontend.log 2>&1 &
    echo $! > /tmp/smart-park-frontend.pid
    
    sleep 3
    
    log_success "Frontend started"
}

show_status() {
    echo ""
    log_info "========================================="
    log_info "  Development Environment Status"
    log_info "========================================="
    echo ""
    
    log_info "Infrastructure:"
    docker ps | grep -E "postgres|redis|etcd|jaeger" | awk '{print "  - " $1 ": " $NF}' || log_warning "No infrastructure services found"
    
    echo ""
    log_info "All-in-One Service:"
    if lsof -ti:8000 >/dev/null 2>&1; then
        log_success "Port 8000 is active (All-in-One Service)"
    else
        log_error "Port 8000 is not active"
    fi
    
    echo ""
    log_info "Frontend:"
    if lsof -ti:3000 >/dev/null 2>&1; then
        log_success "Frontend is running on http://localhost:3000"
    else
        log_error "Frontend is not running"
    fi
    
    echo ""
    log_success "========================================="
    log_success "  Development Environment Ready!"
    log_success "========================================="
    echo ""
    log_info "Service URLs:"
    echo "  - Frontend:    http://localhost:3000"
    echo "  - API:         http://localhost:8000"
    echo "  - Jaeger UI:   http://localhost:16686"
    echo ""
    log_info "API Endpoints:"
    echo "  - Device API:  http://localhost:8000/api/v1/device"
    echo "  - Billing API: http://localhost:8000/api/v1/billing"
    echo "  - Payment API: http://localhost:8000/api/v1/pay"
    echo "  - Admin API:   http://localhost:8000/api/v1/admin"
    echo ""
    log_info "Logs: $PROJECT_ROOT/logs/"
    echo ""
    log_info "To stop services, run: ./scripts/stop-dev.sh"
    echo ""
}

main() {
    log_info "Starting development environment..."
    
    check_dependencies
    
    if [ "$1" = "--rebuild" ]; then
        rebuild_all_in_one
    else
        build_all_in_one
    fi
    
    start_infrastructure
    start_all_in_one
    start_frontend
    
    show_status
}

main "$@"
