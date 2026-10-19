"use client";

import Link from "next/link";
import { DeviceDialog } from "@/components/modules/device-dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { DEVICE_STATUS_LABELS, DEVICE_STATUS_OPTIONS_WITH_ALL, DEVICE_TYPE_LABELS } from "@/constants/device";
import { useApiMutation, useApiQuery } from "@/hooks/use-api";
import { DeviceInfo, deviceService } from "@/services/device";
import { AlertCircle, Edit, Loader2, Plus, Power, RefreshCw, Search, Trash2, Wifi, WifiOff } from "lucide-react";
import { useState } from "react";
import { useRouter } from "next/navigation";

export default function DevicesPage() {
  const [statusFilter, setStatusFilter] = useState<string>("all");
  const [searchQuery, setSearchQuery] = useState("");
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingDevice, setEditingDevice] = useState<DeviceInfo | null>(null);
  const router = useRouter();

  const { data: response, isLoading, error, refetch } = useApiQuery(
    ["devices"],
    () => deviceService.listDevices(),
    { staleTime: 30000 }
  );

  const devices = response?.list ?? [];

  const createMutation = useApiMutation(
    (data: {
      deviceId: string;
      deviceType: string;
      lotId: string;
      laneId: string;
      status: string;
    }) => deviceService.createDevice(data),
    {
      successMessage: "设备创建成功",
      invalidateKeys: [["devices"]],
    }
  );

  const updateMutation = useApiMutation(
    (data: { deviceId: string; data: { deviceType?: string; lotId?: string; laneId?: string; status?: string } }) =>
      deviceService.updateDevice(data.deviceId, data.data),
    {
      successMessage: "设备更新成功",
      invalidateKeys: [["devices"]],
    }
  );

  const deleteMutation = useApiMutation(
    (deviceId: string) => deviceService.deleteDevice(deviceId),
    {
      successMessage: "设备删除成功",
      invalidateKeys: [["devices"]],
    }
  );

  const heartbeatMutation = useApiMutation(
    (deviceId: string) => deviceService.getDeviceHeartbeat(deviceId),
    {
      successMessage: "心跳状态已刷新",
    }
  );

  const openGateMutation = useApiMutation(
    ({ deviceId }: { deviceId: string }) =>
      deviceService.sendCommand(deviceId, "open_gate"),
    {
      successMessage: "开闸指令已发送",
    }
  );

  const filteredDevices = devices.filter((d) => {
    const matchesStatus = statusFilter === "all" || d.status === statusFilter;
    const deviceId = d.deviceId || "";
    const matchesSearch = deviceId.toLowerCase().includes(searchQuery.toLowerCase());
    return matchesStatus && matchesSearch;
  });

  const handleOpenGate = (deviceId: string) => {
    openGateMutation.mutate({ deviceId });
  };

  const handleStatusChange = (value: string | null) => {
    setStatusFilter(value ?? "all");
  };

  const handleAddDevice = () => {
    setEditingDevice(null);
    setDialogOpen(true);
  };

  const handleEditDevice = (device: DeviceInfo) => {
    setEditingDevice(device);
    setDialogOpen(true);
  };

  const handleDeleteDevice = async (deviceId: string) => {
    if (confirm("确定要删除此设备吗？")) {
      await deleteMutation.mutateAsync(deviceId);
    }
  };

  const handleRefreshHeartbeat = async (deviceId: string) => {
    await heartbeatMutation.mutateAsync(deviceId);
  };

  const handleSubmit = async (data: {
    deviceId: string;
    deviceType: string;
    lotId: string;
    laneId: string;
    status: string;
  }) => {
    if (editingDevice) {
      await updateMutation.mutateAsync({
        deviceId: editingDevice.deviceId,
        data: {
          deviceType: data.deviceType,
          lotId: data.lotId,
          laneId: data.laneId,
          status: data.status,
        },
      });
    } else {
      await createMutation.mutateAsync(data);
    }
    setDialogOpen(false);
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold text-gray-900">设备管理</h1>
          <p className="text-gray-500 mt-1">管理停车场设备、监控心跳状态与远程控制</p>
        </div>
        <Button onClick={handleAddDevice}>
          <Plus className="w-4 h-4 mr-2" />
          新增设备
        </Button>
      </div>

      <Card>
        <CardHeader className="pb-3">
          <div className="flex items-center gap-4">
            <div className="relative flex-1 max-w-sm">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
              <Input
                placeholder="搜索设备ID..."
                className="pl-10"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
              />
            </div>
            <Select value={statusFilter} onValueChange={handleStatusChange}>
              <SelectTrigger className="w-[140px]">
                <SelectValue>{DEVICE_STATUS_LABELS[statusFilter]}</SelectValue>
              </SelectTrigger>
              <SelectContent>
                {DEVICE_STATUS_OPTIONS_WITH_ALL.map((option) => (
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
            <div className="flex items-center justify-center py-12">
              <Loader2 className="w-8 h-8 animate-spin text-gray-400" />
            </div>
          ) : error ? (
            <div className="text-center py-12">
              <div className="flex flex-col items-center gap-3">
                <div className="w-16 h-16 rounded-full bg-red-50 flex items-center justify-center">
                  <AlertCircle className="w-8 h-8 text-red-500" />
                </div>
                <div>
                  <p className="text-red-600 font-medium">加载设备列表失败</p>
                  <p className="text-sm text-gray-500 mt-1">请检查网络连接或稍后重试</p>
                </div>
              </div>
              <Button variant="outline" onClick={() => refetch()} className="mt-6">
                <RefreshCw className="w-4 h-4 mr-2" />
                重试
              </Button>
            </div>
          ) : (
            <div className="border rounded-lg">
              <table className="w-full">
                <thead className="bg-gray-50 border-b">
                  <tr>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">设备ID</th>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">设备类型</th>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">在线状态</th>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">设备状态</th>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">最后心跳</th>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">操作</th>
                  </tr>
                </thead>
                <tbody>
                  {filteredDevices.length === 0 ? (
                    <tr>
                      <td colSpan={6} className="px-4 py-8 text-center text-gray-500">
                        暂无设备数据
                      </td>
                    </tr>
                  ) : (
                    filteredDevices.map((device) => (
                      <tr key={device.deviceId} className="border-b last:border-0 hover:bg-gray-50">
                        <td className="px-4 py-3 font-medium">{device.deviceId}</td>
                        <td className="px-4 py-3">
                          <Badge variant="outline">
                            {DEVICE_TYPE_LABELS[device.deviceType] || device.deviceType}
                          </Badge>
                        </td>
                        <td className="px-4 py-3">
                          <div className="flex items-center gap-2">
                            {device.online ? (
                              <>
                                <span className="w-2 h-2 rounded-full bg-green-500"></span>
                                <span className="text-green-600 flex items-center gap-1">
                                  <Wifi className="w-3 h-3" />
                                  在线
                                </span>
                              </>
                            ) : (
                              <>
                                <span className="w-2 h-2 rounded-full bg-red-500"></span>
                                <span className="text-red-600 flex items-center gap-1">
                                  <WifiOff className="w-3 h-3" />
                                  离线
                                </span>
                              </>
                            )}
                          </div>
                        </td>
                        <td className="px-4 py-3">
                          <Badge
                            variant={
                              device.status === "active"
                                ? "default"
                                : device.status === "error"
                                  ? "destructive"
                                  : "secondary"
                            }
                          >
                            {DEVICE_STATUS_LABELS[device.status] || device.status}
                          </Badge>
                        </td>
                        <td className="px-4 py-3 text-gray-600">{device.lastHeartbeat ? new Date(device.lastHeartbeat).toLocaleString("zh-CN") : "-"}</td>
                        <td className="px-4 py-3">
                          <div className="flex items-center gap-2">
                            <Button
                              variant="outline"
                              size="sm"
                              onClick={() => router.push(`/devices/${device.deviceId}`)}
                            >
                              详情
                            </Button>
                            <Button
                              variant="outline"
                              size="sm"
                              onClick={() => handleOpenGate(device.deviceId)}
                              disabled={!device.online || openGateMutation.isPending}
                            >
                              {openGateMutation.isPending ? (
                                <Loader2 className="w-4 h-4 mr-1 animate-spin" />
                              ) : (
                                <Power className="w-4 h-4 mr-1" />
                              )}
                              开闸
                            </Button>
                            <Button
                              variant="outline"
                              size="sm"
                              onClick={() => handleRefreshHeartbeat(device.deviceId)}
                              disabled={heartbeatMutation.isPending}
                            >
                              <RefreshCw className="w-4 h-4" />
                            </Button>
                            <Button
                              variant="outline"
                              size="sm"
                              onClick={() => handleEditDevice(device)}
                            >
                              <Edit className="w-4 h-4" />
                            </Button>
                            <Button
                              variant="outline"
                              size="sm"
                              onClick={() => handleDeleteDevice(device.deviceId)}
                              disabled={deleteMutation.isPending}
                              className="text-red-600 hover:bg-red-50"
                            >
                              <Trash2 className="w-4 h-4" />
                            </Button>
                          </div>
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

      <DeviceDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        device={editingDevice}
        onSubmit={handleSubmit}
        isSubmitting={createMutation.isPending || updateMutation.isPending}
      />
    </div>
  );
}
