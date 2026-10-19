"use client";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import {
  RECONCILIATION_STATUS_LABELS,
  RECONCILIATION_STATUS_VARIANT,
  PAYMENT_METHOD_LABELS,
} from "@/constants/payment";
import { useApiMutation, useApiQuery } from "@/hooks/use-api";
import { formatCents } from "@/lib/utils";
import {
  paymentService,
  ReconciliationRecord,
  ReconciliationReport,
} from "@/services/payment";
import { Calendar, CheckCircle2, AlertTriangle, Clock, Loader2, RefreshCw, Wrench, WifiOff } from "lucide-react";
import { useState } from "react";

function getTodayStr(): string {
  const d = new Date();
  return d.toISOString().slice(0, 10);
}

export default function PaymentReconciliationPage() {
  const [selectedDate, setSelectedDate] = useState(getTodayStr());
  const [dailyRecords, setDailyRecords] = useState<ReconciliationRecord[]>([]);

  const {
    data: initialReport,
    isLoading,
    error,
    refetch,
  } = useApiQuery<ReconciliationReport>(
    ["reconciliation", selectedDate],
    () => paymentService.getReconciliationReport(selectedDate, selectedDate),
    { refetchOnWindowFocus: false }
  );

  const reconcileMutation = useApiMutation(
    (date: string) => paymentService.reconcileDaily(date),
    {
      successMessage: "日核对执行成功",
      invalidateKeys: [["reconciliation"]],
    }
  );

  const fixMutation = useApiMutation(
    (orderIds: string[]) => paymentService.fixMismatchedOrders(orderIds),
    {
      successMessage: "修正成功",
      invalidateKeys: [["reconciliation"]],
    }
  );

  const currentReport = initialReport;
  const records: ReconciliationRecord[] = dailyRecords;

  const handleReconcile = async () => {
    try {
      const result = await reconcileMutation.mutateAsync(selectedDate);
      setDailyRecords(result);
    } catch {
      // error handled by useApiMutation
    }
  };

  const handleFix = async (orderId: string) => {
    await fixMutation.mutateAsync([orderId]);
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
            : "获取对账数据时出现问题，请稍后重试"}
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
        <h1 className="text-2xl font-semibold text-gray-900">支付对账</h1>
        <p className="text-gray-500 mt-1">核对支付订单与实际交易记录</p>
      </div>

      {/* Date picker + reconcile button */}
      <Card>
        <CardContent className="pt-6">
          <div className="flex items-center gap-4">
            <div className="flex items-center gap-2">
              <Calendar className="w-4 h-4 text-gray-500" />
              <Input
                type="date"
                value={selectedDate}
                onChange={(e) => setSelectedDate(e.target.value)}
                className="w-[180px]"
              />
            </div>
            <Button
              onClick={handleReconcile}
              disabled={reconcileMutation.isPending}
            >
              {reconcileMutation.isPending ? (
                <>
                  <Loader2 className="w-4 h-4 mr-2 animate-spin" />
                  核对中...
                </>
              ) : (
                "执行日核对"
              )}
            </Button>
          </div>
        </CardContent>
      </Card>

      {/* Stat cards */}
      <div className="grid grid-cols-3 gap-4">
        <div className="bg-white rounded-xl p-6 border border-gray-100 shadow-sm">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-gray-500 mb-1">匹配数</p>
              <p className="text-2xl font-semibold">
                {currentReport?.matchedOrders ?? "-"}
              </p>
            </div>
            <div className="w-12 h-12 bg-green-50 rounded-lg flex items-center justify-center text-green-600">
              <CheckCircle2 className="w-6 h-6" />
            </div>
          </div>
        </div>
        <div className="bg-white rounded-xl p-6 border border-gray-100 shadow-sm">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-gray-500 mb-1">不匹配数</p>
              <p className="text-2xl font-semibold text-red-600">
                {currentReport?.mismatchedOrders ?? "-"}
              </p>
            </div>
            <div className="w-12 h-12 bg-red-50 rounded-lg flex items-center justify-center text-red-600">
              <AlertTriangle className="w-6 h-6" />
            </div>
          </div>
        </div>
        <div className="bg-white rounded-xl p-6 border border-gray-100 shadow-sm">
          <div className="flex items-center justify-between">
            <div>
              <p className="text-sm text-gray-500 mb-1">总订单数</p>
              <p className="text-2xl font-semibold text-gray-900">
                {currentReport?.totalOrders ?? "-"}
              </p>
            </div>
            <div className="w-12 h-12 bg-amber-50 rounded-lg flex items-center justify-center text-amber-600">
              <Clock className="w-6 h-6" />
            </div>
          </div>
        </div>
      </div>

      {/* Reconciliation records table */}
      <Card>
        <CardHeader className="pb-3">
          <div className="text-lg font-medium">对账明细</div>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="flex items-center justify-center py-16">
              <Loader2 className="w-8 h-8 animate-spin text-gray-400" />
            </div>
          ) : error ? (
            renderErrorState()
          ) : records.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-16 text-center">
              <div className="w-16 h-16 rounded-full bg-gray-100 flex items-center justify-center mb-4">
                <Calendar className="w-8 h-8 text-gray-400" />
              </div>
              <h3 className="text-lg font-medium text-gray-900 mb-1">
                暂无对账数据
              </h3>
              <p className="text-gray-500">请选择日期并执行日核对</p>
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
                      支付方式
                    </th>
                    <th className="text-right px-4 py-3 text-sm font-medium text-gray-500">
                      订单金额
                    </th>
                    <th className="text-right px-4 py-3 text-sm font-medium text-gray-500">
                      实付金额
                    </th>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">
                      交易号
                    </th>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">
                      核对时间
                    </th>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">
                      状态
                    </th>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">
                      备注
                    </th>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">
                      操作
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {records.map((record) => (
                    <tr
                      key={record.id}
                      className="border-b last:border-0 hover:bg-gray-50"
                    >
                      <td className="px-4 py-3 font-mono text-sm">
                        {record.orderId}
                      </td>
                      <td className="px-4 py-3 text-gray-600">
                        {PAYMENT_METHOD_LABELS[record.paymentMethod] ||
                          record.paymentMethod}
                      </td>
                      <td className="px-4 py-3 text-right">
                        ¥{formatCents(record.orderAmount)}
                      </td>
                      <td className="px-4 py-3 text-right">
                        ¥{formatCents(record.paidAmount)}
                      </td>
                      <td className="px-4 py-3 font-mono text-sm text-gray-600">
                        {record.transactionId || "-"}
                      </td>
                      <td className="px-4 py-3 text-gray-600 text-sm">
                        {record.reconciliationTime
                          ? new Date(record.reconciliationTime).toLocaleString(
                              "zh-CN"
                            )
                          : "-"}
                      </td>
                      <td className="px-4 py-3">
                        <Badge
                          variant={
                            RECONCILIATION_STATUS_VARIANT[record.status] ||
                            "secondary"
                          }
                        >
                          {RECONCILIATION_STATUS_LABELS[record.status] ||
                            record.status}
                        </Badge>
                      </td>
                      <td className="px-4 py-3 text-sm text-gray-500">
                        {record.notes || "-"}
                      </td>
                      <td className="px-4 py-3">
                        {record.status === "mismatched" && (
                          <Button
                            variant="ghost"
                            size="sm"
                            className="text-blue-600"
                            onClick={() => handleFix(record.orderId)}
                            disabled={fixMutation.isPending}
                          >
                            {fixMutation.isPending ? (
                              <Loader2 className="w-4 h-4 animate-spin" />
                            ) : (
                              <>
                                <Wrench className="w-4 h-4 mr-1" />
                                修正
                              </>
                            )}
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
    </div>
  );
}
