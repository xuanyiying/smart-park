"use client";

import { useState } from "react";
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
  Price,
  chargingService,
  CreatePriceRequest,
  Station,
} from "@/services/charging";
import { DollarSign, Loader2, Plus, RefreshCw, WifiOff } from "lucide-react";

export default function ChargingPricesPage() {
  const [dialogOpen, setDialogOpen] = useState(false);
  const [selectedStationId, setSelectedStationId] = useState<string>("");

  const {
    data: stationsResponse,
  } = useApiQuery<Station[]>(
    ["charging-stations"],
    () => chargingService.listStations(undefined, 1, 100),
    { refetchOnWindowFocus: false }
  );

  const stations: Station[] = stationsResponse || [];

  const {
    data: pricesResponse,
    isLoading,
    error,
    refetch,
  } = useApiQuery<Price[]>(
    ["charging-prices", selectedStationId],
    () => chargingService.listPrices(selectedStationId),
    {
      enabled: !!selectedStationId,
      refetchOnWindowFocus: false,
    }
  );

  const prices: Price[] = pricesResponse || [];

  const createMutation = useApiMutation(
    (data: CreatePriceRequest) => chargingService.createPrice(data),
    {
      successMessage: "创建价格配置成功",
      invalidateKeys: [["charging-prices", selectedStationId]],
    }
  );

  const handleSubmit = async (data: CreatePriceRequest) => {
    await createMutation.mutateAsync(data);
    setDialogOpen(false);
  };

  const renderEmptyState = () => (
    <div className="flex flex-col items-center justify-center py-16 text-center">
      <div className="w-16 h-16 rounded-full bg-gray-100 flex items-center justify-center mb-4">
        <DollarSign className="w-8 h-8 text-gray-400" />
      </div>
      <h3 className="text-lg font-medium text-gray-900 mb-1">
        {selectedStationId ? "暂无价格配置" : "请选择充电桩"}
      </h3>
      <p className="text-gray-500 mb-4">
        {selectedStationId
          ? "该充电桩还没有配置价格，点击下方按钮添加"
          : "请在上方选择一个充电桩查看价格配置"}
      </p>
      {selectedStationId && (
        <Button onClick={() => setDialogOpen(true)}>
          <Plus className="w-4 h-4 mr-2" />
          新增价格配置
        </Button>
      )}
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
          获取价格配置时出现问题，请稍后重试
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
          <h1 className="text-2xl font-semibold text-gray-900">充电价格配置</h1>
          <p className="text-gray-500 mt-1">配置充电桩分时电价和服务费</p>
        </div>
        {selectedStationId && (
          <Button onClick={() => setDialogOpen(true)}>
            <Plus className="w-4 h-4 mr-2" />
            新增价格配置
          </Button>
        )}
      </div>

      <Card>
        <CardHeader className="pb-3">
          <div className="flex items-center gap-4">
            <label className="text-sm font-medium">选择充电桩</label>
            <Select
              value={selectedStationId}
              onValueChange={(v) => v && setSelectedStationId(v)}
            >
              <SelectTrigger className="w-[240px]">
                <SelectValue placeholder="请选择充电桩" />
              </SelectTrigger>
              <SelectContent>
                {stations.map((station) => (
                  <SelectItem key={station.id} value={station.id}>
                    {station.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </CardHeader>
        <CardContent>
          {!selectedStationId || isLoading ? (
            !selectedStationId ? (
              renderEmptyState()
            ) : (
              <div className="flex items-center justify-center py-16">
                <Loader2 className="w-8 h-8 animate-spin text-gray-400" />
              </div>
            )
          ) : error ? (
            renderErrorState()
          ) : prices.length === 0 ? (
            renderEmptyState()
          ) : (
            <div className="space-y-4">
              {prices.map((price) => (
                <div
                  key={price.id}
                  className="border rounded-lg p-4 hover:bg-gray-50"
                >
                  <div className="flex items-start justify-between">
                    <div className="space-y-2">
                      <div className="flex items-center gap-3">
                        <h3 className="font-medium">{price.name}</h3>
                        {price.isPeakHours ? (
                          <Badge variant="destructive">高峰</Badge>
                        ) : (
                          <Badge variant="secondary">平谷</Badge>
                        )}
                      </div>
                      <div className="text-sm text-gray-500 space-x-4">
                        <span>
                          时段: {price.startHour}:00 - {price.endHour}:00
                        </span>
                        <span>电价: ¥{price.pricePerKwh}/kWh</span>
                        <span>服务费: ¥{price.serviceFee}</span>
                      </div>
                      <div className="text-sm text-gray-500 space-x-4">
                        <span>高峰负载: {price.peakLoad}kW</span>
                        <span>低谷负载: {price.offPeakLoad}kW</span>
                        <span>
                          生效时间: {price.effectiveAt}
                          {price.expiresAt
                            ? ` - ${price.expiresAt}`
                            : " - 永久"}
                        </span>
                      </div>
                    </div>
                  </div>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      <PriceDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        onSubmit={handleSubmit}
        stationId={selectedStationId}
        loading={createMutation.isPending}
      />
    </div>
  );
}

interface PriceDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSubmit: (data: CreatePriceRequest) => void;
  stationId: string;
  loading?: boolean;
}

function PriceDialog({
  open,
  onOpenChange,
  onSubmit,
  stationId,
  loading = false,
}: PriceDialogProps) {
  const [formData, setFormData] = useState<CreatePriceRequest>({
    stationId: "",
    name: "",
    startHour: 0,
    endHour: 24,
    pricePerKwh: 1.2,
    serviceFee: 0.8,
    peakLoad: 1.5,
    offPeakLoad: 0.8,
    isPeakHours: false,
    effectiveAt: new Date().toISOString(),
    expiresAt: "",
  });
  const [errors, setErrors] = useState<
    Partial<Record<keyof CreatePriceRequest, string>>
  >({});

  const handleOpen = () => {
    setFormData({
      stationId,
      name: "",
      startHour: 0,
      endHour: 24,
      pricePerKwh: 1.2,
      serviceFee: 0.8,
      peakLoad: 1.5,
      offPeakLoad: 0.8,
      isPeakHours: false,
      effectiveAt: new Date().toISOString(),
      expiresAt: "",
    });
    setErrors({});
  };

  const validateForm = (): boolean => {
    const newErrors: Partial<Record<keyof CreatePriceRequest, string>> = {};
    if (!formData.name.trim()) newErrors.name = "请输入配置名称";
    if (formData.pricePerKwh <= 0)
      newErrors.pricePerKwh = "电价必须大于0";
    if (formData.startHour < 0 || formData.startHour > 23)
      newErrors.startHour = "开始小时范围0-23";
    if (formData.endHour < 1 || formData.endHour > 24)
      newErrors.endHour = "结束小时范围1-24";
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
    field: keyof CreatePriceRequest,
    value: string | number | boolean
  ) => {
    setFormData((prev) => ({ ...prev, [field]: value }));
    if (errors[field]) {
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
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>新增价格配置</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <label className="text-sm font-medium">
              配置名称 <span className="text-red-500">*</span>
            </label>
            <Input
              placeholder="如：工作日高峰电价"
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
              <label className="text-sm font-medium">开始小时</label>
              <Input
                type="number"
                min="0"
                max="23"
                value={formData.startHour}
                onChange={(e) =>
                  handleInputChange(
                    "startHour",
                    parseInt(e.target.value) || 0
                  )
                }
                className={errors.startHour ? "border-red-500" : ""}
              />
              {errors.startHour && (
                <p className="text-sm text-red-500">{errors.startHour}</p>
              )}
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium">结束小时</label>
              <Input
                type="number"
                min="1"
                max="24"
                value={formData.endHour}
                onChange={(e) =>
                  handleInputChange("endHour", parseInt(e.target.value) || 24)
                }
                className={errors.endHour ? "border-red-500" : ""}
              />
              {errors.endHour && (
                <p className="text-sm text-red-500">{errors.endHour}</p>
              )}
            </div>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <label className="text-sm font-medium">
                电价(元/kWh) <span className="text-red-500">*</span>
              </label>
              <Input
                type="number"
                min="0"
                step="0.01"
                value={formData.pricePerKwh}
                onChange={(e) =>
                  handleInputChange(
                    "pricePerKwh",
                    parseFloat(e.target.value) || 0
                  )
                }
                className={errors.pricePerKwh ? "border-red-500" : ""}
              />
              {errors.pricePerKwh && (
                <p className="text-sm text-red-500">{errors.pricePerKwh}</p>
              )}
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium">服务费(元)</label>
              <Input
                type="number"
                min="0"
                step="0.01"
                value={formData.serviceFee}
                onChange={(e) =>
                  handleInputChange(
                    "serviceFee",
                    parseFloat(e.target.value) || 0
                  )
                }
              />
            </div>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <label className="text-sm font-medium">高峰负载(kW)</label>
              <Input
                type="number"
                min="0"
                step="0.1"
                value={formData.peakLoad}
                onChange={(e) =>
                  handleInputChange(
                    "peakLoad",
                    parseFloat(e.target.value) || 0
                  )
                }
              />
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium">低谷负载(kW)</label>
              <Input
                type="number"
                min="0"
                step="0.1"
                value={formData.offPeakLoad}
                onChange={(e) =>
                  handleInputChange(
                    "offPeakLoad",
                    parseFloat(e.target.value) || 0
                  )
                }
              />
            </div>
          </div>

          <div className="flex items-center gap-2">
            <input
              type="checkbox"
              id="isPeakHours"
              checked={formData.isPeakHours}
              onChange={(e) =>
                handleInputChange("isPeakHours", e.target.checked)
              }
              className="rounded border-gray-300"
            />
            <label htmlFor="isPeakHours" className="text-sm font-medium">
              高峰时段
            </label>
          </div>

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
              {loading ? "提交中..." : "新增"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
