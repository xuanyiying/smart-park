#!/bin/bash

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

check_service() {
    local service_name=$1
    local port=$2
    local pid_file="/tmp/smart-park-${service_name}.pid"
    
    if [ -f "$pid_file" ]; then
        pid=$(cat "$pid_file")
        if kill -0 "$pid" 2>/dev/null; then
            echo -e "  ${GREEN}●${NC} ${service_name} (PID: $pid, Port: $port)"
            return 0
        else
            echo -e "  ${RED}●${NC} ${service_name} (Not running, stale PID file)"
            return 1
        fi
    else
        if lsof -ti:$port >/dev/null 2>&1; then
            pid=$(lsof -ti:$port)
            echo -e "  ${YELLOW}●${NC} ${service_name} (Running on port $port, PID: $pid, no PID file)"
            return 0
        else
            echo -e "  ${RED}●${NC} ${service_name} (Not running)"
            return 1
        fi
    fi
}

check_infrastructure() {
    local service_name=$1
    local container_name=$2
    
    if docker ps --filter "name=$container_name" --filter "status=running" | grep -q "$container_name"; then
        echo -e "  ${GREEN}●${NC} ${service_name}"
        return 0
    else
        if docker ps -a --filter "name=$container_name" | grep -q "$container_name"; then
            echo -e "  ${YELLOW}●${NC} ${service_name} (Stopped)"
            return 1
        else
            echo -e "  ${RED}●${NC} ${service_name} (Not found)"
            return 1
        fi
    fi
}

show_logs() {
    local service_name=$1
    local log_file="$PROJECT_ROOT/logs/${service_name}.log"
    
    if [ -f "$log_file" ]; then
        echo ""
        log_info "Last 10 lines of ${service_name} log:"
        tail -n 10 "$log_file"
    else
        log_warning "No log file found for ${service_name}"
    fi
}

main() {
    echo ""
    log_info "========================================="
    log_info "  Smart Park Services Status"
    log_info "========================================="
    echo ""
    
    log_info "Infrastructure Services:"
    check_infrastructure "PostgreSQL" "postgres"
    check_infrastructure "Redis" "redis"
    check_infrastructure "Etcd" "etcd"
    check_infrastructure "Jaeger" "jaeger"
    
    echo ""
    log_info "Backend Services:"
    check_service "gateway" 8000
    check_service "vehicle" 8001
    check_service "billing" 8002
    check_service "payment" 8003
    check_service "admin" 8004
    
    echo ""
    log_info "Frontend:"
    check_service "frontend" 3000
    
    echo ""
    log_info "Service URLs:"
    echo "  - Frontend:    http://localhost:3000"
    echo "  - Gateway:     http://localhost:8000"
    echo "  - Jaeger UI:   http://localhost:16686"
    echo ""
    
    if [ "$1" = "--logs" ]; then
        service_name=$2
        if [ -n "$service_name" ]; then
            show_logs "$service_name"
        else
            log_error "Please specify a service name: gateway, vehicle, billing, payment, admin, or frontend"
        fi
    fi
    
    if [ "$1" = "--all-logs" ]; then
        for service in gateway vehicle billing payment admin frontend; do
            show_logs "$service"
        done
    fi
}

main "$@"
