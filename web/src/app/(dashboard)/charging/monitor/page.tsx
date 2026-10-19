"use client";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { useApiQuery } from "@/hooks/use-api";
import { formatCents } from "@/lib/utils";
import {
  Session,
  chargingService,
} from "@/services/charging";
import { Activity, Loader2, RefreshCw, WifiOff } from "lucide-react";
import { useState } from "react";

const SESSION_STATUS_LABELS: Record<string, string> = {
  pending: "等待中",
  charging: "充电中",
  completed: "已完成",
  cancelled: "已取消",
  expired: "已过期",
};

const SESSION_STATUS_VARIANT: Record<
  string,
  "default" | "secondary" | "destructive" | "outline"
> = {
  pending: "outline",
  charging: "default",
  completed: "secondary",
  cancelled: "secondary",
  expired: "destructive",
};

const PAYMENT_STATUS_LABELS: Record<string, string> = {
  pending: "待支付",
  paid: "已支付",
  refunded: "已退款",
  failed: "支付失败",
};

const PAYMENT_STATUS_VARIANT: Record<
  string,
  "default" | "secondary" | "destructive" | "outline"
> = {
  pending: "outline",
  paid: "default",
  refunded: "secondary",
  failed: "destructive",
};

export default function ChargingMonitorPage() {
  const [filter, setFilter] = useState<"active" | "all">("active");

  const {
    data: response,
    isLoading,
    error,
    refetch,
  } = useApiQuery<Session[]>(
    ["charging-sessions", filter],
    () => chargingService.listUserSessions("", 1, 100),
    {
      refetchInterval: 30000,
      refetchOnWindowFocus: false,
    }
  );

  const allSessions: Session[] = response || [];
  const sessions =
    filter === "active"
      ? allSessions.filter(
          (s) => s.status === "charging" || s.status === "pending"
        )
      : allSessions;

  const activeCount = allSessions.filter((s) => s.status === "charging").length;

  const renderErrorState = () => {
    const errorMessage = (error as { message?: string })?.message || "";
    const isNetworkError =
      errorMessage.includes("Network Error") ||
      errorMessage.includes("ECONNREFUSED");

    return (
      <div className="flex flex-col items-center justify-center py-16 text-center">
        <div className="w-16 h-16 rounded-full bg-red-50 flex items-center justify-center mb-4">
          <WifiOff className="w-8 h-8 text-red-400" />
        </div>
        <h3 className="text-lg font-medium text-gray-900 mb-1">
          {isNetworkError ? "网络连接失败" : "加载失败"}
        </h3>
        <p className="text-gray-500 mb-4 max-w-sm">
          获取充电会话数据时出现问题，请稍后重试
        </p>
        <Button variant="outline" onClick={() => refetch()}>
          <RefreshCw className="w-4 h-4 mr-2" />
          重新加载
        </Button>
      </div>
    );
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold text-gray-900">充电监控</h1>
          <p className="text-gray-500 mt-1">实时监控充电会话状态</p>
        </div>
        <div className="flex items-center gap-3">
          <div className="flex items-center gap-2 text-sm">
            <span className="inline-block w-2 h-2 rounded-full bg-green-500 animate-pulse" />
            <span className="text-gray-600">
              {activeCount} 个活跃会话
            </span>
          </div>
          <Button
            variant={filter === "active" ? "default" : "outline"}
            size="sm"
            onClick={() => setFilter("active")}
          >
            活跃会话
          </Button>
          <Button
            variant={filter === "all" ? "default" : "outline"}
            size="sm"
            onClick={() => setFilter("all")}
          >
            全部会话
          </Button>
        </div>
      </div>

      <Card>
        <CardHeader className="pb-3">
          <div className="flex items-center justify-between">
            <div className="text-lg font-medium">
              {filter === "active" ? "活跃充电会话" : "全部充电会话"}
            </div>
            <Button variant="ghost" size="sm" onClick={() => refetch()}>
              <RefreshCw className="w-4 h-4" />
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="flex items-center justify-center py-16">
              <Loader2 className="w-8 h-8 animate-spin text-gray-400" />
            </div>
          ) : error ? (
            renderErrorState()
          ) : sessions.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-16 text-center">
              <div className="w-16 h-16 rounded-full bg-gray-100 flex items-center justify-center mb-4">
                <Activity className="w-8 h-8 text-gray-400" />
              </div>
              <h3 className="text-lg font-medium text-gray-900 mb-1">
                暂无{filter === "active" ? "活跃" : ""}会话
              </h3>
              <p className="text-gray-500">
                {filter === "active"
                  ? "当前没有正在进行的充电会话"
                  : "暂无充电会话记录"}
              </p>
            </div>
          ) : (
            <div className="border rounded-lg">
              <table className="w-full">
                <thead className="bg-gray-50 border-b">
                  <tr>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">
                      会话ID
                    </th>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">
                      充电桩ID
                    </th>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">
                      车牌号
                    </th>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">
                      开始时间
                    </th>
                    <th className="text-right px-4 py-3 text-sm font-medium text-gray-500">
                      充电量(kWh)
                    </th>
                    <th className="text-right px-4 py-3 text-sm font-medium text-gray-500">
                      电费
                    </th>
                    <th className="text-right px-4 py-3 text-sm font-medium text-gray-500">
                      总金额
                    </th>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">
                      状态
                    </th>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">
                      支付状态
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {sessions.map((session) => (
                    <tr
                      key={session.id}
                      className="border-b last:border-0 hover:bg-gray-50"
                    >
                      <td className="px-4 py-3 font-mono text-sm">
                        {session.id.slice(0, 8)}...
                      </td>
                      <td className="px-4 py-3 font-mono text-sm">
                        {session.stationId.slice(0, 8)}...
                      </td>
                      <td className="px-4 py-3 font-medium">
                        {session.vehiclePlate}
                      </td>
                      <td className="px-4 py-3 text-gray-600">
                        {session.startTime
                          ? new Date(session.startTime).toLocaleString("zh-CN")
                          : "-"}
                      </td>
                      <td className="px-4 py-3 text-right">
                        {session.chargedEnergy.toFixed(2)}
                      </td>
                      <td className="px-4 py-3 text-right">
                        ¥{formatCents(session.cost)}
                      </td>
                      <td className="px-4 py-3 text-right font-medium">
                        ¥{formatCents(session.totalAmount)}
                      </td>
                      <td className="px-4 py-3">
                        <Badge
                          variant={
                            SESSION_STATUS_VARIANT[session.status] ||
                            "secondary"
                          }
                        >
                          {SESSION_STATUS_LABELS[session.status] ||
                            session.status}
                        </Badge>
                      </td>
                      <td className="px-4 py-3">
                        <Badge
                          variant={
                            PAYMENT_STATUS_VARIANT[session.paymentStatus] ||
                            "secondary"
                          }
                        >
                          {PAYMENT_STATUS_LABELS[session.paymentStatus] ||
                            session.paymentStatus}
                        </Badge>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
