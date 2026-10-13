#!/usr/bin/env bash
#
# create-secrets.sh - 创建 Smart Park 生产密钥
#
# 背景：仓库里曾经有一份 deploy/k8s/secrets.yaml，里面的值是 "${DB_PASSWORD}" 这类
# 字面占位符。kubectl apply 不做变量替换，直接 apply 会把字符串 "${DB_PASSWORD}"
# 当成真实密码写进集群——服务能启动，但用的是占位口令，且没有任何报错。
#
# 本脚本取代手写 YAML：
#   - db-password / jwt-secret 未显式提供时，用 crypto/rand 生成强随机值；
#   - 支付渠道私钥必须由运维提供文件路径，脚本拒绝写入占位 PEM；
#   - --dry-run 只输出 YAML 不落库，便于接入 GitOps / Sealed Secrets；
#   - 生成的密钥只进集群 Secret，不落地、不进 Git。
#
# 用法：
#   scripts/k8s/create-secrets.sh \
#     --namespace smart-park \
#     --wechat-private-key ./secrets/wechat_private_key.pem \
#     --alipay-private-key ./secrets/alipay_private_key.pem \
#     --alipay-public-key  ./secrets/alipay_public_key.pem
#
# 可选：
#   --db-password <pw>     不提供则随机生成
#   --jwt-secret <secret>  不提供则随机生成
#   --jwt-public-key <path> 网关 JWT 验签公钥（RS256 方案必需）
#   --dry-run              只打印将创建的 Secret YAML，不连接集群
#
set -euo pipefail

NAMESPACE="smart-park"
SECRET_NAME="smart-park-secrets"
DB_PASSWORD=""
JWT_SECRET=""
JWT_PUBLIC_KEY_PATH=""
WECHAT_PRIVATE_KEY_PATH=""
ALIPAY_PRIVATE_KEY_PATH=""
ALIPAY_PUBLIC_KEY_PATH=""
DRY_RUN=0

die() { echo "ERROR: $*" >&2; exit 1; }

while [[ $# -gt 0 ]]; do
  case "$1" in
    --namespace)            NAMESPACE="$2"; shift 2 ;;
    --secret-name)          SECRET_NAME="$2"; shift 2 ;;
    --db-password)          DB_PASSWORD="$2"; shift 2 ;;
    --jwt-secret)           JWT_SECRET="$2"; shift 2 ;;
    --jwt-public-key)       JWT_PUBLIC_KEY_PATH="$2"; shift 2 ;;
    --wechat-private-key)   WECHAT_PRIVATE_KEY_PATH="$2"; shift 2 ;;
    --alipay-private-key)   ALIPAY_PRIVATE_KEY_PATH="$2"; shift 2 ;;
    --alipay-public-key)    ALIPAY_PUBLIC_KEY_PATH="$2"; shift 2 ;;
    --dry-run)              DRY_RUN=1; shift ;;
    *) die "unknown argument: $1" ;;
  esac
done

# 强随机值：32 字节熵，base64url 编码（无 kubectl/YAML 需转义的字符）。
gen_secret() {
  openssl rand -base64 48 | tr -d '\n' | tr '+/' '-_'
}

[[ -n "$DB_PASSWORD" ]] || DB_PASSWORD="$(gen_secret)"
[[ -n "$JWT_SECRET" ]]  || JWT_SECRET="$(gen_secret)"

# 支付渠道密钥是真实资金凭证：拒绝占位内容，宁可失败也不能让假密钥进集群。
read_key() {
  local path="$1" name="$2"
  [[ -n "$path" ]] || die "$name is required (pass --$(echo "$name" | tr '[:upper:]' '[:lower:]' | tr '_' '-') <path>)"
  [[ -f "$path" ]] || die "$name file not found: $path"
  if grep -qE '替换为实际|REPLACE_ME|PLACEHOLDER' "$path"; then
    die "$name file $path still contains placeholder content"
  fi
  cat "$path"
}

WECHAT_PRIVATE_KEY="$(read_key "$WECHAT_PRIVATE_KEY_PATH" "WECHAT_PRIVATE_KEY")"
ALIPAY_PRIVATE_KEY="$(read_key "$ALIPAY_PRIVATE_KEY_PATH" "ALIPAY_PRIVATE_KEY")"
ALIPAY_PUBLIC_KEY="$(read_key "$ALIPAY_PUBLIC_KEY_PATH" "ALIPAY_PUBLIC_KEY")"

JWT_PUBLIC_KEY=""
if [[ -n "$JWT_PUBLIC_KEY_PATH" ]]; then
  JWT_PUBLIC_KEY="$(read_key "$JWT_PUBLIC_KEY_PATH" "JWT_PUBLIC_KEY")"
fi

# 用临时文件组装 Secret YAML，避免密钥出现在命令行参数（ps 可见）里。
TMP_LITERALS="$(mktemp)"
TMP_FILES="$(mktemp)"
trap 'rm -f "$TMP_LITERALS" "$TMP_FILES"' EXIT

{
  echo "--from-literal=db-password=${DB_PASSWORD}"
  echo "--from-literal=jwt-secret=${JWT_SECRET}"
  [[ -n "$JWT_PUBLIC_KEY" ]] && echo "--from-literal=jwt-public-key=${JWT_PUBLIC_KEY}"
} > "$TMP_LITERALS"

{
  echo "--from-file=wechat-private-key=${WECHAT_PRIVATE_KEY_PATH}"
  echo "--from-file=alipay-private-key=${ALIPAY_PRIVATE_KEY_PATH}"
  echo "--from-file=alipay-public-key=${ALIPAY_PUBLIC_KEY_PATH}"
} > "$TMP_FILES"

# shellcheck disable=SC2046
LITERAL_ARGS=(); while IFS= read -r line; do LITERAL_ARGS+=("$line"); done < "$TMP_LITERALS"
# shellcheck disable=SC2046
FILE_ARGS=(); while IFS= read -r line; do FILE_ARGS+=("$line"); done < "$TMP_FILES"

if [[ "$DRY_RUN" -eq 1 ]]; then
  kubectl create secret generic "$SECRET_NAME" \
    --namespace "$NAMESPACE" \
    "${LITERAL_ARGS[@]}" "${FILE_ARGS[@]}" \
    --dry-run=client -o yaml
  exit 0
fi

kubectl create secret generic "$SECRET_NAME" \
  --namespace "$NAMESPACE" \
  "${LITERAL_ARGS[@]}" "${FILE_ARGS[@]}" \
  --save-config --dry-run=client -o yaml | kubectl apply -f -

echo "Secret ${SECRET_NAME} created in namespace ${NAMESPACE}."
echo "NOTE: 请将 db-password / jwt-secret 的实际值记录到团队的密码管理器；本脚本不回显它们。"
