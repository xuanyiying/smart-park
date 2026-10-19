// 核对状态标签
export const RECONCILIATION_STATUS_LABELS: Record<string, string> = {
  matched: "匹配",
  mismatched: "不匹配",
  pending: "待核对",
};

// 核对状态 Badge 样式
export const RECONCILIATION_STATUS_VARIANT: Record<string, "default" | "secondary" | "destructive" | "outline"> = {
  matched: "default",
  mismatched: "destructive",
  pending: "outline",
};

// 支付方式标签
export const PAYMENT_METHOD_LABELS: Record<string, string> = {
  wechat: "微信支付",
  alipay: "支付宝",
  cash: "现金",
  card: "银行卡",
};

// 支付状态标签
export const PAYMENT_STATUS_LABELS: Record<string, string> = {
  pending: "待支付",
  paid: "已支付",
  refunded: "已退款",
  failed: "支付失败",
  cancelled: "已取消",
};

// 支付状态 Badge 样式
export const PAYMENT_STATUS_VARIANT: Record<string, "default" | "secondary" | "destructive" | "outline"> = {
  pending: "outline",
  paid: "default",
  refunded: "secondary",
  failed: "destructive",
  cancelled: "secondary",
};
