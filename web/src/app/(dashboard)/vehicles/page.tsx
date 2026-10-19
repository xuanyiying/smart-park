"use client";

import { VehicleDialog } from "@/components/modules/vehicle-dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { VEHICLE_TYPE_LABELS, VEHICLE_TYPE_OPTIONS_WITH_ALL } from "@/constants/vehicle";
import { useApiMutation, useApiQuery } from "@/hooks/use-api";
import { adminService, Vehicle } from "@/services/admin";
import { Loader2, Plus, RefreshCw, Search, WifiOff } from "lucide-react";
import { useState } from "react";

export default function VehiclesPage() {
  const [search, setSearch] = useState("");
  const [typeFilter, setTypeFilter] = useState<string>("all");
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingVehicle, setEditingVehicle] = useState<Vehicle | null>(null);

  const {
    data: response,
    isLoading,
    error,
    refetch,
  } = useApiQuery(
    ["vehicles", typeFilter],
    () => adminService.listVehicles(1, 100, typeFilter === "all" ? undefined : typeFilter),
    {
      refetchOnWindowFocus: false,
    }
  );

  const createMutation = useApiMutation(
    (data: {
      plateNumber: string;
      vehicleType: string;
      ownerName: string;
      ownerPhone: string;
      monthlyValidUntil: string;
    }) => adminService.createVehicle(data),
    {
      successMessage: "车辆添加成功",
      invalidateKeys: [["vehicles"]],
    }
  );

  const updateMutation = useApiMutation(
    (data: {
      id: string; data: {
        plateNumber: string;
        vehicleType: string;
        ownerName: string;
        ownerPhone: string;
        monthlyValidUntil: string;
      }
    }) => adminService.updateVehicle(data.id, data.data),
    {
      successMessage: "车辆更新成功",
      invalidateKeys: [["vehicles"]],
    }
  );

  const vehicles = response?.list || [];

  const getVehicleTypeDisplay = (type: string) => VEHICLE_TYPE_LABELS[type] || type;

  const filteredVehicles = vehicles.filter((v: Vehicle) => {
    const plateNumber = v.plateNumber || "";
    const matchSearch = plateNumber.toLowerCase().includes(search.toLowerCase());
    return matchSearch;
  });

  const handleTypeChange = (value: string | null) => {
    if (value) setTypeFilter(value);
  };

  const handleAddVehicle = () => {
    setEditingVehicle(null);
    setDialogOpen(true);
  };

  const handleEditVehicle = (vehicle: Vehicle) => {
    setEditingVehicle(vehicle);
    setDialogOpen(true);
  };

  const handleSubmit = async (data: {
    plateNumber: string;
    vehicleType: string;
    ownerName: string;
    ownerPhone: string;
    monthlyValidUntil: string;
  }) => {
    if (editingVehicle) {
      await updateMutation.mutateAsync({ id: editingVehicle.id, data });
      setDialogOpen(false);
      return;
    }

    await createMutation.mutateAsync(data);
    setDialogOpen(false);
  };

  // 渲染空状态
  const renderEmptyState = () => (
    <div className="flex flex-col items-center justify-center py-16 text-center">
      <div className="w-16 h-16 rounded-full bg-gray-100 flex items-center justify-center mb-4">
        <Search className="w-8 h-8 text-gray-400" />
      </div>
      <h3 className="text-lg font-medium text-gray-900 mb-1">
        {search || typeFilter !== "all" ? "未找到匹配的车辆" : "暂无车辆"}
      </h3>
      <p className="text-gray-500 mb-4">
        {search || typeFilter !== "all"
          ? "请尝试调整搜索条件或筛选条件"
          : "您还没有添加任何车辆，点击下方按钮添加第一辆车"}
      </p>
      {!search && typeFilter === "all" && (
        <Button onClick={handleAddVehicle}>
          <Plus className="w-4 h-4 mr-2" />
          新增车辆
        </Button>
      )}
    </div>
  );

  // 渲染错误状态
  const renderErrorState = () => {
    const errorMessage = (error as { message?: string })?.message || "";
    const isNetworkError = errorMessage.includes("Network Error") || errorMessage.includes("ECONNREFUSED");
    const isTimeout = errorMessage.includes("timeout");

    return (
      <div className="flex flex-col items-center justify-center py-16 text-center">
        <div className="w-16 h-16 rounded-full bg-red-50 flex items-center justify-center mb-4">
          <WifiOff className="w-8 h-8 text-red-400" />
        </div>
        <h3 className="text-lg font-medium text-gray-900 mb-1">
          {isNetworkError ? "网络连接失败" : isTimeout ? "请求超时" : "加载失败"}
        </h3>
        <p className="text-gray-500 mb-4 max-w-sm">
          {isNetworkError
            ? "无法连接到服务器，请检查网络连接或稍后重试"
            : isTimeout
              ? "服务器响应时间过长，请稍后重试"
              : "获取车辆列表时出现问题，请稍后重试"}
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
          <h1 className="text-2xl font-semibold text-gray-900">车辆管理</h1>
          <p className="text-gray-500 mt-1">管理车辆信息</p>
        </div>
        <Button onClick={handleAddVehicle}>
          <Plus className="w-4 h-4 mr-2" />
          新增车辆
        </Button>
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
            <Select value={typeFilter} onValueChange={handleTypeChange}>
              <SelectTrigger className="w-[140px]">
                <SelectValue>{VEHICLE_TYPE_LABELS[typeFilter] || "全部"}</SelectValue>
              </SelectTrigger>
              <SelectContent>
                {VEHICLE_TYPE_OPTIONS_WITH_ALL.map((option) => (
                  <SelectItem key={option.value} value={option.value}>
                    {option.label}
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
          ) : filteredVehicles.length === 0 ? (
            renderEmptyState()
          ) : (
            <div className="border rounded-lg">
              <table className="w-full">
                <thead className="bg-gray-50 border-b">
                  <tr>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">车牌号</th>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">车辆类型</th>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">车主姓名</th>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">联系方式</th>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">月租有效期</th>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">操作</th>
                  </tr>
                </thead>
                <tbody>
                  {filteredVehicles.map((vehicle: Vehicle) => (
                    <tr key={vehicle.id} className="border-b last:border-0 hover:bg-gray-50">
                      <td className="px-4 py-3 font-medium">{vehicle.plateNumber}</td>
                      <td className="px-4 py-3">
                        <Badge variant={vehicle.vehicleType === "monthly" ? "default" : "secondary"}>
                          {getVehicleTypeDisplay(vehicle.vehicleType)}
                        </Badge>
                      </td>
                      <td className="px-4 py-3">{vehicle.ownerName}</td>
                      <td className="px-4 py-3 text-gray-600">{vehicle.ownerPhone}</td>
                      <td className="px-4 py-3 text-gray-600">
                        {vehicle.monthlyValidUntil || "-"}
                      </td>
                      <td className="px-4 py-3">
                        <Button
                          variant="ghost"
                          size="sm"
                          onClick={() => handleEditVehicle(vehicle)}
                        >
                          编辑
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

      <VehicleDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        vehicle={editingVehicle}
        onSubmit={handleSubmit}
        isLoading={createMutation.isPending || updateMutation.isPending}
      />
    </div>
  );
}
