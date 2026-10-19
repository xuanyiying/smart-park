"use client";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { ORDER_STATUS_CONFIG, ORDER_STATUS_OPTIONS_WITH_ALL, PAY_METHOD_LABELS } from "@/constants/order";
import { adminService, Order } from "@/services/admin";
import { formatCents } from "@/lib/utils";
import { Loader2, Search } from "lucide-react";
import { useEffect, useState } from "react";

export default function OrdersPage() {
  const [search, setSearch] = useState("");
  const [statusFilter, setStatusFilter] = useState<string | null>("all");
  const [orders, setOrders] = useState<Order[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetchOrders();
  }, []);

  const fetchOrders = async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await adminService.listOrders(1, 100);
      setOrders(data.list || []);
    } catch (err) {
      setError("网络错误，请稍后重试");
      console.error("Failed to fetch orders:", err);
    } finally {
      setLoading(false);
    }
  };

  const filteredOrders = orders.filter((order) => {
    const plateNumber = order.plateNumber || "";
    const matchSearch = plateNumber.toLowerCase().includes(search.toLowerCase());
    const matchStatus = statusFilter === "all" || order.status === statusFilter;
    return matchSearch && matchStatus;
  });

  const handleStatusChange = (value: string | null) => {
    setStatusFilter(value || "all");
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold text-gray-900">订单管理</h1>
        <p className="text-gray-500 mt-1">管理停车订单</p>
      </div>

      <Card>
        <CardHeader className="pb-3">
          <div className="flex items-center gap-4">
            <div className="relative flex-1 max-w-sm">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
              <Input
                placeholder="搜索车牌号..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="pl-10"
              />
            </div>
            <Select value={statusFilter || "all"} onValueChange={handleStatusChange}>
              <SelectTrigger className="w-[140px]">
                <SelectValue>{ORDER_STATUS_CONFIG[statusFilter || "all"].label}</SelectValue>
              </SelectTrigger>
              <SelectContent>
                {ORDER_STATUS_OPTIONS_WITH_ALL.map((option) => (
                  <SelectItem key={option.value} value={option.value}>
                    {option.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </CardHeader>
        <CardContent>
          {loading ? (
            <div className="flex items-center justify-center py-12">
              <Loader2 className="w-8 h-8 animate-spin text-gray-400" />
            </div>
          ) : error ? (
            <div className="text-center py-12">
              <p className="text-red-500">{error}</p>
              <Button variant="outline" onClick={fetchOrders} className="mt-4">
                重试
              </Button>
            </div>
          ) : (
            <div className="border rounded-lg">
              <table className="w-full">
                <thead className="bg-gray-50 border-b">
                  <tr>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">订单号</th>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">车牌号</th>
                    <th className="text-right px-4 py-3 text-sm font-medium text-gray-500">金额</th>
                    <th className="text-right px-4 py-3 text-sm font-medium text-gray-500">优惠</th>
                    <th className="text-right px-4 py-3 text-sm font-medium text-gray-500">实付</th>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">状态</th>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">支付时间</th>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">支付方式</th>
                  </tr>
                </thead>
                <tbody>
                  {filteredOrders.length === 0 ? (
                    <tr>
                      <td colSpan={8} className="px-4 py-8 text-center text-gray-500">
                        暂无订单数据
                      </td>
                    </tr>
                  ) : (
                    filteredOrders.map((order) => (
                      <tr key={order.id} className="border-b last:border-0 hover:bg-gray-50">
                        <td className="px-4 py-3 font-mono text-sm">{order.id}</td>
                        <td className="px-4 py-3 font-medium">{order.plateNumber}</td>
                        <td className="px-4 py-3 text-right">¥{formatCents(order.amount)}</td>
                        <td className="px-4 py-3 text-right text-green-600">-¥{formatCents(order.discountAmount)}</td>
                        <td className="px-4 py-3 text-right font-medium">¥{formatCents(order.finalAmount)}</td>
                        <td className="px-4 py-3">
                          <Badge variant={ORDER_STATUS_CONFIG[order.status]?.variant || "secondary"}>
                            {ORDER_STATUS_CONFIG[order.status]?.label || order.status}
                          </Badge>
                        </td>
                        <td className="px-4 py-3 text-gray-600">
                          {order.payTime || "-"}
                        </td>
                        <td className="px-4 py-3 text-gray-600">
                          {order.payMethod ? PAY_METHOD_LABELS[order.payMethod] || order.payMethod : "-"}
                        </td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}