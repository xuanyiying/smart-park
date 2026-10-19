"use client";

import { useState, useEffect } from "react";
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
import { useApiMutation, useApiQuery } from "@/hooks/use-api";
import { deviceService, Manufacturer, CreateManufacturerData, UpdateManufacturerData } from "@/services/device";
import { AlertTriangle, Edit, Loader2, Plus, Trash2, WifiOff } from "lucide-react";

const defaultFormData: CreateManufacturerData = {
  name: "",
  website: "",
  contactInfo: "",
  description: "",
};

export default function ManufacturersPage() {
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingItem, setEditingItem] = useState<Manufacturer | null>(null);
  const [formData, setFormData] = useState<CreateManufacturerData>(defaultFormData);

  const { data: response, isLoading, error, refetch } = useApiQuery(
    ["manufacturers"],
    () => deviceService.listManufacturers(),
    { staleTime: 30000 }
  );

  const manufacturers = response?.list ?? [];

  const createMutation = useApiMutation(
    (data: CreateManufacturerData) => deviceService.createManufacturer(data),
    {
      successMessage: "厂商创建成功",
      invalidateKeys: [["manufacturers"]],
    }
  );

  const updateMutation = useApiMutation(
    (data: { id: string; data: UpdateManufacturerData }) =>
      deviceService.updateManufacturer(data.id, data.data),
    {
      successMessage: "厂商更新成功",
      invalidateKeys: [["manufacturers"]],
    }
  );

  const deleteMutation = useApiMutation(
    (id: string) => deviceService.deleteManufacturer(id),
    {
      successMessage: "厂商删除成功",
      invalidateKeys: [["manufacturers"]],
    }
  );

  useEffect(() => {
    if (dialogOpen) {
      if (editingItem) {
        setFormData({
          name: editingItem.name,
          website: editingItem.website,
          contactInfo: editingItem.contactInfo,
          description: editingItem.description,
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

  const handleEdit = (item: Manufacturer) => {
    setEditingItem(item);
    setDialogOpen(true);
  };

  const handleDelete = async (id: string) => {
    if (confirm("确定要删除此厂商吗？")) {
      await deleteMutation.mutateAsync(id);
    }
  };

  const handleSubmit = async (e: { preventDefault: () => void }) => {
    e.preventDefault();
    if (editingItem) {
      await updateMutation.mutateAsync({
        id: editingItem.id,
        data: formData,
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
          <h1 className="text-2xl font-semibold text-gray-900">厂商管理</h1>
          <p className="text-gray-500 mt-1">管理设备厂商信息</p>
        </div>
        <Button onClick={handleAdd}>
          <Plus className="w-4 h-4 mr-2" />
          新增厂商
        </Button>
      </div>

      <Card>
        <CardContent className="pt-6">
          {isLoading ? (
            <div className="flex items-center justify-center py-12">
              <Loader2 className="w-8 h-8 animate-spin text-gray-400" />
            </div>
          ) : error ? (
            <div className="flex flex-col items-center justify-center py-12">
              {(() => {
                const err = error as { code?: number; message?: string };
                const isNetworkError =
                  err.code === 0 ||
                  !err.code ||
                  (err.message && (err.message.includes("Network") || err.message.includes("timeout") || err.message.includes("超时")));
                return (
                  <>
                    <div className={`w-16 h-16 rounded-full flex items-center justify-center mb-4 ${isNetworkError ? "bg-orange-50" : "bg-red-50"}`}>
                      {isNetworkError ? (
                        <WifiOff className="w-8 h-8 text-orange-500" />
                      ) : (
                        <AlertTriangle className="w-8 h-8 text-red-500" />
                      )}
                    </div>
                    <h3 className={`text-lg font-semibold mb-1 ${isNetworkError ? "text-orange-700" : "text-red-700"}`}>
                      {isNetworkError ? "网络连接异常" : "加载厂商列表失败"}
                    </h3>
                    <p className="text-gray-500 text-sm mb-4">
                      {isNetworkError
                        ? "请检查网络连接是否正常，或稍后重试"
                        : err.message || "请求失败，请稍后重试"}
                    </p>
                    <Button variant="outline" onClick={() => refetch()} className="gap-2">
                      <Loader2 className="w-4 h-4" />
                      重试
                    </Button>
                  </>
                );
              })()}
            </div>
          ) : (
            <div className="border rounded-lg">
              <table className="w-full">
                <thead className="bg-gray-50 border-b">
                  <tr>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">名称</th>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">官网</th>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">联系方式</th>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">描述</th>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">操作</th>
                  </tr>
                </thead>
                <tbody>
                  {manufacturers.length === 0 ? (
                    <tr>
                      <td colSpan={5} className="px-4 py-8 text-center text-gray-500">
                        暂无厂商数据
                      </td>
                    </tr>
                  ) : (
                    manufacturers.map((item: Manufacturer) => (
                      <tr key={item.id} className="border-b last:border-0 hover:bg-gray-50">
                        <td className="px-4 py-3 font-medium">{item.name}</td>
                        <td className="px-4 py-3">
                          {item.website ? (
                            <a
                              href={item.website}
                              target="_blank"
                              rel="noopener noreferrer"
                              className="text-blue-600 hover:underline"
                            >
                              {item.website}
                            </a>
                          ) : (
                            <span className="text-gray-400">-</span>
                          )}
                        </td>
                        <td className="px-4 py-3 text-gray-600">{item.contactInfo || "-"}</td>
                        <td className="px-4 py-3 text-gray-600 max-w-[200px] truncate">{item.description || "-"}</td>
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
        <DialogContent className="sm:max-w-[500px]">
          <form onSubmit={handleSubmit}>
            <DialogHeader>
              <DialogTitle>{editingItem ? "编辑厂商" : "新增厂商"}</DialogTitle>
              <DialogDescription>
                {editingItem ? "修改厂商信息" : "添加新厂商到系统"}
              </DialogDescription>
            </DialogHeader>

            <div className="grid gap-4 py-4">
              <div className="grid gap-2">
                <Label htmlFor="name">名称</Label>
                <Input
                  id="name"
                  value={formData.name}
                  onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                  placeholder="请输入厂商名称"
                  required
                />
              </div>

              <div className="grid gap-2">
                <Label htmlFor="website">官网</Label>
                <Input
                  id="website"
                  value={formData.website}
                  onChange={(e) => setFormData({ ...formData, website: e.target.value })}
                  placeholder="请输入官网地址"
                />
              </div>

              <div className="grid gap-2">
                <Label htmlFor="contactInfo">联系方式</Label>
                <Input
                  id="contactInfo"
                  value={formData.contactInfo}
                  onChange={(e) => setFormData({ ...formData, contactInfo: e.target.value })}
                  placeholder="请输入联系方式"
                />
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
