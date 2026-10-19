"use client";

import { cn } from "@/lib/utils";
import {
  BarChart3,
  Car,
  ChevronDown,
  CreditCard,
  FileText,
  HardDrive,
  LayoutDashboard,
  ShieldBan,
  Building2,
  Users,
  Wallet,
  Warehouse,
  Zap
} from "lucide-react";
import Link from "next/link";
import { usePathname } from "next/navigation";
import { useState } from "react";

const menuItems = [
  { href: "/", label: "数据概览", icon: LayoutDashboard },
  { href: "/parking-lots", label: "停车场", icon: Warehouse },
  { href: "/vehicles", label: "车辆管理", icon: Car },
  { href: "/blacklist", label: "黑名单", icon: ShieldBan },
  { href: "/orders", label: "订单管理", icon: FileText },
  { href: "/billing", label: "计费规则", icon: CreditCard },
  {
    href: "/devices",
    label: "设备管理",
    icon: HardDrive,
    children: [
      { href: "/devices", label: "设备列表" },
      { href: "/devices/manufacturers", label: "厂商管理" },
      { href: "/devices/firmwares", label: "固件管理" },
    ],
  },
  {
    href: "/charging",
    label: "充电管理",
    icon: Zap,
    children: [
      { href: "/charging/stations", label: "充电桩" },
      { href: "/charging/monitor", label: "充电监控" },
      { href: "/charging/orders", label: "充电订单" },
      { href: "/charging/prices", label: "价格配置" },
    ],
  },
  {
    href: "/payments",
    label: "支付管理",
    icon: Wallet,
    children: [
      { href: "/payments", label: "支付对账" },
      { href: "/payments/orders", label: "支付订单" },
    ],
  },
  { href: "/users", label: "用户权限", icon: Users },
  { href: "/tenants", label: "租户管理", icon: Building2 },
  { href: "/reports", label: "报表统计", icon: BarChart3 },
];

export function Sidebar() {
  const pathname = usePathname();
  const [expandedMenus, setExpandedMenus] = useState<string[]>(() => {
    const expanded: string[] = [];
    menuItems.forEach((item) => {
      if (item.children) {
        const hasActiveChild = item.children.some(
          (child) => pathname === child.href || pathname.startsWith(child.href)
        );
        if (hasActiveChild) expanded.push(item.href);
      }
    });
    return expanded;
  });

  const toggleMenu = (href: string) => {
    setExpandedMenus((prev) =>
      prev.includes(href)
        ? prev.filter((h) => h !== href)
        : [...prev, href]
    );
  };

  return (
    <aside className="w-64 bg-white border-r border-gray-200 min-h-screen">
      <div className="p-4 border-b border-gray-200 flex items-center gap-2">
        <img src="/logo.svg" alt="Smart Park" className="w-8 h-8" />
        <h1 className="text-xl font-bold text-blue-600">Smart Park</h1>
      </div>
      <nav className="p-2">
        {menuItems.map((item) => {
          const Icon = item.icon;
          const hasChildren = !!item.children;
          const isExpanded = expandedMenus.includes(item.href);
          const isActive =
            !hasChildren &&
            (pathname === item.href ||
              (item.href !== "/" && pathname.startsWith(item.href)));
          const isGroupActive =
            hasChildren &&
            item.children!.some(
              (child) =>
                pathname === child.href ||
                pathname.startsWith(child.href)
            );

          if (hasChildren) {
            return (
              <div key={item.href}>
                <button
                  onClick={() => toggleMenu(item.href)}
                  className={cn(
                    "flex items-center gap-3 px-3 py-2 rounded-lg text-sm font-medium transition-colors w-full",
                    isGroupActive
                      ? "bg-blue-50 text-blue-600"
                      : "text-gray-600 hover:bg-gray-50"
                  )}
                >
                  <Icon className="w-5 h-5" />
                  <span className="flex-1 text-left">{item.label}</span>
                  <ChevronDown
                    className={cn(
                      "w-4 h-4 transition-transform",
                      isExpanded ? "rotate-180" : ""
                    )}
                  />
                </button>
                {isExpanded && (
                  <div className="ml-8 mt-1 space-y-1">
                    {item.children!.map((child) => {
                      const isChildActive =
                        pathname === child.href ||
                        pathname.startsWith(child.href);
                      return (
                        <Link
                          key={child.href}
                          href={child.href}
                          className={cn(
                            "flex items-center gap-2 px-3 py-1.5 rounded-lg text-sm transition-colors",
                            isChildActive
                              ? "bg-blue-50 text-blue-600 font-medium"
                              : "text-gray-500 hover:bg-gray-50 hover:text-gray-700"
                          )}
                        >
                          {child.label}
                        </Link>
                      );
                    })}
                  </div>
                )}
              </div>
            );
          }

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
