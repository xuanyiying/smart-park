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

stop_service() {
    local service=$1
    local port=$2
    
    log_info "Stopping $service service..."
    
    pid_file="/tmp/smart-park-$service.pid"
    if [ -f "$pid_file" ]; then
        pid=$(cat "$pid_file")
        if kill -0 "$pid" 2>/dev/null; then
            log_info "Stopping $service service (PID: $pid)..."
            kill "$pid" 2>/dev/null || log_warning "Failed to stop $service service"
        else
            log_warning "$service service is not running"
        fi
        rm -f "$pid_file"
    else
        log_warning "No PID file found for $service service"
    fi
    
    if lsof -ti:$port >/dev/null 2>&1; then
        log_info "Killing process on port $port..."
        lsof -ti:$port | xargs kill -9 2>/dev/null || log_warning "Failed to kill process on port $port"
    fi
    
    log_success "$service service stopped"
}

stop_microservices() {
    log_info "Stopping microservices..."
    
    local services=("gateway:8000" "vehicle:8001" "billing:8002" "payment:8003" "admin:8004")
    
    for svc in "${services[@]}"; do
        IFS=':' read -r name port <<< "$svc"
        stop_service "$name" "$port"
    done
    
    log_success "All microservices stopped"
}

stop_frontend() {
    log_info "Stopping frontend..."
    
    pid_file="/tmp/smart-park-frontend.pid"
    if [ -f "$pid_file" ]; then
        pid=$(cat "$pid_file")
        if kill -0 "$pid" 2>/dev/null; then
            log_info "Stopping frontend (PID: $pid)..."
            kill "$pid" 2>/dev/null || log_warning "Failed to stop frontend"
        else
            log_warning "Frontend is not running"
        fi
        rm -f "$pid_file"
    else
        log_warning "No PID file found for frontend"
    fi
    
    if lsof -ti:3000 >/dev/null 2>&1; then
        log_info "Killing process on port 3000..."
        lsof -ti:3000 | xargs kill -9 2>/dev/null || log_warning "Failed to kill process on port 3000"
    fi
    
    log_success "Frontend stopped"
}

stop_infrastructure() {
    log_info "Stopping infrastructure services..."
    
    cd "$PROJECT_ROOT"
    
    if docker ps | grep -q etcd; then
        log_info "Stopping Etcd..."
        docker-compose -f deploy/docker-compose.infra.yml stop etcd 2>/dev/null || log_warning "Failed to stop Etcd"
    fi
    
    if docker ps | grep -q jaeger; then
        log_info "Stopping Jaeger..."
        docker-compose -f deploy/docker-compose.infra.yml stop jaeger 2>/dev/null || log_warning "Failed to stop Jaeger"
    fi
    
    log_success "Infrastructure services stopped"
}

show_status() {
    echo ""
    log_info "Checking services status..."
    echo ""
    
    log_info "Microservices:"
    local services=("gateway:8000" "vehicle:8001" "billing:8002" "payment:8003" "admin:8004")
    for svc in "${services[@]}"; do
        IFS=':' read -r name port <<< "$svc"
        if lsof -ti:$port >/dev/null 2>&1; then
            log_warning "$name service still running on port $port"
        else
            log_success "$name service stopped"
        fi
    done
    
    echo ""
    log_info "Frontend:"
    if lsof -ti:3000 >/dev/null 2>&1; then
        log_warning "Frontend still running on port 3000"
    else
        log_success "Frontend stopped"
    fi
    
    echo ""
    log_success "All services stopped"
}

main() {
    log_info "Stopping development environment..."
    
    stop_microservices
    stop_frontend
    stop_infrastructure
    
    show_status
}

main "$@"
