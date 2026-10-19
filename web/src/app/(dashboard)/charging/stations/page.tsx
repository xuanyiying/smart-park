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
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { useApiMutation, useApiQuery } from "@/hooks/use-api";
import {
  chargingService,
  CreateStationRequest,
  Station,
  UpdateStationRequest,
} from "@/services/charging";
import {
  Loader2,
  Pencil,
  Plus,
  RefreshCw,
  Trash2,
  WifiOff,
  Zap,
} from "lucide-react";
import Link from "next/link";
import { useState } from "react";

const STATION_TYPE_LABELS: Record<string, string> = {
  ac: "交流慢充",
  dc: "直流快充",
  fast_dc: "超充",
};

const STATION_STATUS_LABELS: Record<string, string> = {
  available: "可用",
  offline: "离线",
  maintenance: "维护中",
};

const STATION_STATUS_VARIANT: Record<string, "default" | "secondary" | "destructive"> = {
  available: "default",
  offline: "secondary",
  maintenance: "destructive",
};

const CONNECTOR_TYPE_LABELS: Record<string, string> = {
  ac: "交流",
  dc: "直流",
  fast_dc: "超充",
};

export default function ChargingStationsPage() {
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingStation, setEditingStation] = useState<Station | null>(null);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [deletingStation, setDeletingStation] = useState<Station | null>(null);

  const {
    data: response,
    isLoading,
    error,
    refetch,
  } = useApiQuery<Station[]>(
    ["charging-stations"],
    () => chargingService.listStations(undefined, 1, 100),
    { refetchOnWindowFocus: false }
  );

  const stations: Station[] = response || [];

  const createMutation = useApiMutation(
    (data: CreateStationRequest) => chargingService.createStation(data),
    {
      successMessage: "创建充电桩成功",
      invalidateKeys: [["charging-stations"]],
    }
  );

  const updateMutation = useApiMutation(
    ({ id, data }: { id: string; data: UpdateStationRequest }) =>
      chargingService.updateStation(id, data),
    {
      successMessage: "更新充电桩成功",
      invalidateKeys: [["charging-stations"]],
    }
  );

  const deleteMutation = useApiMutation(
    (id: string) => chargingService.deleteStation(id),
    {
      successMessage: "删除充电桩成功",
      invalidateKeys: [["charging-stations"]],
    }
  );

  const handleAddStation = () => {
    setEditingStation(null);
    setDialogOpen(true);
  };

  const handleEditStation = (station: Station) => {
    setEditingStation(station);
    setDialogOpen(true);
  };

  const handleDeleteClick = (station: Station) => {
    setDeletingStation(station);
    setDeleteDialogOpen(true);
  };

  const handleConfirmDelete = async () => {
    if (deletingStation) {
      await deleteMutation.mutateAsync(deletingStation.id);
      setDeleteDialogOpen(false);
      setDeletingStation(null);
    }
  };

  const isSubmitting =
    createMutation.isPending || updateMutation.isPending;

  const renderEmptyState = () => (
    <div className="flex flex-col items-center justify-center py-16 text-center">
      <div className="w-16 h-16 rounded-full bg-gray-100 flex items-center justify-center mb-4">
        <Zap className="w-8 h-8 text-gray-400" />
      </div>
      <h3 className="text-lg font-medium text-gray-900 mb-1">暂无充电桩</h3>
      <p className="text-gray-500 mb-4">
        您还没有创建任何充电桩，点击下方按钮添加第一个充电桩
      </p>
      <Button onClick={handleAddStation}>
        <Plus className="w-4 h-4 mr-2" />
        新增充电桩
      </Button>
    </div>
  );

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
            : "获取充电桩列表时出现问题，请稍后重试"}
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
          <h1 className="text-2xl font-semibold text-gray-900">充电桩管理</h1>
          <p className="text-gray-500 mt-1">管理充电站和充电桩设备</p>
        </div>
        <Button onClick={handleAddStation}>
          <Plus className="w-4 h-4 mr-2" />
          新增充电桩
        </Button>
      </div>

      <Card>
        <CardHeader className="pb-3">
          <div className="text-lg font-medium">充电桩列表</div>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="flex items-center justify-center py-16">
              <Loader2 className="w-8 h-8 animate-spin text-gray-400" />
            </div>
          ) : error ? (
            renderErrorState()
          ) : stations.length === 0 ? (
            renderEmptyState()
          ) : (
            <div className="space-y-4">
              {stations.map((station) => (
                <div
                  key={station.id}
                  className="border rounded-lg p-4 hover:bg-gray-50"
                >
                  <div className="flex items-start justify-between">
                    <div className="space-y-2">
                      <div className="flex items-center gap-3">
                        <Link
                          href={`/charging/stations/${station.id}`}
                          className="font-medium text-blue-600 hover:underline"
                        >
                          {station.name}
                        </Link>
                        <Badge
                          variant={
                            STATION_STATUS_VARIANT[station.status] || "secondary"
                          }
                        >
                          {STATION_STATUS_LABELS[station.status] || station.status}
                        </Badge>
                        <Badge variant="outline">
                          {STATION_TYPE_LABELS[station.stationType] ||
                            station.stationType}
                        </Badge>
                      </div>
                      <div className="text-sm text-gray-500 space-x-4">
                        <span>
                          连接器类型:{" "}
                          {CONNECTOR_TYPE_LABELS[station.connectorType] ||
                            station.connectorType}
                        </span>
                        <span>最大功率: {station.maxPower}kW</span>
                        <span>
                          可用枪数: {station.availableConnectors}/
                          {station.totalConnectors}
                        </span>
                        {station.location && (
                          <span>位置: {station.location}</span>
                        )}
                        {station.floor && <span>楼层: {station.floor}</span>}
                      </div>
                      <div className="text-xs text-gray-400">
                        创建时间: {station.createdAt}
                      </div>
                    </div>
                    <div className="flex gap-2">
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => handleEditStation(station)}
                      >
                        <Pencil className="w-4 h-4" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        className="text-red-500"
                        onClick={() => handleDeleteClick(station)}
                      >
                        <Trash2 className="w-4 h-4" />
                      </Button>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      <StationDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        onSubmit={async (data) => {
          if (editingStation) {
            const updateData: UpdateStationRequest = {
              name: data.name,
              status: ("status" in data ? (data as Record<string, unknown>).status : null) as string || editingStation.status,
              maxPower: data.maxPower,
              totalConnectors: data.totalConnectors,
              location: data.location,
              floor: data.floor,
            };
            await updateMutation.mutateAsync({
              id: editingStation.id,
              data: updateData,
            });
          } else {
            await createMutation.mutateAsync(data);
          }
          setDialogOpen(false);
        }}
        editingStation={editingStation}
        loading={isSubmitting}
      />

      <Dialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>确认删除</DialogTitle>
          </DialogHeader>
          <div className="py-4">
            <p className="text-sm text-gray-600">
              确定要删除充电桩{" "}
              <strong>&quot;{deletingStation?.name}&quot;</strong> 吗？此操作无法撤销。
            </p>
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setDeleteDialogOpen(false)}
              disabled={deleteMutation.isPending}
            >
              取消
            </Button>
            <Button
              variant="destructive"
              onClick={handleConfirmDelete}
              disabled={deleteMutation.isPending}
            >
              {deleteMutation.isPending ? "删除中..." : "确认删除"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

interface StationDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSubmit: (data: CreateStationRequest) => void;
  editingStation?: Station | null;
  loading?: boolean;
}

const stationTypeOptions = [
  { value: "ac", label: "交流慢充" },
  { value: "dc", label: "直流快充" },
  { value: "fast_dc", label: "超充" },
];

const connectorTypeOptions = [
  { value: "ac", label: "交流" },
  { value: "dc", label: "直流" },
  { value: "fast_dc", label: "超充" },
];

const statusOptions = [
  { value: "available", label: "可用" },
  { value: "offline", label: "离线" },
  { value: "maintenance", label: "维护中" },
];

function StationDialog({
  open,
  onOpenChange,
  onSubmit,
  editingStation,
  loading = false,
}: StationDialogProps) {
  const [formData, setFormData] = useState<CreateStationRequest & { status?: string }>({
    lotId: "",
    name: "",
    stationType: "dc",
    connectorType: "dc",
    maxPower: 60,
    voltage: 380,
    totalConnectors: 2,
    location: "",
    floor: "",
  });
  const [errors, setErrors] = useState<
    Partial<Record<keyof CreateStationRequest, string>>
  >({});

  const isEditMode = !!editingStation;

  const handleOpen = () => {
    if (editingStation) {
      setFormData({
        lotId: editingStation.lotId,
        name: editingStation.name,
        stationType: editingStation.stationType,
        connectorType: editingStation.connectorType,
        maxPower: editingStation.maxPower,
        voltage: editingStation.voltage,
        totalConnectors: editingStation.totalConnectors,
        location: editingStation.location,
        floor: editingStation.floor,
        status: editingStation.status,
      });
    } else {
      setFormData({
        lotId: "",
        name: "",
        stationType: "dc",
        connectorType: "dc",
        maxPower: 60,
        voltage: 380,
        totalConnectors: 2,
        location: "",
        floor: "",
      });
    }
    setErrors({});
  };

  const validateForm = (): boolean => {
    const newErrors: Partial<Record<keyof CreateStationRequest, string>> = {};
    if (!formData.name.trim()) newErrors.name = "请输入充电桩名称";
    if (!formData.lotId.trim() && !isEditMode)
      newErrors.lotId = "请输入停车场ID";
    if (formData.maxPower <= 0) newErrors.maxPower = "最大功率必须大于0";
    if (formData.totalConnectors <= 0)
      newErrors.totalConnectors = "连接器数量必须大于0";
    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (validateForm()) {
      onSubmit(formData);
    }
  };

  const handleInputChange = (
    field: string,
    value: string | number
  ) => {
    setFormData((prev) => ({ ...prev, [field]: value }));
    if (errors[field as keyof CreateStationRequest]) {
      setErrors((prev) => ({ ...prev, [field]: undefined }));
    }
  };

  return (
    <Dialog
      open={open}
      onOpenChange={(v) => {
        if (v) handleOpen();
        onOpenChange(v);
      }}
    >
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>
            {isEditMode ? "编辑充电桩" : "新增充电桩"}
          </DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          {!isEditMode && (
            <div className="space-y-2">
              <label className="text-sm font-medium">
                停车场ID <span className="text-red-500">*</span>
              </label>
              <Input
                placeholder="请输入停车场ID"
                value={formData.lotId}
                onChange={(e) => handleInputChange("lotId", e.target.value)}
                className={errors.lotId ? "border-red-500" : ""}
              />
              {errors.lotId && (
                <p className="text-sm text-red-500">{errors.lotId}</p>
              )}
            </div>
          )}

          <div className="space-y-2">
            <label className="text-sm font-medium">
              充电桩名称 <span className="text-red-500">*</span>
            </label>
            <Input
              placeholder="请输入充电桩名称"
              value={formData.name}
              onChange={(e) => handleInputChange("name", e.target.value)}
              className={errors.name ? "border-red-500" : ""}
            />
            {errors.name && (
              <p className="text-sm text-red-500">{errors.name}</p>
            )}
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <label className="text-sm font-medium">充电桩类型</label>
              <Select
                value={formData.stationType}
                onValueChange={(v) => v && handleInputChange("stationType", v)}
              >
                <SelectTrigger className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {stationTypeOptions.map((opt) => (
                    <SelectItem key={opt.value} value={opt.value}>
                      {opt.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium">连接器类型</label>
              <Select
                value={formData.connectorType}
                onValueChange={(v) =>
                  v && handleInputChange("connectorType", v)
                }
              >
                <SelectTrigger className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {connectorTypeOptions.map((opt) => (
                    <SelectItem key={opt.value} value={opt.value}>
                      {opt.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>

          <div className="grid grid-cols-3 gap-4">
            <div className="space-y-2">
              <label className="text-sm font-medium">
                最大功率(kW) <span className="text-red-500">*</span>
              </label>
              <Input
                type="number"
                min="1"
                value={formData.maxPower}
                onChange={(e) =>
                  handleInputChange("maxPower", parseFloat(e.target.value) || 0)
                }
                className={errors.maxPower ? "border-red-500" : ""}
              />
              {errors.maxPower && (
                <p className="text-sm text-red-500">{errors.maxPower}</p>
              )}
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium">电压(V)</label>
              <Input
                type="number"
                min="0"
                value={formData.voltage}
                onChange={(e) =>
                  handleInputChange("voltage", parseFloat(e.target.value) || 0)
                }
              />
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium">
                连接器数量 <span className="text-red-500">*</span>
              </label>
              <Input
                type="number"
                min="1"
                value={formData.totalConnectors}
                onChange={(e) =>
                  handleInputChange(
                    "totalConnectors",
                    parseInt(e.target.value) || 1
                  )
                }
                className={errors.totalConnectors ? "border-red-500" : ""}
              />
              {errors.totalConnectors && (
                <p className="text-sm text-red-500">{errors.totalConnectors}</p>
              )}
            </div>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <label className="text-sm font-medium">位置</label>
              <Input
                placeholder="如：A区"
                value={formData.location}
                onChange={(e) => handleInputChange("location", e.target.value)}
              />
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium">楼层</label>
              <Input
                placeholder="如：B1"
                value={formData.floor}
                onChange={(e) => handleInputChange("floor", e.target.value)}
              />
            </div>
          </div>

          {isEditMode && (
            <div className="space-y-2">
              <label className="text-sm font-medium">状态</label>
              <Select
                value={formData.status || "available"}
                onValueChange={(v) => v && handleInputChange("status", v)}
              >
                <SelectTrigger className="w-full">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {statusOptions.map((opt) => (
                    <SelectItem key={opt.value} value={opt.value}>
                      {opt.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          )}

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={loading}
            >
              取消
            </Button>
            <Button type="submit" disabled={loading}>
              {loading ? "提交中..." : isEditMode ? "保存" : "新增"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
