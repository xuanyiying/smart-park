"use client";

import { useState } from "react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useApiQuery } from "@/hooks/use-api";
import {
  Session,
  ChargingSummary,
  chargingService,
} from "@/services/charging";
import { formatCents } from "@/lib/utils";
import { FileText, Loader2, Search, WifiOff } from "lucide-react";

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

const statusFilterOptions = [
  { value: "all", label: "全部状态" },
  { value: "charging", label: "充电中" },
  { value: "completed", label: "已完成" },
  { value: "cancelled", label: "已取消" },
  { value: "expired", label: "已过期" },
];

const paymentFilterOptions = [
  { value: "all", label: "全部" },
  { value: "pending", label: "待支付" },
  { value: "paid", label: "已支付" },
  { value: "failed", label: "支付失败" },
];

export default function ChargingOrdersPage() {
  const [search, setSearch] = useState("");
  const [statusFilter, setStatusFilter] = useState("all");
  const [paymentFilter, setPaymentFilter] = useState("all");
  const [detailDialogOpen, setDetailDialogOpen] = useState(false);
  const [selectedSession, setSelectedSession] = useState<Session | null>(null);
  const [summary, setSummary] = useState<ChargingSummary | null>(null);

  const {
    data: response,
    isLoading,
    error,
    refetch,
  } = useApiQuery<Session[]>(
    ["charging-orders"],
    () => chargingService.listUserSessions("", 1, 100),
    { refetchOnWindowFocus: false }
  );

  const allSessions: Session[] = response || [];

  const filteredSessions = allSessions.filter((session) => {
    const matchSearch =
      session.vehiclePlate.toLowerCase().includes(search.toLowerCase()) ||
      session.id.toLowerCase().includes(search.toLowerCase());
    const matchStatus =
      statusFilter === "all" || session.status === statusFilter;
    const matchPayment =
      paymentFilter === "all" || session.paymentStatus === paymentFilter;
    return matchSearch && matchStatus && matchPayment;
  });

  const handleViewDetail = async (session: Session) => {
    setSelectedSession(session);
    setDetailDialogOpen(true);
    try {
      const data = await chargingService.getChargingSummary(session.id);
      setSummary(data);
    } catch {
      setSummary(null);
    }
  };

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
          获取充电订单数据时出现问题，请稍后重试
        </p>
        <Button variant="outline" onClick={() => refetch()}>
          重试
        </Button>
      </div>
    );
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold text-gray-900">充电订单</h1>
        <p className="text-gray-500 mt-1">管理充电会话和支付订单</p>
      </div>

      <Card>
        <CardHeader className="pb-3">
          <div className="flex items-center gap-4">
            <div className="relative flex-1 max-w-sm">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
              <Input
                placeholder="搜索车牌号或会话ID..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="pl-10"
              />
            </div>
            <Select value={statusFilter} onValueChange={(v) => v && setStatusFilter(v)}>
              <SelectTrigger className="w-[140px]">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {statusFilterOptions.map((opt) => (
                  <SelectItem key={opt.value} value={opt.value}>
                    {opt.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <Select value={paymentFilter} onValueChange={(v) => v && setPaymentFilter(v)}>
              <SelectTrigger className="w-[140px]">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {paymentFilterOptions.map((opt) => (
                  <SelectItem key={opt.value} value={opt.value}>
                    {opt.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="flex items-center justify-center py-16">
              <Loader2 className="w-8 h-8 animate-spin text-gray-400" />
            </div>
          ) : error ? (
            renderErrorState()
          ) : filteredSessions.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-16 text-center">
              <div className="w-16 h-16 rounded-full bg-gray-100 flex items-center justify-center mb-4">
                <FileText className="w-8 h-8 text-gray-400" />
              </div>
              <h3 className="text-lg font-medium text-gray-900 mb-1">
                暂无充电订单
              </h3>
              <p className="text-gray-500">没有匹配的充电订单记录</p>
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
                      服务费
                    </th>
                    <th className="text-right px-4 py-3 text-sm font-medium text-gray-500">
                      总金额
                    </th>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">
                      状态
                    </th>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">
                      支付
                    </th>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">
                      操作
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {filteredSessions.map((session) => (
                    <tr
                      key={session.id}
                      className="border-b last:border-0 hover:bg-gray-50 cursor-pointer"
                      onClick={() => handleViewDetail(session)}
                    >
                      <td className="px-4 py-3 font-mono text-sm">
                        {session.id.slice(0, 8)}...
                      </td>
                      <td className="px-4 py-3 font-medium">
                        {session.vehiclePlate}
                      </td>
                      <td className="px-4 py-3 text-gray-600 text-sm">
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
                      <td className="px-4 py-3 text-right">
                        ¥{formatCents(session.serviceFee)}
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
                      <td className="px-4 py-3">
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={(e) => {
                            e.stopPropagation();
                            handleViewDetail(session);
                          }}
                        >
                          详情
                        </Button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </CardContent>
      </Card>

      <Dialog open={detailDialogOpen} onOpenChange={setDetailDialogOpen}>
        <DialogContent className="sm:max-w-lg">
          <DialogHeader>
            <DialogTitle>充电订单详情</DialogTitle>
          </DialogHeader>
          {selectedSession && (
            <div className="space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <p className="text-sm text-gray-500">会话ID</p>
                  <p className="font-mono text-sm">{selectedSession.id}</p>
                </div>
                <div>
                  <p className="text-sm text-gray-500">车牌号</p>
                  <p className="font-medium">{selectedSession.vehiclePlate}</p>
                </div>
                <div>
                  <p className="text-sm text-gray-500">开始时间</p>
                  <p className="text-sm">
                    {selectedSession.startTime
                      ? new Date(
                          selectedSession.startTime
                        ).toLocaleString("zh-CN")
                      : "-"}
                  </p>
                </div>
                <div>
                  <p className="text-sm text-gray-500">结束时间</p>
                  <p className="text-sm">
                    {selectedSession.endTime
                      ? new Date(
                          selectedSession.endTime
                        ).toLocaleString("zh-CN")
                      : "-"}
                  </p>
                </div>
              </div>

              <div className="border-t pt-4">
                <h4 className="text-sm font-medium mb-2">充电详情</h4>
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <p className="text-sm text-gray-500">充电量</p>
                    <p className="font-medium">
                      {selectedSession.chargedEnergy.toFixed(2)} kWh
                    </p>
                  </div>
                  {summary && (
                    <div>
                      <p className="text-sm text-gray-500">充电时长</p>
                      <p className="font-medium">
                        {summary.duration.toFixed(1)} 小时
                      </p>
                    </div>
                  )}
                </div>
              </div>

              <div className="border-t pt-4">
                <h4 className="text-sm font-medium mb-2">费用明细</h4>
                <div className="space-y-2">
                  <div className="flex justify-between text-sm">
                    <span className="text-gray-500">电费</span>
                    <span>¥{formatCents(selectedSession.cost)}</span>
                  </div>
                  <div className="flex justify-between text-sm">
                    <span className="text-gray-500">服务费</span>
                    <span>¥{formatCents(selectedSession.serviceFee)}</span>
                  </div>
                  <div className="flex justify-between font-medium border-t pt-2">
                    <span>合计</span>
                    <span>¥{formatCents(selectedSession.totalAmount)}</span>
                  </div>
                </div>
              </div>

              <div className="border-t pt-4">
                <h4 className="text-sm font-medium mb-2">支付信息</h4>
                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <p className="text-sm text-gray-500">支付状态</p>
                    <Badge
                      variant={
                        PAYMENT_STATUS_VARIANT[
                          selectedSession.paymentStatus
                        ] || "secondary"
                      }
                    >
                      {PAYMENT_STATUS_LABELS[selectedSession.paymentStatus] ||
                        selectedSession.paymentStatus}
                    </Badge>
                  </div>
                  {selectedSession.paymentMethod && (
                    <div>
                      <p className="text-sm text-gray-500">支付方式</p>
                      <p className="text-sm">
                        {selectedSession.paymentMethod}
                      </p>
                    </div>
                  )}
                  {selectedSession.payTime && (
                    <div>
                      <p className="text-sm text-gray-500">支付时间</p>
                      <p className="text-sm">
                        {new Date(selectedSession.payTime).toLocaleString(
                          "zh-CN"
                        )}
                      </p>
                    </div>
                  )}
                  {selectedSession.transactionId && (
                    <div>
                      <p className="text-sm text-gray-500">交易号</p>
                      <p className="font-mono text-sm">
                        {selectedSession.transactionId}
                      </p>
                    </div>
                  )}
                </div>
              </div>
            </div>
          )}
        </DialogContent>
      </Dialog>
    </div>
  );
}
