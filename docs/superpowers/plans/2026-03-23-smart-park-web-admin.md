# Smart Park Web管理端实施计划

> **For agentic workers:** 使用 superpowers:subagent-driven-development 或 superpowers:executing-plans 逐任务执行。

**目标:** 为智慧停车场系统构建Web管理后台,包含数据概览、停车场管理、车辆管理、订单管理、计费规则、设备控制、用户权限、报表统计模块。

**架构:** Next.js 16 + TypeScript + Tailwind CSS v4 + shadcn/ui,响应式布局,对接后端Kratos微服务API。

**技术栈:**
- Next.js 16 (App Router)
- TypeScript
- Tailwind CSS v4
- shadcn/ui
- TanStack Query
- Zustand

---

## 文件结构

```
web/
├── app/
│   ├── (auth)/           # 登录页
│   │   └── login/
│   │       └── page.tsx
│   ├── (dashboard)/      # 管理后台
│   │   ├── layout.tsx    # 侧边栏+顶部
│   │   ├── page.tsx      # 首页/数据概览
│   │   ├── parking-lots/ # 停车场
│   │   │   ├── page.tsx  # 列表
│   │   │   └── [id]/     # 详情/编辑
│   │   ├── vehicles/     # 车辆
│   │   ├── orders/       # 订单
│   │   ├── billing/      # 计费规则
│   │   ├── devices/      # 设备控制
│   │   ├── users/        # 用户权限
│   │   └── reports/      # 报表统计
│   └── layout.tsx        # Root layout
├── components/
│   ├── ui/               # shadcn/ui组件
│   ├── layout/           # 布局组件(Sidebar, Header)
│   └── modules/          # 业务组件
├── lib/                  # 工具函数
│   ├── utils.ts
│   └── api.ts           # 统一API封装
├── services/             # API调用
│   ├── admin.ts
│   ├── billing.ts
│   └── vehicle.ts
└── types/                # TypeScript类型
    └── api.ts
```

---

## 实施任务

### 任务1: 项目初始化

**Files:**
- Create: `web/package.json`
- Create: `web/tsconfig.json`
- Create: `web/next.config.ts`
- Create: `web/tailwind.config.ts`
- Create: `web/postcss.config.mjs`
- Create: `web/app/globals.css`
- Create: `web/app/layout.tsx`

- [ ] **Step 1: 创建Next.js项目配置文件**

```bash
cd web
npx create-next-app@latest . --typescript --tailwind --eslint --app --src-dir --no-import-alias --turbopack
```

- [ ] **Step 2: 安装shadcn/ui依赖**

```bash
npx shadcn@latest init -d
```

- [ ] **Step 3: 添加基础组件**

```bash
npx shadcn@latest add button input card table form dialog select dropdown-menu badge avatar sheet tooltip -y
```

- [ ] **Step 4: 安装额外依赖**

```bash
npm install @tanstack/react-query zustand lucide-react clsx tailwind-merge recharts date-fns
```

- [ ] **Step 5: 验证项目运行**

```bash
npm run dev
# Expected: http://localhost:3000 可访问
```

---

### 任务2: 布局组件

**Files:**
- Create: `web/components/layout/sidebar.tsx`
- Create: `web/components/layout/header.tsx`
- Create: `web/app/(dashboard)/layout.tsx`
- Modify: `web/app/layout.tsx`

- [ ] **Step 1: 创建Sidebar组件**

```tsx
// components/layout/sidebar.tsx
import Link from "next/link";
import { usePathname } from "next/navigation";
import { cn } from "@/lib/utils";
import {
  LayoutDashboard,
  Car,
  FileText,
  CreditCard,
  Settings,
  Users,
  BarChart3,
  Warehouse,
} from "lucide-react";

const menuItems = [
  { href: "/", label: "数据概览", icon: LayoutDashboard },
  { href: "/parking-lots", label: "停车场", icon: Warehouse },
  { href: "/vehicles", label: "车辆管理", icon: Car },
  { href: "/orders", label: "订单管理", icon: FileText },
  { href: "/billing", label: "计费规则", icon: CreditCard },
  { href: "/devices", label: "设备控制", icon: Settings },
  { href: "/users", label: "用户权限", icon: Users },
  { href: "/reports", label: "报表统计", icon: BarChart3 },
];

export function Sidebar() {
  const pathname = usePathname();
  return (
    <aside className="w-64 bg-white border-r border-gray-200 min-h-screen">
      <div className="p-4 border-b border-gray-200">
        <h1 className="text-xl font-bold text-blue-600">Smart Park</h1>
      </div>
      <nav className="p-2">
        {menuItems.map((item) => {
          const Icon = item.icon;
          const isActive = pathname === item.href || (item.href !== "/" && pathname.startsWith(item.href));
          return (
            <Link
              key={item.href}
              href={item.href}
              className={cn(
                "flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-colors",
                isActive
                  ? "bg-blue-50 text-blue-600"
                  : "text-gray-600 hover:bg-gray-50"
              )}
            >
              <Icon className="w-5 h-5" />
              {item.label}
            </Link>
          );
        })}
      </nav>
    </aside>
  );
}
```

- [ ] **Step 2: 创建Header组件**

```tsx
// components/layout/header.tsx
"use client";
import { Bell, User, LogOut } from "lucide-react";
import { useState } from "react";

export function Header() {
  return (
    <header className="h-16 bg-white border-b border-gray-200 flex items-center justify-between px-6">
      <div className="text-sm text-gray-500">管理后台</div>
      <div className="flex items-center gap-4">
        <button className="p-2 hover:bg-gray-100 rounded-lg relative">
          <Bell className="w-5 h-5 text-gray-600" />
          <span className="absolute top-1 right-1 w-2 h-2 bg-red-500 rounded-full" />
        </button>
        <div className="flex items-center gap-2">
          <div className="w-8 h-8 bg-blue-100 rounded-full flex items-center justify-center">
            <User className="w-4 h-4 text-blue-600" />
          </div>
          <span className="text-sm font-medium">管理员</span>
        </div>
      </div>
    </header>
  );
}
```

- [ ] **Step 3: 创建Dashboard Layout**

```tsx
// app/(dashboard)/layout.tsx
import { Sidebar } from "@/components/layout/sidebar";
import { Header } from "@/components/layout/header";

export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="flex min-h-screen bg-gray-50">
      <Sidebar />
      <div className="flex-1 flex flex-col">
        <Header />
        <main className="flex-1 p-6">{children}</main>
      </div>
    </div>
  );
}
```

---

### 任务3: API服务层

**Files:**
- Create: `web/lib/api.ts`
- Create: `web/services/admin.ts`
- Create: `web/services/vehicle.ts`
- Create: `web/services/billing.ts`

- [ ] **Step 1: 创建API基础封装**

```ts
// lib/api.ts
const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8000";

class ApiClient {
  private baseUrl: string;

  constructor(baseUrl: string) {
    this.baseUrl = baseUrl;
  }

  private async request<T>(endpoint: string, options?: RequestInit): Promise<T> {
    const response = await fetch(`${this.baseUrl}${endpoint}`, {
      ...options,
      headers: {
        "Content-Type": "application/json",
        ...options?.headers,
      },
    });

    if (!response.ok) {
      const error = await response.json().catch(() => ({ message: "Request failed" }));
      throw new Error(error.message || `HTTP ${response.status}`);
    }

    return response.json();
  }

  get<T>(endpoint: string): Promise<T> {
    return this.request<T>(endpoint, { method: "GET" });
  }

  post<T>(endpoint: string, data?: unknown): Promise<T> {
    return this.request<T>(endpoint, {
      method: "POST",
      body: data ? JSON.stringify(data) : undefined,
    });
  }

  put<T>(endpoint: string, data?: unknown): Promise<T> {
    return this.request<T>(endpoint, {
      method: "PUT",
      body: data ? JSON.stringify(data) : undefined,
    });
  }

  delete<T>(endpoint: string): Promise<T> {
    return this.request<T>(endpoint, { method: "DELETE" });
  }
}

export const api = new ApiClient(API_BASE);
```

- [ ] **Step 2: 创建Admin服务**

```ts
// services/admin.ts
import { api } from "@/lib/api";

export interface ParkingLot {
  id: string;
  name: string;
  address: string;
  lanes: number;
  status: string;
  created_at: string;
  updated_at: string;
}

export interface Vehicle {
  id: string;
  plate_number: string;
  vehicle_type: string;
  owner_name: string;
  owner_phone: string;
  monthly_valid_until: string;
  created_at: string;
}

export interface Order {
  id: string;
  record_id: string;
  lot_id: string;
  plate_number: string;
  amount: number;
  discount_amount: number;
  final_amount: number;
  status: string;
  pay_time: string;
  pay_method: string;
}

export interface DailyReport {
  lot_id: string;
  date: string;
  total_entries: number;
  total_exits: number;
  total_vehicles: number;
  total_amount: number;
  total_discount: number;
  net_amount: number;
}

export const adminService = {
  listParkingLots: (page = 1, pageSize = 10) =>
    api.get(`/api/v1/admin/lots?page=${page}&page_size=${pageSize}`),

  getParkingLot: (id: string) => api.get(`/api/v1/admin/lots/${id}`),

  createParkingLot: (data: { name: string; address: string; lanes: number }) =>
    api.post("/api/v1/admin/lots", data),

  updateParkingLot: (data: { id: string; name: string; address: string; lanes: number; status: string }) =>
    api.put(`/api/v1/admin/lots/${data.id}`, data),

  listVehicles: (page = 1, pageSize = 10, vehicleType?: string) =>
    api.get(`/api/v1/admin/vehicles?page=${page}&page_size=${pageSize}&vehicle_type=${vehicleType || ""}`),

  createVehicle: (data: { plate_number: string; vehicle_type: string; owner_name: string; owner_phone: string; monthly_valid_until: string }) =>
    api.post("/api/v1/admin/vehicles", data),

  listOrders: (page = 1, pageSize = 10, status?: string, lotId?: string) =>
    api.get(`/api/v1/admin/orders?page=${page}&page_size=${page_size}&status=${status || ""}&lot_id=${lotId || ""}`),

  getOrder: (id: string) => api.get(`/api/v1/admin/orders/${id}`),

  getDailyReport: (lotId: string, date: string) =>
    api.get(`/api/v1/admin/reports/daily?lot_id=${lotId}&date=${date}`),

  getMonthlyReport: (lotId: string, year: number, month: number) =>
    api.get(`/api/v1/admin/reports/monthly?lot_id=${lotId}&year=${year}&month=${month}`),
};
```

---

### 任务4: 数据概览页面

**Files:**
- Create: `web/app/(dashboard)/page.tsx`
- Modify: `web/app/globals.css`

- [ ] **Step 1: 创建统计卡片组件**

```tsx
// components/modules/stats-card.tsx
"use client";
import { cn } from "@/lib/utils";

interface StatsCardProps {
  title: string;
  value: string | number;
  change?: number;
  icon: React.ReactNode;
  trend?: "up" | "down" | "neutral";
}

export function StatsCard({ title, value, change, icon, trend }: StatsCardProps) {
  return (
    <div className="bg-white rounded-xl p-6 border border-gray-100 shadow-sm">
      <div className="flex items-center justify-between">
        <div>
          <p className="text-sm text-gray-500 mb-1">{title}</p>
          <p className="text-2xl font-semibold">{value}</p>
          {change !== undefined && (
            <p className={cn(
              "text-sm mt-1",
              trend === "up" ? "text-green-600" : trend === "down" ? "text-red-600" : "text-gray-500"
            )}>
              {trend === "up" ? "↑" : trend === "down" ? "↓" : ""} {Math.abs(change)}%
            </p>
          )}
        </div>
        <div className="w-12 h-12 bg-blue-50 rounded-lg flex items-center justify-center text-blue-600">
          {icon}
        </div>
      </div>
    </div>
  );
}
```

- [ ] **Step 2: 创建Dashboard页面**

```tsx
// app/(dashboard)/page.tsx
"use client";
import { useEffect, useState } from "react";
import { StatsCard } from "@/components/modules/stats-card";
import { DollarSign, Car, Activity, Server } from "lucide-react";

export default function DashboardPage() {
  const [stats, setStats] = useState({
    todayIncome: 0,
    todayEntries: 0,
    todayExits: 0,
    onlineDevices: 0,
  });

  useEffect(() => {
    // TODO: 替换为实际API调用
    setStats({
      todayIncome: 12850,
      todayEntries: 156,
      todayExits: 142,
      onlineDevices: 8,
    });
  }, []);

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold text-gray-900">数据概览</h1>
        <p className="text-gray-500 mt-1">实时掌握停车场运营情况</p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <StatsCard
          title="今日收入"
          value={`¥${stats.todayIncome.toLocaleString()}`}
          change={12}
          trend="up"
          icon={<DollarSign className="w-6 h-6" />}
        />
        <StatsCard
          title="今日入场"
          value={stats.todayEntries}
          change={5}
          trend="up"
          icon={<Car className="w-6 h-6" />}
        />
        <StatsCard
          title="今日出场"
          value={stats.todayExits}
          change={-2}
          trend="down"
          icon={<Activity className="w-6 h-6" />}
        />
        <StatsCard
          title="在线设备"
          value={stats.onlineDevices}
          icon={<Server className="w-6 h-6" />}
        />
      </div>

      {/* 7天趋势图表 - TODO */}
      <div className="bg-white rounded-xl p-6 border border-gray-100 shadow-sm">
        <h2 className="text-lg font-medium mb-4">7天趋势</h2>
        <div className="h-64 flex items-center justify-center text-gray-400">
          图表组件占位 - 将使用recharts实现
        </div>
      </div>
    </div>
  );
}
```

---

### 任务5: 停车场管理模块

**Files:**
- Create: `web/app/(dashboard)/parking-lots/page.tsx`
- Create: `web/app/(dashboard)/parking-lots/[id]/page.tsx`
- Create: `web/components/modules/parking-lots-table.tsx`

- [ ] **Step 1: 创建停车场列表页面**

```tsx
// app/(dashboard)/parking-lots/page.tsx
"use client";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Plus, Search, MoreVertical } from "lucide-react";
import Link from "next/link";

const mockParkingLots = [
  { id: "1", name: "智慧中心停车场", address: "北京市朝阳区", lanes: 4, status: "active" },
  { id: "2", name: "商业广场停车场", address: "北京市海淀区", lanes: 6, status: "active" },
  { id: "3", name: "小区地下停车场", address: "北京市东城区", lanes: 2, status: "inactive" },
];

export default function ParkingLotsPage() {
  const [search, setSearch] = useState("");

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold text-gray-900">停车场管理</h1>
          <p className="text-gray-500 mt-1">管理停车场信息</p>
        </div>
        <Button>
          <Plus className="w-4 h-4 mr-2" />
          新增停车场
        </Button>
      </div>

      <Card>
        <CardHeader className="pb-3">
          <div className="flex items-center gap-4">
            <div className="relative flex-1 max-w-sm">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
              <Input
                placeholder="搜索停车场..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="pl-10"
              />
            </div>
          </div>
        </CardHeader>
        <CardContent>
          <div className="border rounded-lg">
            <table className="w-full">
              <thead className="bg-gray-50 border-b">
                <tr>
                  <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">名称</th>
                  <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">地址</th>
                  <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">车道数</th>
                  <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">状态</th>
                  <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">操作</th>
                </tr>
              </thead>
              <tbody>
                {mockParkingLots.map((lot) => (
                  <tr key={lot.id} className="border-b last:border-0 hover:bg-gray-50">
                    <td className="px-4 py-3 font-medium">{lot.name}</td>
                    <td className="px-4 py-3 text-gray-600">{lot.address}</td>
                    <td className="px-4 py-3">{lot.lanes}</td>
                    <td className="px-4 py-3">
                      <Badge variant={lot.status === "active" ? "default" : "secondary"}>
                        {lot.status === "active" ? "运营中" : "已停用"}
                      </Badge>
                    </td>
                    <td className="px-4 py-3">
                      <Link href={`/parking-lots/${lot.id}`}>
                        <Button variant="ghost" size="sm">查看</Button>
                      </Link>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
```

---

### 任务6: 车辆管理模块

**Files:**
- Create: `web/app/(dashboard)/vehicles/page.tsx`

---

### 任务7: 订单管理模块

**Files:**
- Create: `web/app/(dashboard)/orders/page.tsx`

---

### 任务8: 计费规则模块

**Files:**
- Create: `web/app/(dashboard)/billing/page.tsx`

---

### 任务9: 设备控制模块

**Files:**
- Create: `web/app/(dashboard)/devices/page.tsx`

---

### 任务10: 用户权限模块

**Files:**
- Create: `web/app/(dashboard)/users/page.tsx`

---

### 任务11: 报表统计模块

**Files:**
- Create: `web/app/(dashboard)/reports/page.tsx`

---

## 开发命令

```bash
# 启动开发服务器
cd web && npm run dev

# 构建生产版本
cd web && npm run build

# 代码检查
cd web && npm run lint
```

## 状态

- [ ] 任务1: 项目初始化
- [ ] 任务2: 布局组件
- [ ] 任务3: API服务层
- [ ] 任务4: 数据概览页面
- [ ] 任务5: 停车场管理模块
- [ ] 任务6: 车辆管理模块
- [ ] 任务7: 订单管理模块
- [ ] 任务8: 计费规则模块
- [ ] 任务9: 设备控制模块
- [ ] 任务10: 用户权限模块
- [ ] 任务11: 报表统计模块
