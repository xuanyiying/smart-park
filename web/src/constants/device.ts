// 设备类型常量
export const DEVICE_TYPES = {
  CAMERA: "camera",
  GATE: "gate",
  SENSOR: "sensor",
  DISPLAY: "display",
} as const;

// 设备类型标签映射
export const DEVICE_TYPE_LABELS: Record<string, string> = {
  [DEVICE_TYPES.CAMERA]: "摄像头",
  [DEVICE_TYPES.GATE]: "道闸",
  [DEVICE_TYPES.SENSOR]: "传感器",
  [DEVICE_TYPES.DISPLAY]: "显示屏",
};

// 设备类型选项（用于下拉选择）
export const DEVICE_TYPE_OPTIONS = [
  { value: DEVICE_TYPES.CAMERA, label: DEVICE_TYPE_LABELS[DEVICE_TYPES.CAMERA] },
  { value: DEVICE_TYPES.GATE, label: DEVICE_TYPE_LABELS[DEVICE_TYPES.GATE] },
  { value: DEVICE_TYPES.SENSOR, label: DEVICE_TYPE_LABELS[DEVICE_TYPES.SENSOR] },
  { value: DEVICE_TYPES.DISPLAY, label: DEVICE_TYPE_LABELS[DEVICE_TYPES.DISPLAY] },
];

// 设备状态常量
export const DEVICE_STATUS = {
  ACTIVE: "active",
  INACTIVE: "inactive",
  MAINTENANCE: "maintenance",
  ERROR: "error",
  NORMAL: "normal",
  OFFLINE: "offline",
  UPGRADING: "upgrading",
  FAULT: "fault",
} as const;

// 设备状态标签映射
export const DEVICE_STATUS_LABELS: Record<string, string> = {
  [DEVICE_STATUS.ACTIVE]: "正常",
  [DEVICE_STATUS.INACTIVE]: "停用",
  [DEVICE_STATUS.MAINTENANCE]: "维护中",
  [DEVICE_STATUS.ERROR]: "故障",
  [DEVICE_STATUS.NORMAL]: "正常",
  [DEVICE_STATUS.OFFLINE]: "离线",
  [DEVICE_STATUS.UPGRADING]: "升级中",
  [DEVICE_STATUS.FAULT]: "故障",
  all: "全部",
};

// 设备状态选项
export const DEVICE_STATUS_OPTIONS = [
  { value: DEVICE_STATUS.ACTIVE, label: DEVICE_STATUS_LABELS[DEVICE_STATUS.ACTIVE] },
  { value: DEVICE_STATUS.INACTIVE, label: DEVICE_STATUS_LABELS[DEVICE_STATUS.INACTIVE] },
  { value: DEVICE_STATUS.MAINTENANCE, label: DEVICE_STATUS_LABELS[DEVICE_STATUS.MAINTENANCE] },
  { value: DEVICE_STATUS.ERROR, label: DEVICE_STATUS_LABELS[DEVICE_STATUS.ERROR] },
  { value: DEVICE_STATUS.OFFLINE, label: DEVICE_STATUS_LABELS[DEVICE_STATUS.OFFLINE] },
  { value: DEVICE_STATUS.UPGRADING, label: DEVICE_STATUS_LABELS[DEVICE_STATUS.UPGRADING] },
];

// 设备状态选项（包含"全部"）
export const DEVICE_STATUS_OPTIONS_WITH_ALL = [
  { value: "all", label: "全部" },
  ...DEVICE_STATUS_OPTIONS,
];

// 故障严重级别标签
export const FAULT_SEVERITY_LABELS: Record<string, string> = {
  critical: "严重",
  major: "主要",
  minor: "次要",
  warning: "警告",
};

// 故障严重级别 Badge 样式
export const FAULT_SEVERITY_VARIANT: Record<string, "default" | "secondary" | "destructive" | "outline"> = {
  critical: "destructive",
  major: "default",
  minor: "secondary",
  warning: "outline",
};

// 故障状态标签
export const FAULT_STATUS_LABELS: Record<string, string> = {
  open: "未解决",
  detected: "已检测",
  resolved: "已解决",
  closed: "已关闭",
};

// 故障状态 Badge 样式
export const FAULT_STATUS_VARIANT: Record<string, "default" | "secondary" | "destructive" | "outline"> = {
  open: "destructive",
  detected: "outline",
  resolved: "default",
  closed: "secondary",
};

// 日志级别标签
export const LOG_LEVEL_LABELS: Record<string, string> = {
  info: "信息",
  warn: "警告",
  error: "错误",
  debug: "调试",
};

// 日志级别 Badge 样式
export const LOG_LEVEL_VARIANT: Record<string, "default" | "secondary" | "destructive" | "outline"> = {
  info: "outline",
  warn: "secondary",
  error: "destructive",
  debug: "secondary",
};

// 升级状态标签
export const UPGRADE_STATUS_LABELS: Record<string, string> = {
  pending: "等待中",
  downloading: "下载中",
  installing: "安装中",
  in_progress: "进行中",
  completed: "完成",
  failed: "失败",
  cancelled: "已取消",
};

// 升级状态 Badge 样式
export const UPGRADE_STATUS_VARIANT: Record<string, "default" | "secondary" | "destructive" | "outline"> = {
  pending: "outline",
  downloading: "outline",
  installing: "default",
  in_progress: "default",
  completed: "default",
  failed: "destructive",
  cancelled: "secondary",
};

// 固件状态标签
export const FIRMWARE_STATUS_LABELS: Record<string, string> = {
  draft: "草稿",
  published: "已发布",
  deprecated: "已废弃",
};
