// 订单状态
export const ORDER_STATUS = {
    PAID: "paid",
    UNPAID: "unpaid",
    PENDING: "pending",
    REFUNDED: "refunded",
} as const;

// 订单状态配置（包含标签和样式变体）
export const ORDER_STATUS_CONFIG: Record<
    string,
    { label: string; variant: "default" | "destructive" | "secondary" }
> = {
    all: { label: "全部", variant: "default" },
    [ORDER_STATUS.PAID]: { label: "已支付", variant: "default" },
    [ORDER_STATUS.UNPAID]: { label: "未支付", variant: "destructive" },
    [ORDER_STATUS.PENDING]: { label: "处理中", variant: "secondary" },
    [ORDER_STATUS.REFUNDED]: { label: "已退款", variant: "secondary" },
};

// 订单状态标签
export const ORDER_STATUS_LABELS: Record<string, string> = {
    all: "全部",
    [ORDER_STATUS.PAID]: "已支付",
    [ORDER_STATUS.UNPAID]: "未支付",
    [ORDER_STATUS.PENDING]: "处理中",
    [ORDER_STATUS.REFUNDED]: "已退款",
};

// 订单状态选项
export const ORDER_STATUS_OPTIONS = [
    { value: ORDER_STATUS.PAID, label: ORDER_STATUS_LABELS[ORDER_STATUS.PAID] },
    { value: ORDER_STATUS.UNPAID, label: ORDER_STATUS_LABELS[ORDER_STATUS.UNPAID] },
    { value: ORDER_STATUS.PENDING, label: ORDER_STATUS_LABELS[ORDER_STATUS.PENDING] },
];

// 订单状态选项（包含"全部"，用于筛选）
export const ORDER_STATUS_OPTIONS_WITH_ALL = [
    { value: "all", label: "全部" },
    ...ORDER_STATUS_OPTIONS,
];

// 支付方式
export const PAY_METHODS = {
    WECHAT: "wechat",
    ALIPAY: "alipay",
} as const;

// 支付方式标签
export const PAY_METHOD_LABELS: Record<string, string> = {
    [PAY_METHODS.WECHAT]: "微信支付",
    [PAY_METHODS.ALIPAY]: "支付宝",
};

// 支付方式选项
export const PAY_METHOD_OPTIONS = [
    { value: PAY_METHODS.WECHAT, label: PAY_METHOD_LABELS[PAY_METHODS.WECHAT] },
    { value: PAY_METHODS.ALIPAY, label: PAY_METHOD_LABELS[PAY_METHODS.ALIPAY] },
];
