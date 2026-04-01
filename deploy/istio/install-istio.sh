#!/bin/bash

# Istio 安装脚本

set -e

echo "=== 开始安装 Istio ==="

# 下载 Istio
ISTIO_VERSION="1.20.0"
echo "下载 Istio $ISTIO_VERSION..."
curl -L https://istio.io/downloadIstio | ISTIO_VERSION=$ISTIO_VERSION sh -

# 添加 istioctl 到 PATH
export PATH="$(pwd)/istio-$ISTIO_VERSION/bin:$PATH"

# 安装 Istio 到 Kubernetes 集群
echo "安装 Istio 到 Kubernetes 集群..."
istioctl install --set profile=default -y

# 启用自动注入
echo "启用命名空间的自动注入..."
kubectl label namespace smart-park istio-injection=enabled

echo "=== Istio 安装完成 ==="
