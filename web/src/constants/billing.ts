// 计费规则类型
export const RULE_TYPES = {
    BASE: "base",
    DISCOUNT: "discount",
    EXEMPTION: "exemption",
    TIME_BASED: "time_based",
    MONTHLY: "monthly",
} as const;

// 计费规则类型标签
export const RULE_TYPE_LABELS: Record<string, string> = {
    [RULE_TYPES.BASE]: "基础计费",
    [RULE_TYPES.DISCOUNT]: "折扣优惠",
    [RULE_TYPES.EXEMPTION]: "费用减免",
    [RULE_TYPES.TIME_BASED]: "时段收费",
    [RULE_TYPES.MONTHLY]: "月卡收费",
};

// 计费规则类型配置
export const RULE_TYPE_CONFIG: Record<
    string,
    {
        label: string;
        description: string;
        conditionFields: string[];
        actionFields: string[];
    }
> = {
    [RULE_TYPES.BASE]: {
        label: "基础计费",
        description: "按小时或按次收费",
        conditionFields: ["vehicle_type"],
        actionFields: ["base_fee", "hourly_rate", "max_daily"],
    },
    [RULE_TYPES.DISCOUNT]: {
        label: "折扣优惠",
        description: "满足条件时给予折扣",
        conditionFields: ["vehicle_type", "duration_min"],
        actionFields: ["discount_percent"],
    },
    [RULE_TYPES.EXEMPTION]: {
        label: "费用减免",
        description: "特定车辆减免费用",
        conditionFields: ["vehicle_type"],
        actionFields: ["exemption_amount"],
    },
    [RULE_TYPES.TIME_BASED]: {
        label: "时段收费",
        description: "不同时段不同费率",
        conditionFields: ["start_hour", "end_hour"],
        actionFields: ["hourly_rate", "multiplier"],
    },
    [RULE_TYPES.MONTHLY]: {
        label: "月卡收费",
        description: "包月车辆收费",
        conditionFields: ["plate_prefix"],
        actionFields: ["monthly_fee", "valid_days"],
    },
};

// 计费规则类型选项
export const RULE_TYPE_OPTIONS = Object.values(RULE_TYPE_CONFIG).map((config) => ({
    value: Object.keys(RULE_TYPE_CONFIG).find((key) => RULE_TYPE_CONFIG[key] === config) || "",
    label: config.label,
}));

// 优先级标签
export const RULE_PRIORITY_LABELS: Record<number, string> = {
    0: "最低",
    1: "低",
    2: "中",
    3: "高",
    4: "最高",
};

// 优先级选项
export const RULE_PRIORITY_OPTIONS = [
    { value: 0, label: "最低" },
    { value: 1, label: "低" },
    { value: 2, label: "中" },
    { value: 3, label: "高" },
    { value: 4, label: "最高" },
];

// 条件字段配置
export const CONDITION_FIELD_CONFIG: Record<
    string,
    { label: string; type: string; placeholder?: string; min?: number; max?: number }
> = {
    vehicle_type: { label: "车辆类型", type: "select", placeholder: "选择车辆类型" },
    duration_min: { label: "停车时长(分钟)", type: "number", placeholder: "例如: 120", min: 0 },
    start_hour: { label: "开始时段", type: "number", placeholder: "0-23", min: 0, max: 23 },
    end_hour: { label: "结束时段", type: "number", placeholder: "0-23", min: 0, max: 23 },
    plate_prefix: { label: "车牌前缀", type: "text", placeholder: "例如: 京A" },
};

// 动作字段配置
export const ACTION_FIELD_CONFIG: Record<
    string,
    { label: string; type: string; placeholder?: string; min?: number; max?: number; suffix?: string; step?: number }
> = {
    base_fee: { label: "起步价", type: "number", placeholder: "例如: 5", min: 0, suffix: "元" },
    hourly_rate: { label: "每小时费率", type: "number", placeholder: "例如: 2", min: 0, suffix: "元/小时" },
    max_daily: { label: "每日封顶", type: "number", placeholder: "例如: 50", min: 0, suffix: "元" },
    discount_percent: { label: "折扣比例", type: "number", placeholder: "例如: 20", min: 0, max: 100, suffix: "%" },
    exemption_amount: { label: "减免金额", type: "number", placeholder: "例如: 100", min: 0, suffix: "元" },
    multiplier: { label: "费率倍数", type: "number", placeholder: "例如: 1.5", min: 0, step: 0.1 },
    monthly_fee: { label: "月卡费用", type: "number", placeholder: "例如: 300", min: 0, suffix: "元" },
    valid_days: { label: "有效天数", type: "number", placeholder: "例如: 30", min: 1, suffix: "天" },
};
