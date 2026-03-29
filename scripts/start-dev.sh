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

build_services() {
    log_info "Building microservices..."
    
    cd "$PROJECT_ROOT"
    
    mkdir -p bin
    
    local services=("gateway" "vehicle" "billing" "payment" "admin")
    
    for service in "${services[@]}"; do
        log_info "Building $service service..."
        if [ -f "bin/$service-svc" ] && [ "$1" != "--rebuild" ]; then
            log_success "$service service binary already exists, skipping build"
            continue
        fi
        
        if go build -o "bin/$service-svc" "./cmd/$service" 2>&1; then
            log_success "$service service built successfully"
        else
            log_error "Failed to build $service service"
            exit 1
        fi
        
        # Add delay to avoid resource issues
        sleep 2
    done
    
    log_success "All microservices built successfully"
}

start_service() {
    local service=$1
    local port=$2
    local grpc_port=$3
    local config=$4
    
    log_info "Starting $service service..."
    
    cd "$PROJECT_ROOT"
    
    if lsof -ti:$port >/dev/null 2>&1; then
        log_warning "Port $port is already in use, $service service may already be running"
        return
    fi
    
    mkdir -p logs
    
    # Set environment variables for service discovery
    export SERVICE_NAME=$service
    export SERVICE_PORT=$port
    export GRPC_PORT=$grpc_port
    
    ./bin/$service-svc -conf "$config" > "logs/$service.log" 2>&1 &
    echo $! > "/tmp/smart-park-$service.pid"
    
    sleep 3
    
    if lsof -ti:$port >/dev/null 2>&1; then
        log_success "$service service started on port $port"
    else
        log_error "Failed to start $service service"
        log_info "Check logs at: logs/$service.log"
        tail -20 "logs/$service.log"
    fi
}

start_microservices() {
    log_info "Starting microservices..."
    
    # Start services in order with delays to avoid resource issues
    start_service "admin" "8004" "9004" "./configs"
    sleep 2
    
    start_service "vehicle" "8001" "9001" "./configs"
    sleep 2
    
    start_service "billing" "8002" "9002" "./configs"
    sleep 2
    
    start_service "payment" "8003" "9003" "./configs"
    sleep 2
    
    start_service "gateway" "8000" "9000" "./configs"
    
    log_success "All microservices started"
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
    log_info "Microservices:"
    local services=("gateway:8000" "vehicle:8001" "billing:8002" "payment:8003" "admin:8004")
    for svc in "${services[@]}"; do
        IFS=':' read -r name port <<< "$svc"
        if lsof -ti:$port >/dev/null 2>&1; then
            log_success "$name service on port $port"
        else
            log_error "$name service on port $port"
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
    log_success "  Development Environment Ready!"
    log_success "========================================="
    echo ""
    log_info "Service URLs:"
    echo "  - Frontend:    http://localhost:3000"
    echo "  - Gateway:     http://localhost:8000"
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
    build_services "$@"
    start_infrastructure
    start_microservices
    start_frontend
    
    show_status
}

main "$@"
