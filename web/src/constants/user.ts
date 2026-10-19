// 用户角色
export const USER_ROLES = {
    ADMIN: "admin",
    MANAGER: "manager",
    OPERATOR: "operator",
} as const;

export type UserRole = typeof USER_ROLES[keyof typeof USER_ROLES];

// 用户角色标签
export const USER_ROLE_LABELS: Record<UserRole | "all", string> = {
    all: "全部",
    [USER_ROLES.ADMIN]: "管理员",
    [USER_ROLES.MANAGER]: "经理",
    [USER_ROLES.OPERATOR]: "操作员",
};

// 用户角色选项
export const USER_ROLE_OPTIONS = [
    { value: USER_ROLES.ADMIN, label: USER_ROLE_LABELS[USER_ROLES.ADMIN] },
    { value: USER_ROLES.MANAGER, label: USER_ROLE_LABELS[USER_ROLES.MANAGER] },
    { value: USER_ROLES.OPERATOR, label: USER_ROLE_LABELS[USER_ROLES.OPERATOR] },
];

// 用户角色选项（包含"全部"，用于筛选）
export const USER_ROLE_OPTIONS_WITH_ALL = [
    { value: "all", label: "全部" },
    ...USER_ROLE_OPTIONS,
];
