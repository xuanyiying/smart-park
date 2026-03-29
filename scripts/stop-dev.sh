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

stop_all_in_one() {
    log_info "Stopping all-in-one service..."
    
    pid_file="/tmp/smart-park-all-in-one.pid"
    if [ -f "$pid_file" ]; then
        pid=$(cat "$pid_file")
        if kill -0 "$pid" 2>/dev/null; then
            log_info "Stopping all-in-one service (PID: $pid)..."
            kill "$pid" 2>/dev/null || log_warning "Failed to stop all-in-one service"
        else
            log_warning "All-in-one service is not running"
        fi
        rm -f "$pid_file"
    else
        log_warning "No PID file found for all-in-one service"
    fi
    
    if lsof -ti:8000 >/dev/null 2>&1; then
        log_info "Killing process on port 8000..."
        lsof -ti:8000 | xargs kill -9 2>/dev/null || log_warning "Failed to kill process on port 8000"
    fi
    
    log_success "All-in-one service stopped"
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
    
    log_info "All-in-One Service:"
    if lsof -ti:8000 >/dev/null 2>&1; then
        log_error "Port 8000 is still active"
    else
        log_success "Port 8000 is stopped"
    fi
    
    echo ""
    log_info "Frontend:"
    if lsof -ti:3000 >/dev/null 2>&1; then
        log_error "Port 3000 is still active"
    else
        log_success "Port 3000 is stopped"
    fi
    
    echo ""
    log_info "Infrastructure:"
    docker ps | grep -E "postgres|redis|etcd|jaeger" | awk '{print "  - " $1 ": " $NF}' || log_success "No infrastructure services running"
    
    echo ""
    log_success "========================================="
    log_success "  Development Environment Stopped!"
    log_success "========================================="
    echo ""
}

main() {
    log_info "Stopping development environment..."
    
    stop_all_in_one
    stop_frontend
    
    if [ "$1" = "--all" ]; then
        stop_infrastructure
    else
        log_info "Infrastructure services are still running (use --all to stop them)"
    fi
    
    show_status
}

main "$@"
