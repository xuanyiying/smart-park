"use client";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { FIRMWARE_STATUS_LABELS } from "@/constants/device";
import { useApiMutation, useApiQuery } from "@/hooks/use-api";
import { CreateFirmwareData, deviceService, Firmware, UpdateFirmwareData } from "@/services/device";
import { AlertCircle, Edit, Loader2, Plus, Trash2, WifiOff } from "lucide-react";
import { useEffect, useState } from "react";

const firmwareStatusVariant: Record<string, "default" | "secondary" | "destructive" | "outline"> = {
  draft: "outline",
  published: "default",
  deprecated: "secondary",
};

const FIRMWARE_STATUS_OPTIONS = [
  { value: "draft", label: "草稿" },
  { value: "published", label: "已发布" },
  { value: "deprecated", label: "已废弃" },
];

const defaultFormData: CreateFirmwareData = {
  firmwareId: "",
  manufacturer: "",
  model: "",
  version: "",
  url: "",
  size: 0,
  md5: "",
  description: "",
  status: "draft",
};

function formatBytes(bytes: number): string {
  if (bytes === 0) return "0 KB";
  const kb = bytes / 1024;
  if (kb < 1024) return `${kb.toFixed(1)} KB`;
  const mb = kb / 1024;
  return `${mb.toFixed(1)} MB`;
}

export default function FirmwaresPage() {
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingItem, setEditingItem] = useState<Firmware | null>(null);
  const [formData, setFormData] = useState<CreateFirmwareData>(defaultFormData);
  const [filterManufacturer, setFilterManufacturer] = useState("");
  const [filterModel, setFilterModel] = useState("");

  const { data: response, isLoading, error, refetch } = useApiQuery(
    ["firmwares", filterManufacturer, filterModel],
    () => deviceService.listFirmwares(filterManufacturer || undefined, filterModel || undefined),
    { staleTime: 30000 }
  );

  const firmwares = response?.list ?? [];

  const createMutation = useApiMutation(
    (data: CreateFirmwareData) => deviceService.createFirmware(data),
    {
      successMessage: "固件创建成功",
      invalidateKeys: [["firmwares"]],
    }
  );

  const updateMutation = useApiMutation(
    (data: { id: string; data: UpdateFirmwareData }) =>
      deviceService.updateFirmware(data.id, data.data),
    {
      successMessage: "固件更新成功",
      invalidateKeys: [["firmwares"]],
    }
  );

  const deleteMutation = useApiMutation(
    (id: string) => deviceService.deleteFirmware(id),
    {
      successMessage: "固件删除成功",
      invalidateKeys: [["firmwares"]],
    }
  );

  useEffect(() => {
    if (dialogOpen) {
      if (editingItem) {
        setFormData({
          firmwareId: editingItem.firmwareId,
          manufacturer: editingItem.manufacturer,
          model: editingItem.model,
          version: editingItem.version,
          url: editingItem.url,
          size: editingItem.size,
          md5: editingItem.md5,
          description: editingItem.description,
          status: editingItem.status,
        });
      } else {
        setFormData(defaultFormData);
      }
    }
  }, [dialogOpen, editingItem]);

  const handleAdd = () => {
    setEditingItem(null);
    setDialogOpen(true);
  };

  const handleEdit = (item: Firmware) => {
    setEditingItem(item);
    setDialogOpen(true);
  };

  const handleDelete = async (id: string) => {
    if (confirm("确定要删除此固件吗？")) {
      await deleteMutation.mutateAsync(id);
    }
  };

  const handleSubmit = async (e: { preventDefault: () => void }) => {
    e.preventDefault();
    if (editingItem) {
      const { firmwareId, ...updateData } = formData;
      await updateMutation.mutateAsync({
        id: editingItem.id,
        data: updateData,
      });
    } else {
      await createMutation.mutateAsync(formData);
    }
    setDialogOpen(false);
  };

  const isSubmitting = createMutation.isPending || updateMutation.isPending;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold text-gray-900">固件管理</h1>
          <p className="text-gray-500 mt-1">管理设备固件版本与发布</p>
        </div>
        <Button onClick={handleAdd}>
          <Plus className="w-4 h-4 mr-2" />
          新增固件
        </Button>
      </div>

      <Card>
        <CardHeader className="pb-3">
          <div className="flex items-center gap-4">
            <div className="flex items-center gap-2">
              <label className="text-sm font-medium text-gray-600 whitespace-nowrap">厂商</label>
              <Input
                placeholder="按厂商筛选"
                className="w-[180px]"
                value={filterManufacturer}
                onChange={(e) => setFilterManufacturer(e.target.value)}
              />
            </div>
            <div className="flex items-center gap-2">
              <label className="text-sm font-medium text-gray-600 whitespace-nowrap">型号</label>
              <Input
                placeholder="按型号筛选"
                className="w-[180px]"
                value={filterModel}
                onChange={(e) => setFilterModel(e.target.value)}
              />
            </div>
          </div>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="flex items-center justify-center py-12">
              <Loader2 className="w-8 h-8 animate-spin text-gray-400" />
            </div>
          ) : error ? (
            <div className="text-center py-12">
              <div className="mx-auto w-12 h-12 rounded-full bg-red-50 flex items-center justify-center mb-4">
                {error && (error as { message?: string }).message && /network|fetch|timeout|abort/i.test((error as { message?: string }).message || "") ? (
                  <WifiOff className="w-6 h-6 text-red-500" />
                ) : (
                  <AlertCircle className="w-6 h-6 text-red-500" />
                )}
              </div>
              <h3 className="text-base font-medium text-gray-900 mb-1">
                {error && (error as { message?: string }).message && /network|fetch|timeout|abort/i.test((error as { message?: string }).message || "")
                  ? "网络连接异常"
                  : "加载失败"}
              </h3>
              <p className="text-sm text-red-500 mb-4">
                {error && (error as { message?: string }).message
                  ? (error as { message?: string }).message
                  : "加载固件列表失败，请稍后重试"}
              </p>
              <Button variant="outline" onClick={() => refetch()}>
                重试
              </Button>
            </div>
          ) : (
            <div className="border rounded-lg">
              <table className="w-full">
                <thead className="bg-gray-50 border-b">
                  <tr>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">固件ID</th>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">厂商</th>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">型号</th>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">版本</th>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">大小</th>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">MD5</th>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">状态</th>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">发布日期</th>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">操作</th>
                  </tr>
                </thead>
                <tbody>
                  {firmwares.length === 0 ? (
                    <tr>
                      <td colSpan={9} className="px-4 py-8 text-center text-gray-500">
                        暂无固件数据
                      </td>
                    </tr>
                  ) : (
                    firmwares.map((item: Firmware) => (
                      <tr key={item.id} className="border-b last:border-0 hover:bg-gray-50">
                        <td className="px-4 py-3 font-medium">{item.firmwareId}</td>
                        <td className="px-4 py-3 text-gray-600">{item.manufacturer}</td>
                        <td className="px-4 py-3 text-gray-600">{item.model}</td>
                        <td className="px-4 py-3 text-gray-600">{item.version}</td>
                        <td className="px-4 py-3 text-gray-600">{formatBytes(item.size)}</td>
                        <td className="px-4 py-3 text-gray-600 font-mono text-xs">{item.md5}</td>
                        <td className="px-4 py-3">
                          <Badge variant={firmwareStatusVariant[item.status] || "outline"}>
                            {FIRMWARE_STATUS_LABELS[item.status] || item.status}
                          </Badge>
                        </td>
                        <td className="px-4 py-3 text-gray-600">{item.releaseDate ? new Date(item.releaseDate).toLocaleDateString("zh-CN") : "-"}</td>
                        <td className="px-4 py-3">
                          <div className="flex items-center gap-2">
                            <Button
                              variant="outline"
                              size="sm"
                              onClick={() => handleEdit(item)}
                            >
                              <Edit className="w-4 h-4" />
                            </Button>
                            <Button
                              variant="outline"
                              size="sm"
                              onClick={() => handleDelete(item.id)}
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

      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className="sm:max-w-[560px]">
          <form onSubmit={handleSubmit}>
            <DialogHeader>
              <DialogTitle>{editingItem ? "编辑固件" : "新增固件"}</DialogTitle>
              <DialogDescription>
                {editingItem ? "修改固件信息" : "添加新固件到系统"}
              </DialogDescription>
            </DialogHeader>

            <div className="grid gap-4 py-4">
              <div className="grid gap-2">
                <Label htmlFor="firmwareId">固件ID</Label>
                <Input
                  id="firmwareId"
                  value={formData.firmwareId}
                  onChange={(e) => setFormData({ ...formData, firmwareId: e.target.value })}
                  placeholder="请输入固件ID"
                  disabled={!!editingItem}
                  required
                />
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div className="grid gap-2">
                  <Label htmlFor="manufacturer">厂商</Label>
                  <Input
                    id="manufacturer"
                    value={formData.manufacturer}
                    onChange={(e) => setFormData({ ...formData, manufacturer: e.target.value })}
                    placeholder="请输入厂商"
                    required
                  />
                </div>
                <div className="grid gap-2">
                  <Label htmlFor="model">型号</Label>
                  <Input
                    id="model"
                    value={formData.model}
                    onChange={(e) => setFormData({ ...formData, model: e.target.value })}
                    placeholder="请输入型号"
                    required
                  />
                </div>
              </div>

              <div className="grid gap-2">
                <Label htmlFor="version">版本</Label>
                <Input
                  id="version"
                  value={formData.version}
                  onChange={(e) => setFormData({ ...formData, version: e.target.value })}
                  placeholder="请输入版本号"
                  required
                />
              </div>

              <div className="grid gap-2">
                <Label htmlFor="url">下载地址</Label>
                <Input
                  id="url"
                  value={formData.url}
                  onChange={(e) => setFormData({ ...formData, url: e.target.value })}
                  placeholder="请输入固件下载地址"
                  required
                />
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div className="grid gap-2">
                  <Label htmlFor="size">大小 (bytes)</Label>
                  <Input
                    id="size"
                    type="number"
                    min={0}
                    value={formData.size}
                    onChange={(e) => setFormData({ ...formData, size: parseInt(e.target.value) || 0 })}
                    placeholder="请输入固件大小"
                    required
                  />
                </div>
                <div className="grid gap-2">
                  <Label htmlFor="md5">MD5</Label>
                  <Input
                    id="md5"
                    value={formData.md5}
                    onChange={(e) => setFormData({ ...formData, md5: e.target.value })}
                    placeholder="请输入MD5校验值"
                    required
                  />
                </div>
              </div>

              <div className="grid gap-2">
                <Label htmlFor="description">描述</Label>
                <Input
                  id="description"
                  value={formData.description}
                  onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                  placeholder="请输入描述"
                />
              </div>

              <div className="grid gap-2">
                <Label htmlFor="status">状态</Label>
                <Select
                  value={formData.status}
                  onValueChange={(value) => setFormData({ ...formData, status: value ?? formData.status ?? "draft" })}
                >
                  <SelectTrigger>
                    <SelectValue placeholder="选择状态" />
                  </SelectTrigger>
                  <SelectContent>
                    {FIRMWARE_STATUS_OPTIONS.map((option) => (
                      <SelectItem key={option.value} value={option.value}>
                        {option.label}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            </div>

            <DialogFooter>
              <Button
                type="button"
                variant="outline"
                onClick={() => setDialogOpen(false)}
                disabled={isSubmitting}
              >
                取消
              </Button>
              <Button type="submit" disabled={isSubmitting}>
                {isSubmitting && <Loader2 className="w-4 h-4 mr-2 animate-spin" />}
                {editingItem ? "保存" : "创建"}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  );
}
