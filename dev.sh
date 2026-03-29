#!/bin/bash

cat << 'EOF'
╔═══════════════════════════════════════════════════════════╗
║                                                           ║
║   ███████╗██████╗  █████╗  ██████╗███████╗                ║
║   ██╔════╝██╔══██╗██╔══██╗██╔════╝██╔════╝                ║
║   ███████╗██████╔╝███████║██║     █████╗                  ║
║   ╚════██║██╔══██╗██╔══██║██║     ██╔══╝                  ║
║   ███████║██████╔╝██║  ██║╚██████╗███████╗                ║
║   ╚══════╝╚═════╝ ╚═╝  ╚═╝ ╚═════╝╚══════╝                ║
║                                                           ║
║   Smart Park - 停车场管理系统                              ║
║                                                           ║
╚═══════════════════════════════════════════════════════════╝
EOF

echo ""
echo "请选择操作："
echo ""
echo "  1) 启动所有服务"
echo "  2) 停止所有服务"
echo "  3) 查看服务状态"
echo "  4) 重启所有服务"
echo "  5) 查看日志"
echo "  6) 退出"
echo ""

read -p "请输入选项 [1-6]: " choice

case $choice in
    1)
        echo ""
        ./scripts/start-all.sh
        ;;
    2)
        echo ""
        read -p "是否停止基础设施服务？(y/N): " stop_infra
        if [[ $stop_infra =~ ^[Yy]$ ]]; then
            ./scripts/stop-all.sh --all
        else
            ./scripts/stop-all.sh
        fi
        ;;
    3)
        echo ""
        ./scripts/status.sh
        ;;
    4)
        echo ""
        echo "正在重启所有服务..."
        ./scripts/stop-all.sh
        sleep 2
        ./scripts/start-all.sh --skip-build
        ;;
    5)
        echo ""
        echo "可用的服务："
        echo "  1) gateway"
        echo "  2) vehicle"
        echo "  3) billing"
        echo "  4) payment"
        echo "  5) admin"
        echo "  6) frontend"
        echo "  7) 所有服务"
        echo ""
        read -p "请选择服务 [1-7]: " log_choice
        
        case $log_choice in
            1) ./scripts/status.sh --logs gateway ;;
            2) ./scripts/status.sh --logs vehicle ;;
            3) ./scripts/status.sh --logs billing ;;
            4) ./scripts/status.sh --logs payment ;;
            5) ./scripts/status.sh --logs admin ;;
            6) ./scripts/status.sh --logs frontend ;;
            7) ./scripts/status.sh --all-logs ;;
            *) echo "无效选项" ;;
        esac
        ;;
    6)
        echo "再见！"
        exit 0
        ;;
    *)
        echo "无效选项"
        exit 1
        ;;
esac
