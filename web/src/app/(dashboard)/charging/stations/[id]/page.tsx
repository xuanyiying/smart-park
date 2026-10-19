"use client";

import { useState } from "react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
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
  Station,
  Connector,
  chargingService,
  CreateConnectorRequest,
} from "@/services/charging";
import { ArrowLeft, Loader2, Plus, Trash2 } from "lucide-react";
import Link from "next/link";

const CONNECTOR_TYPE_LABELS: Record<string, string> = {
  ac: "交流",
  dc: "直流",
  fast_dc: "超充",
};

const CONNECTOR_STATUS_LABELS: Record<string, string> = {
  available: "可用",
  charging: "充电中",
  faulted: "故障",
  offline: "离线",
};

const CONNECTOR_STATUS_VARIANT: Record<
  string,
  "default" | "secondary" | "destructive" | "outline"
> = {
  available: "default",
  charging: "outline",
  faulted: "destructive",
  offline: "secondary",
};

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

export default function ChargingStationDetailPage({
  params,
}: {
  params: { id: string };
}) {
  const [dialogOpen, setDialogOpen] = useState(false);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [deletingConnector, setDeletingConnector] = useState<Connector | null>(
    null
  );

  const {
    data: station,
    isLoading: stationLoading,
    error: stationError,
    refetch: refetchStation,
  } = useApiQuery<Station>(
    ["charging-station", params.id],
    () => chargingService.getStation(params.id),
    { refetchOnWindowFocus: false }
  );

  const {
    data: connectorsResponse,
    isLoading: connectorsLoading,
    refetch: refetchConnectors,
  } = useApiQuery<Connector[]>(
    ["charging-connectors", params.id],
    () => chargingService.listConnectors(params.id),
    {
      refetchInterval: 30000,
      refetchOnWindowFocus: false,
    }
  );

  const connectors: Connector[] = connectorsResponse || [];

  const createConnectorMutation = useApiMutation(
    (data: CreateConnectorRequest) => chargingService.createConnector(data),
    {
      successMessage: "创建充电枪成功",
      invalidateKeys: [
        ["charging-connectors", params.id],
        ["charging-stations"],
      ],
    }
  );

  const handleAddConnector = () => {
    setDialogOpen(true);
  };

  const handleDeleteClick = (connector: Connector) => {
    setDeletingConnector(connector);
    setDeleteDialogOpen(true);
  };

  const handleConfirmDelete = async () => {
    if (deletingConnector) {
      setDeleteDialogOpen(false);
      setDeletingConnector(null);
    }
  };

  if (stationLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="w-8 h-8 animate-spin text-gray-400" />
      </div>
    );
  }

  if (stationError || !station) {
    return (
      <div className="flex flex-col items-center justify-center py-16 text-center">
        <h3 className="text-lg font-medium text-gray-900 mb-1">加载失败</h3>
        <p className="text-gray-500 mb-4">获取充电桩信息时出现问题</p>
        <Button variant="outline" onClick={() => refetchStation()}>
          重新加载
        </Button>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-4">
        <Link href="/charging/stations">
          <Button variant="ghost" size="sm">
            <ArrowLeft className="w-4 h-4 mr-1" />
            返回
          </Button>
        </Link>
        <div>
          <h1 className="text-2xl font-semibold text-gray-900">
            {station.name}
          </h1>
          <p className="text-gray-500 mt-1">充电桩详情和充电枪管理</p>
        </div>
      </div>

      <Card>
        <CardContent className="pt-6">
          <h2 className="text-lg font-medium mb-4">基本信息</h2>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
            <div>
              <p className="text-sm text-gray-500">类型</p>
              <Badge variant="outline">
                {STATION_TYPE_LABELS[station.stationType] ||
                  station.stationType}
              </Badge>
            </div>
            <div>
              <p className="text-sm text-gray-500">状态</p>
              <Badge variant={station.status === "available" ? "default" : "secondary"}>
                {STATION_STATUS_LABELS[station.status] || station.status}
              </Badge>
            </div>
            <div>
              <p className="text-sm text-gray-500">最大功率</p>
              <p className="font-medium">{station.maxPower}kW</p>
            </div>
            <div>
              <p className="text-sm text-gray-500">电压</p>
              <p className="font-medium">{station.voltage}V</p>
            </div>
            <div>
              <p className="text-sm text-gray-500">可用枪数</p>
              <p className="font-medium">
                {station.availableConnectors}/{station.totalConnectors}
              </p>
            </div>
            <div>
              <p className="text-sm text-gray-500">位置</p>
              <p className="font-medium">{station.location || "-"}</p>
            </div>
            <div>
              <p className="text-sm text-gray-500">楼层</p>
              <p className="font-medium">{station.floor || "-"}</p>
            </div>
            <div>
              <p className="text-sm text-gray-500">创建时间</p>
              <p className="font-medium text-sm">{station.createdAt}</p>
            </div>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardContent className="pt-6">
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-lg font-medium">充电枪列表</h2>
            <Button onClick={handleAddConnector}>
              <Plus className="w-4 h-4 mr-2" />
              新增充电枪
            </Button>
          </div>

          {connectorsLoading ? (
            <div className="flex items-center justify-center py-12">
              <Loader2 className="w-8 h-8 animate-spin text-gray-400" />
            </div>
          ) : connectors.length === 0 ? (
            <div className="text-center py-8 text-gray-500">
              暂无充电枪数据
            </div>
          ) : (
            <div className="border rounded-lg">
              <table className="w-full">
                <thead className="bg-gray-50 border-b">
                  <tr>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">
                      编号
                    </th>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">
                      类型
                    </th>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">
                      状态
                    </th>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">
                      最大功率(kW)
                    </th>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">
                      电压(V)
                    </th>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">
                      电流(A)
                    </th>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">
                      操作
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {connectors.map((connector) => (
                    <tr
                      key={connector.id}
                      className="border-b last:border-0 hover:bg-gray-50"
                    >
                      <td className="px-4 py-3 font-medium">
                        #{connector.number}
                      </td>
                      <td className="px-4 py-3">
                        {CONNECTOR_TYPE_LABELS[connector.type] ||
                          connector.type}
                      </td>
                      <td className="px-4 py-3">
                        <Badge
                          variant={
                            CONNECTOR_STATUS_VARIANT[connector.status] ||
                            "secondary"
                          }
                        >
                          {CONNECTOR_STATUS_LABELS[connector.status] ||
                            connector.status}
                        </Badge>
                      </td>
                      <td className="px-4 py-3">{connector.maxPower}</td>
                      <td className="px-4 py-3">{connector.voltage}</td>
                      <td className="px-4 py-3">{connector.current}</td>
                      <td className="px-4 py-3">
                        <Button
                          variant="ghost"
                          size="sm"
                          className="text-red-500"
                          onClick={() => handleDeleteClick(connector)}
                        >
                          <Trash2 className="w-4 h-4" />
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

      <ConnectorDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        onSubmit={async (data) => {
          await createConnectorMutation.mutateAsync({
            ...data,
            stationId: params.id,
          });
          setDialogOpen(false);
        }}
        loading={createConnectorMutation.isPending}
      />

      <Dialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>确认删除</DialogTitle>
          </DialogHeader>
          <div className="py-4">
            <p className="text-sm text-gray-600">
              确定要删除充电枪 #{deletingConnector?.number} 吗？此操作无法撤销。
            </p>
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setDeleteDialogOpen(false)}
            >
              取消
            </Button>
            <Button variant="destructive" onClick={handleConfirmDelete}>
              确认删除
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

interface ConnectorDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSubmit: (data: Omit<CreateConnectorRequest, "stationId">) => void;
  loading?: boolean;
}

const connectorTypeOptions = [
  { value: "ac", label: "交流" },
  { value: "dc", label: "直流" },
  { value: "fast_dc", label: "超充" },
];

function ConnectorDialog({
  open,
  onOpenChange,
  onSubmit,
  loading = false,
}: ConnectorDialogProps) {
  const [formData, setFormData] = useState({
    number: 1,
    type: "dc",
    maxPower: 60,
    voltage: 380,
  });
  const [errors, setErrors] = useState<
    Partial<Record<string, string>>
  >({});

  const handleOpen = () => {
    setFormData({ number: 1, type: "dc", maxPower: 60, voltage: 380 });
    setErrors({});
  };

  const validateForm = (): boolean => {
    const newErrors: Partial<Record<string, string>> = {};
    if (formData.number <= 0) newErrors.number = "编号必须大于0";
    if (formData.maxPower <= 0) newErrors.maxPower = "最大功率必须大于0";
    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (validateForm()) {
      onSubmit(formData);
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
          <DialogTitle>新增充电枪</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <label className="text-sm font-medium">
                编号 <span className="text-red-500">*</span>
              </label>
              <Input
                type="number"
                min="1"
                value={formData.number}
                onChange={(e) =>
                  setFormData((prev) => ({
                    ...prev,
                    number: parseInt(e.target.value) || 1,
                  }))
                }
                className={errors.number ? "border-red-500" : ""}
              />
              {errors.number && (
                <p className="text-sm text-red-500">{errors.number}</p>
              )}
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium">类型</label>
              <Select
                value={formData.type}
                onValueChange={(v) =>
                  v && setFormData((prev) => ({ ...prev, type: v }))
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

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <label className="text-sm font-medium">
                最大功率(kW) <span className="text-red-500">*</span>
              </label>
              <Input
                type="number"
                min="1"
                value={formData.maxPower}
                onChange={(e) =>
                  setFormData((prev) => ({
                    ...prev,
                    maxPower: parseFloat(e.target.value) || 0,
                  }))
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
                  setFormData((prev) => ({
                    ...prev,
                    voltage: parseFloat(e.target.value) || 0,
                  }))
                }
              />
            </div>
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
