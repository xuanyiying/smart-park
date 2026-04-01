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

stop_backend_services() {
    log_info "Stopping backend services..."
    
    for service in vehicle billing payment admin gateway analytics frontend; do
        pid_file="/tmp/smart-park-${service}.pid"
        if [ -f "$pid_file" ]; then
            pid=$(cat "$pid_file")
            if kill -0 "$pid" 2>/dev/null; then
                log_info "Stopping ${service} service (PID: $pid)..."
                kill "$pid" 2>/dev/null || log_warning "Failed to stop ${service} service"
            else
                log_warning "${service} service is not running"
            fi
            rm -f "$pid_file"
        else
            log_warning "No PID file found for ${service} service"
        fi
    done
    
    for port in 8000 8001 8002 8003 8004 8006 3000; do
        if lsof -ti:$port >/dev/null 2>&1; then
            log_info "Killing process on port $port..."
            lsof -ti:$port | xargs kill -9 2>/dev/null || log_warning "Failed to kill process on port $port"
        fi
    done
    
    log_success "Backend services stopped"
}

stop_infrastructure() {
    log_info "Stopping infrastructure services..."
    
    cd "$PROJECT_ROOT"
    
    if docker ps | grep -q etcd; then
        log_info "Stopping Etcd..."
        docker-compose -f deploy/docker-compose.infra.yml stop etcd
    fi
    
    if docker ps | grep -q jaeger; then
        log_info "Stopping Jaeger..."
        docker-compose -f deploy/docker-compose.infra.yml stop jaeger
    fi
    
    log_success "Infrastructure services stopped"
}

show_status() {
    echo ""
    log_info "Checking services status..."
    echo ""
    
    log_info "Backend Services:"
    for port in 8000 8001 8002 8003 8004 8006 3000; do
        if lsof -ti:$port >/dev/null 2>&1; then
            log_error "Port $port is still active"
        else
            log_success "Port $port is stopped"
        fi
    done
    
    echo ""
    log_info "Infrastructure:"
    docker ps | grep -E "postgres|redis|etcd|jaeger" | awk '{print "  - " $1 ": " $NF}' || log_success "No infrastructure services running"
    
    echo ""
    log_success "========================================="
    log_success "  All Services Stopped Successfully!"
    log_success "========================================="
    echo ""
}

main() {
    log_info "Stopping all services..."
    
    stop_backend_services
    
    if [ "$1" = "--all" ]; then
        stop_infrastructure
    else
        log_info "Infrastructure services are still running (use --all to stop them)"
    fi
    
    show_status
}

main "$@"
