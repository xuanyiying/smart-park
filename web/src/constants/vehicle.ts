// 车辆类型
export const VEHICLE_TYPES = {
  TEMPORARY: "temporary",
  MONTHLY: "monthly",
  VIP: "vip",
} as const;

// 车辆类型标签
export const VEHICLE_TYPE_LABELS: Record<string, string> = {
  [VEHICLE_TYPES.TEMPORARY]: "临时车",
  [VEHICLE_TYPES.MONTHLY]: "月租车",
  [VEHICLE_TYPES.VIP]: "VIP",
};

// 车辆类型选项（用于下拉选择）
export const VEHICLE_TYPE_OPTIONS = [
  { value: VEHICLE_TYPES.TEMPORARY, label: VEHICLE_TYPE_LABELS[VEHICLE_TYPES.TEMPORARY] },
  { value: VEHICLE_TYPES.MONTHLY, label: VEHICLE_TYPE_LABELS[VEHICLE_TYPES.MONTHLY] },
  { value: VEHICLE_TYPES.VIP, label: VEHICLE_TYPE_LABELS[VEHICLE_TYPES.VIP] },
];

// 车辆类型选项（包含"全部"，用于筛选）
export const VEHICLE_TYPE_OPTIONS_WITH_ALL = [
  { value: "all", label: "全部" },
  ...VEHICLE_TYPE_OPTIONS,
];

// 计费规则中的车辆类型（与车辆管理中的类型不同）
export const BILLING_VEHICLE_TYPES = {
  SMALL: "small",
  LARGE: "large",
  ELECTRIC: "electric",
  MOTORCYCLE: "motorcycle",
} as const;

// 计费规则车辆类型标签
export const BILLING_VEHICLE_TYPE_LABELS: Record<string, string> = {
  [BILLING_VEHICLE_TYPES.SMALL]: "小型车",
  [BILLING_VEHICLE_TYPES.LARGE]: "大型车",
  [BILLING_VEHICLE_TYPES.ELECTRIC]: "电动车",
  [BILLING_VEHICLE_TYPES.MOTORCYCLE]: "摩托车",
};

// 计费规则车辆类型选项
export const BILLING_VEHICLE_TYPE_OPTIONS = [
  { value: BILLING_VEHICLE_TYPES.SMALL, label: BILLING_VEHICLE_TYPE_LABELS[BILLING_VEHICLE_TYPES.SMALL] },
  { value: BILLING_VEHICLE_TYPES.LARGE, label: BILLING_VEHICLE_TYPE_LABELS[BILLING_VEHICLE_TYPES.LARGE] },
  { value: BILLING_VEHICLE_TYPES.ELECTRIC, label: BILLING_VEHICLE_TYPE_LABELS[BILLING_VEHICLE_TYPES.ELECTRIC] },
  { value: BILLING_VEHICLE_TYPES.MOTORCYCLE, label: BILLING_VEHICLE_TYPE_LABELS[BILLING_VEHICLE_TYPES.MOTORCYCLE] },
];
