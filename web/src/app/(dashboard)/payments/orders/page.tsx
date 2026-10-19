"use client";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  PAYMENT_STATUS_LABELS,
  PAYMENT_STATUS_VARIANT,
  PAYMENT_METHOD_LABELS,
} from "@/constants/payment";
import { useApiMutation, useApiQuery } from "@/hooks/use-api";
import { formatCents } from "@/lib/utils";
import { paymentService, PaymentOrder } from "@/services/payment";
import { FileText, Loader2, RefreshCw, RotateCcw, WifiOff } from "lucide-react";
import { useState } from "react";

const statusFilterOptions = [
  { value: "all", label: "全部" },
  { value: "pending", label: "待支付" },
  { value: "paid", label: "已支付" },
  { value: "refunded", label: "已退款" },
  { value: "failed", label: "失败" },
];

export default function PaymentOrdersPage() {
  const [statusFilter, setStatusFilter] = useState("all");
  const [refundDialogOpen, setRefundDialogOpen] = useState(false);
  const [refundingOrder, setRefundingOrder] = useState<PaymentOrder | null>(
    null
  );
  const [refundReason, setRefundReason] = useState("");

  const {
    data: response,
    isLoading,
    error,
    refetch,
  } = useApiQuery<{ list: PaymentOrder[]; total: number }>(
    ["payment-orders", statusFilter],
    () =>
      paymentService.listOrders(
        1,
        100,
        statusFilter === "all" ? undefined : statusFilter
      ),
    { refetchOnWindowFocus: false }
  );

  const orders: PaymentOrder[] = response?.list || [];

  const refundMutation = useApiMutation(
    ({ orderId, reason }: { orderId: string; reason: string }) =>
      paymentService.refund(orderId, reason),
    {
      successMessage: "退款申请已提交",
      invalidateKeys: [["payment-orders"]],
    }
  );

  const handleRefundClick = (order: PaymentOrder) => {
    setRefundingOrder(order);
    setRefundReason("");
    setRefundDialogOpen(true);
  };

  const handleConfirmRefund = async () => {
    if (refundingOrder) {
      await refundMutation.mutateAsync({
        orderId: refundingOrder.id,
        reason: refundReason || "管理员操作退款",
      });
      setRefundDialogOpen(false);
      setRefundingOrder(null);
      setRefundReason("");
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
          {isNetworkError
            ? "无法连接到服务器，请检查网络连接或稍后重试"
            : "获取支付订单时出现问题，请稍后重试"}
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
      <div>
        <h1 className="text-2xl font-semibold text-gray-900">支付订单</h1>
        <p className="text-gray-500 mt-1">管理支付订单和退款操作</p>
      </div>

      <Card>
        <CardHeader className="pb-3">
          <div className="flex items-center gap-4">
            <Select
              value={statusFilter}
              onValueChange={(v) => v && setStatusFilter(v)}
            >
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
          </div>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="flex items-center justify-center py-16">
              <Loader2 className="w-8 h-8 animate-spin text-gray-400" />
            </div>
          ) : error ? (
            renderErrorState()
          ) : orders.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-16 text-center">
              <div className="w-16 h-16 rounded-full bg-gray-100 flex items-center justify-center mb-4">
                <FileText className="w-8 h-8 text-gray-400" />
              </div>
              <h3 className="text-lg font-medium text-gray-900 mb-1">
                暂无支付订单
              </h3>
              <p className="text-gray-500">没有匹配的支付订单记录</p>
            </div>
          ) : (
            <div className="border rounded-lg">
              <table className="w-full">
                <thead className="bg-gray-50 border-b">
                  <tr>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">
                      订单号
                    </th>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">
                      车牌号
                    </th>
                    <th className="text-right px-4 py-3 text-sm font-medium text-gray-500">
                      金额
                    </th>
                    <th className="text-right px-4 py-3 text-sm font-medium text-gray-500">
                      折扣
                    </th>
                    <th className="text-right px-4 py-3 text-sm font-medium text-gray-500">
                      实付
                    </th>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">
                      支付方式
                    </th>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">
                      状态
                    </th>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">
                      支付时间
                    </th>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">
                      操作
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {orders.map((order) => (
                    <tr
                      key={order.id}
                      className="border-b last:border-0 hover:bg-gray-50"
                    >
                      <td className="px-4 py-3 font-mono text-sm">
                        {order.id}
                      </td>
                      <td className="px-4 py-3 font-medium">
                        {order.plateNumber}
                      </td>
                      <td className="px-4 py-3 text-right">
                        ¥{formatCents(order.amount)}
                      </td>
                      <td className="px-4 py-3 text-right text-green-600">
                        -¥{formatCents(order.discountAmount)}
                      </td>
                      <td className="px-4 py-3 text-right font-medium">
                        ¥{formatCents(order.finalAmount)}
                      </td>
                      <td className="px-4 py-3 text-gray-600">
                        {PAYMENT_METHOD_LABELS[order.payMethod] || order.payMethod || "-"}
                      </td>
                      <td className="px-4 py-3">
                        <Badge
                          variant={
                            PAYMENT_STATUS_VARIANT[order.status] || "secondary"
                          }
                        >
                          {PAYMENT_STATUS_LABELS[order.status] || order.status}
                        </Badge>
                      </td>
                      <td className="px-4 py-3 text-gray-600 text-sm">
                        {order.payTime
                          ? new Date(order.payTime).toLocaleString("zh-CN")
                          : "-"}
                      </td>
                      <td className="px-4 py-3">
                        {order.status === "paid" && (
                          <Button
                            variant="ghost"
                            size="sm"
                            className="text-amber-600"
                            onClick={() => handleRefundClick(order)}
                          >
                            <RotateCcw className="w-4 h-4 mr-1" />
                            退款
                          </Button>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Refund confirmation dialog */}
      <Dialog open={refundDialogOpen} onOpenChange={setRefundDialogOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>确认退款</DialogTitle>
          </DialogHeader>
          <div className="py-4 space-y-4">
            <p className="text-sm text-gray-600">
              确定要对订单{" "}
              <strong className="font-mono">{refundingOrder?.id}</strong>{" "}
              进行退款吗？退款金额为{" "}
              <strong>¥{formatCents(refundingOrder?.finalAmount)}</strong>，此操作无法撤销。
            </p>
            <div>
              <label className="text-sm font-medium text-gray-700 mb-1 block">
                退款原因
              </label>
              <input
                type="text"
                value={refundReason}
                onChange={(e) => setRefundReason(e.target.value)}
                placeholder="请输入退款原因（选填）"
                className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
              />
            </div>
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setRefundDialogOpen(false)}
              disabled={refundMutation.isPending}
            >
              取消
            </Button>
            <Button
              variant="destructive"
              onClick={handleConfirmRefund}
              disabled={refundMutation.isPending}
            >
              {refundMutation.isPending ? (
                <>
                  <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                  退款中...
                </>
              ) : (
                "确认退款"
              )}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
