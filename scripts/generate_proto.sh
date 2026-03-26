#!/bin/bash

set -e

echo "Generating proto files with kratos..."

cd "$(dirname "$0")/.."

generate_service() {
    local proto_file=$1
    echo "Generating $proto_file..."
    kratos proto client -p ./third_party "$proto_file" 2>&1 || echo "Warning: $proto_file may have issues"
}

generate_service api/admin/v1/admin.proto
generate_service api/billing/v1/billing.proto
generate_service api/payment/v1/payment.proto
generate_service api/vehicle/v1/vehicle.proto

echo "Proto generation complete!"
echo ""
echo "Generated files:"
ls -la api/*/v1/*.pb.go 2>/dev/null || echo "No .pb.go files found"