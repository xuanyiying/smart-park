#!/bin/bash
# Smart Park 部署脚本
# 用法: ./deploy.sh [environment] [version]
# 示例: ./deploy.sh production v1.0.0

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 默认配置
ENV=${1:-development}
VERSION=${2:-latest}
COMPOSE_FILE="deploy/docker-compose.yml"

# 打印带颜色的信息
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查依赖
check_dependencies() {
    log_info "检查依赖..."
    
    if ! command -v docker &> /dev/null; then
        log_error "Docker 未安装"
        exit 1
    fi
    
    if ! command -v docker-compose &> /dev/null; then
        log_error "Docker Compose 未安装"
        exit 1
    fi
    
    log_info "依赖检查通过"
}

# 加载环境配置
load_env() {
    if [ -f ".env.${ENV}" ]; then
        log_info "加载环境配置: .env.${ENV}"
        export $(cat .env.${ENV} | grep -v '^#' | xargs)
    elif [ -f ".env" ]; then
        log_info "加载环境配置: .env"
        export $(cat .env | grep -v '^#' | xargs)
    else
        log_warn "未找到环境配置文件，使用默认配置"
    fi
}

# 构建镜像
build_images() {
    log_info "构建 Docker 镜像..."
    
    export VERSION=${VERSION}
    docker-compose -f ${COMPOSE_FILE} build --parallel
    
    log_info "镜像构建完成"
}

# 推送镜像到仓库
push_images() {
    log_info "推送镜像到仓库..."
    
    docker-compose -f ${COMPOSE_FILE} push
    
    log_info "镜像推送完成"
}

# 部署服务
deploy_services() {
    log_info "部署 Smart Park ${VERSION} 到 ${ENV} 环境..."
    
    # 创建网络（如果不存在）
    docker network create smart-park-network 2>/dev/null || true
    
    # 启动基础设施
    log_info "启动基础设施..."
    docker-compose -f ${COMPOSE_FILE} up -d postgres redis etcd jaeger
    
    # 等待数据库就绪
    log_info "等待数据库就绪..."
    sleep 10
    
    # 启动业务服务
    log_info "启动业务服务..."
    docker-compose -f ${COMPOSE_FILE} up -d gateway vehicle billing payment admin
    
    log_info "服务部署完成"
}

# 健康检查
health_check() {
    log_info "执行健康检查..."
    
    services=("gateway:8000" "vehicle:8001" "billing:8002" "payment:8003" "admin:8004")
    
    for service in "${services[@]}"; do
        IFS=':' read -r name port <<< "$service"
        
        max_attempts=30
        attempt=1
        
        while [ $attempt -le $max_attempts ]; do
            if curl -s "http://localhost:${port}/health" > /dev/null 2>&1; then
                log_info "${name} 服务健康"
                break
            fi
            
            if [ $attempt -eq $max_attempts ]; then
                log_error "${name} 服务健康检查失败"
                return 1
            fi
            
            log_warn "${name} 服务未就绪，等待中... (${attempt}/${max_attempts})"
            sleep 2
            ((attempt++))
        done
    done
    
    log_info "所有服务健康检查通过"
}

# 滚动更新
rolling_update() {
    log_info "执行滚动更新..."
    
    services=("gateway" "vehicle" "billing" "payment" "admin")
    
    for service in "${services[@]}"; do
        log_info "更新 ${service} 服务..."
        
        # 拉取最新镜像
        docker-compose -f ${COMPOSE_FILE} pull ${service}
        
        # 重启服务
        docker-compose -f ${COMPOSE_FILE} up -d --no-deps ${service}
        
        # 等待服务就绪
        sleep 5
        
        log_info "${service} 服务更新完成"
    done
    
    log_info "滚动更新完成"
}

# 回滚部署
rollback() {
    log_info "执行回滚..."
    
    # 使用上一个版本
    PREV_VERSION=$(docker images --format "{{.Tag}}" smart-park/gateway | grep -v "latest" | head -1)
    
    if [ -z "$PREV_VERSION" ]; then
        log_error "未找到上一个版本"
        exit 1
    fi
    
    log_info "回滚到版本: ${PREV_VERSION}"
    
    export VERSION=${PREV_VERSION}
    docker-compose -f ${COMPOSE_FILE} up -d
    
    log_info "回滚完成"
}

# 查看服务状态
status() {
    log_info "查看服务状态..."
    docker-compose -f ${COMPOSE_FILE} ps
}

# 查看日志
logs() {
    service=$1
    if [ -z "$service" ]; then
        docker-compose -f ${COMPOSE_FILE} logs -f --tail=100
    else
        docker-compose -f ${COMPOSE_FILE} logs -f --tail=100 ${service}
    fi
}

# 停止服务
stop() {
    log_info "停止服务..."
    docker-compose -f ${COMPOSE_FILE} down
    log_info "服务已停止"
}

# 清理资源
cleanup() {
    log_warn "清理未使用的 Docker 资源..."
    docker system prune -f
    docker volume prune -f
    log_info "清理完成"
}

# 数据库备份
backup() {
    BACKUP_DIR="./backups"
    DATE=$(date +%Y%m%d_%H%M%S)
    
    mkdir -p ${BACKUP_DIR}
    
    log_info "执行数据库备份..."
    
    docker exec smart-park-postgres pg_dump -U postgres parking > ${BACKUP_DIR}/parking_${DATE}.sql
    gzip ${BACKUP_DIR}/parking_${DATE}.sql
    
    # 保留最近 7 天备份
    find ${BACKUP_DIR} -name "parking_*.sql.gz" -mtime +7 -delete
    
    log_info "备份完成: ${BACKUP_DIR}/parking_${DATE}.sql.gz"
}

# 数据库恢复
restore() {
    BACKUP_FILE=$1
    
    if [ -z "$BACKUP_FILE" ]; then
        log_error "请指定备份文件路径"
        exit 1
    fi
    
    if [ ! -f "$BACKUP_FILE" ]; then
        log_error "备份文件不存在: ${BACKUP_FILE}"
        exit 1
    fi
    
    log_warn "恢复将覆盖现有数据，是否继续? (y/n)"
    read -r confirm
    if [ "$confirm" != "y" ]; then
        log_info "取消恢复"
        exit 0
    fi
    
    log_info "执行数据库恢复..."
    
    # 解压备份
    if [[ "$BACKUP_FILE" == *.gz ]]; then
        gunzip -c "$BACKUP_FILE" | docker exec -i smart-park-postgres psql -U postgres parking
    else
        docker exec -i smart-park-postgres psql -U postgres parking < "$BACKUP_FILE"
    fi
    
    log_info "恢复完成"
}

# 显示帮助信息
show_help() {
    echo "Smart Park 部署脚本"
    echo ""
    echo "用法: ./deploy.sh [命令] [选项]"
    echo ""
    echo "命令:"
    echo "  deploy [env] [version]    部署服务 (默认: development latest)"
    echo "  build                     构建 Docker 镜像"
    echo "  push                      推送镜像到仓库"
    echo "  update                    滚动更新服务"
    echo "  rollback                  回滚到上一个版本"
    echo "  status                    查看服务状态"
    echo "  logs [service]            查看日志 (可选指定服务)"
    echo "  stop                      停止所有服务"
    echo "  cleanup                   清理未使用的 Docker 资源"
    echo "  backup                    备份数据库"
    echo "  restore <file>            从备份文件恢复数据库"
    echo "  health                    执行健康检查"
    echo "  help                      显示帮助信息"
    echo ""
    echo "示例:"
    echo "  ./deploy.sh deploy production v1.0.0"
    echo "  ./deploy.sh logs gateway"
    echo "  ./deploy.sh backup"
}

# 主函数
main() {
    command=$1
    shift
    
    case $command in
        deploy)
            check_dependencies
            load_env
            build_images
            deploy_services
            health_check
            ;;
        build)
            check_dependencies
            build_images
            ;;
        push)
            check_dependencies
            push_images
            ;;
        update)
            check_dependencies
            load_env
            rolling_update
            health_check
            ;;
        rollback)
            check_dependencies
            rollback
            ;;
        status)
            status
            ;;
        logs)
            logs "$@"
            ;;
        stop)
            stop
            ;;
        cleanup)
            cleanup
            ;;
        backup)
            backup
            ;;
        restore)
            restore "$@"
            ;;
        health)
            health_check
            ;;
        help|*)
            show_help
            ;;
    esac
}

main "$@"
