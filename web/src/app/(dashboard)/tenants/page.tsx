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
import { useApiMutation, useApiQuery } from "@/hooks/use-api";
import { tenantService, type Tenant, type TenantConfig } from "@/services/tenant";
import {
  Building2,
  Loader2,
  Pencil,
  Plus,
  Power,
  RefreshCw,
  Trash2,
  WifiOff,
} from "lucide-react";
import { useState } from "react";

interface TenantFormData {
  name: string;
  code: string;
  maxParkingLots: number;
  maxDevices: number;
  maxUsers: number;
}

const initialFormData: TenantFormData = {
  name: "",
  code: "",
  maxParkingLots: 5,
  maxDevices: 50,
  maxUsers: 20,
};

function toConfigData(data: TenantFormData): { name: string; config: Partial<TenantConfig> } {
  return {
    name: data.name,
    config: {
      maxParkingLots: data.maxParkingLots,
      maxDevices: data.maxDevices,
      maxUsers: data.maxUsers,
    },
  };
}

export default function TenantsPage() {
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingTenant, setEditingTenant] = useState<Tenant | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<Tenant | null>(null);
  const [formData, setFormData] = useState<TenantFormData>(initialFormData);
  const [formError, setFormError] = useState("");

  const {
    data: tenants,
    isLoading,
    error,
    refetch,
  } = useApiQuery<Tenant[]>(["tenants"], () => tenantService.list(1, 100), {
    refetchOnWindowFocus: false,
  });

  const createMutation = useApiMutation(
    (data: { name: string; code: string; config: Partial<TenantConfig> }) =>
      tenantService.create(data),
    {
      successMessage: "租户创建成功",
      invalidateKeys: [["tenants"]],
    },
  );

  const updateMutation = useApiMutation(
    ({ id, data }: { id: string; data: { name: string; config: Partial<TenantConfig> } }) =>
      tenantService.update(id, data),
    {
      successMessage: "租户更新成功",
      invalidateKeys: [["tenants"]],
    },
  );

  const enableMutation = useApiMutation(
    (id: string) => tenantService.enable(id),
    {
      successMessage: "租户已启用",
      invalidateKeys: [["tenants"]],
    },
  );

  const disableMutation = useApiMutation(
    (id: string) => tenantService.disable(id),
    {
      successMessage: "租户已停用",
      invalidateKeys: [["tenants"]],
    },
  );

  const deleteMutation = useApiMutation(
    (id: string) => tenantService.remove(id),
    {
      successMessage: "租户已删除",
      invalidateKeys: [["tenants"]],
    },
  );

  const handleAdd = () => {
    setEditingTenant(null);
    setFormData(initialFormData);
    setFormError("");
    setDialogOpen(true);
  };

  const handleEdit = (tenant: Tenant) => {
    setEditingTenant(tenant);
    setFormData({
      name: tenant.name,
      code: tenant.code,
      maxParkingLots: tenant.config?.maxParkingLots ?? 5,
      maxDevices: tenant.config?.maxDevices ?? 50,
      maxUsers: tenant.config?.maxUsers ?? 20,
    });
    setFormError("");
    setDialogOpen(true);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!formData.name.trim()) {
      setFormError("请输入租户名称");
      return;
    }
    if (!editingTenant && !formData.code.trim()) {
      setFormError("请输入租户代码");
      return;
    }

    try {
      if (editingTenant) {
        await updateMutation.mutateAsync({
          id: editingTenant.id,
          data: toConfigData(formData),
        });
      } else {
        await createMutation.mutateAsync({
          name: formData.name.trim(),
          code: formData.code.trim(),
          config: toConfigData(formData).config,
        });
      }
      setDialogOpen(false);
    } catch {
      // error handled by useApiMutation
    }
  };

  const handleToggleStatus = async (tenant: Tenant) => {
    if (tenant.status === "active") {
      await disableMutation.mutateAsync(tenant.id);
    } else {
      await enableMutation.mutateAsync(tenant.id);
    }
  };

  const handleConfirmDelete = async () => {
    if (deleteTarget) {
      await deleteMutation.mutateAsync(deleteTarget.id);
      setDeleteTarget(null);
    }
  };

  const renderErrorState = () => {
    const errorMessage = (error as { message?: string })?.message || "";
    const isNetworkError =
      errorMessage.includes("Network Error") || errorMessage.includes("ECONNREFUSED");

    return (
      <Card>
        <CardContent className="py-16 text-center">
          <div className="w-16 h-16 rounded-full bg-red-50 flex items-center justify-center mx-auto mb-4">
            <WifiOff className="w-8 h-8 text-red-400" />
          </div>
          <h3 className="text-lg font-medium text-gray-900 mb-1">
            {isNetworkError ? "网络连接失败" : "加载失败"}
          </h3>
          <p className="text-gray-500 mb-4">获取租户数据时出现问题，请稍后重试</p>
          <Button variant="outline" onClick={() => refetch()}>
            <RefreshCw className="w-4 h-4 mr-2" />
            重新加载
          </Button>
        </CardContent>
      </Card>
    );
  };

  const statusLabel = (status: string) =>
    ({ active: "启用", disabled: "停用", expired: "已过期" })[status] || status;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold text-gray-900">租户管理</h1>
          <p className="text-gray-500 mt-1">多租户配额与启停控制</p>
        </div>
        <Button onClick={handleAdd}>
          <Plus className="w-4 h-4 mr-2" />
          新增租户
        </Button>
      </div>

      <Card>
        <CardHeader className="pb-3">
          <div className="text-lg font-medium">租户列表</div>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="flex items-center justify-center py-16">
              <Loader2 className="w-8 h-8 animate-spin text-gray-400" />
            </div>
          ) : error ? (
            renderErrorState()
          ) : !tenants || tenants.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-16 text-center">
              <div className="w-16 h-16 rounded-full bg-gray-100 flex items-center justify-center mb-4">
                <Building2 className="w-8 h-8 text-gray-400" />
              </div>
              <h3 className="text-lg font-medium text-gray-900 mb-1">暂无租户</h3>
              <p className="text-gray-500">点击右上角按钮创建第一个租户</p>
            </div>
          ) : (
            <div className="border rounded-lg overflow-hidden">
              <table className="w-full">
                <thead className="bg-gray-50 border-b">
                  <tr>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">名称</th>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">代码</th>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">状态</th>
                    <th className="text-right px-4 py-3 text-sm font-medium text-gray-500">停车场</th>
                    <th className="text-right px-4 py-3 text-sm font-medium text-gray-500">设备</th>
                    <th className="text-right px-4 py-3 text-sm font-medium text-gray-500">用户</th>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">创建时间</th>
                    <th className="text-right px-4 py-3 text-sm font-medium text-gray-500">操作</th>
                  </tr>
                </thead>
                <tbody>
                  {tenants.map((tenant) => (
                    <tr key={tenant.id} className="border-b last:border-0 hover:bg-gray-50">
                      <td className="px-4 py-3 font-medium">{tenant.name}</td>
                      <td className="px-4 py-3 font-mono text-sm text-gray-600">{tenant.code}</td>
                      <td className="px-4 py-3">
                        <Badge
                          variant={
                            tenant.status === "active"
                              ? "default"
                              : tenant.status === "expired"
                                ? "outline"
                                : "secondary"
                          }
                        >
                          {statusLabel(tenant.status)}
                        </Badge>
                      </td>
                      <td className="px-4 py-3 text-right">{tenant.config?.maxParkingLots ?? "-"}</td>
                      <td className="px-4 py-3 text-right">{tenant.config?.maxDevices ?? "-"}</td>
                      <td className="px-4 py-3 text-right">{tenant.config?.maxUsers ?? "-"}</td>
                      <td className="px-4 py-3 text-gray-500 text-sm">
                        {tenant.createdAt
                          ? new Date(tenant.createdAt).toLocaleString("zh-CN")
                          : "-"}
                      </td>
                      <td className="px-4 py-3">
                        <div className="flex justify-end gap-1">
                          <Button
                            variant="ghost"
                            size="sm"
                            onClick={() => handleToggleStatus(tenant)}
                            disabled={
                              enableMutation.isPending || disableMutation.isPending
                            }
                            title={tenant.status === "active" ? "停用" : "启用"}
                          >
                            <Power
                              className={
                                tenant.status === "active"
                                  ? "w-4 h-4 text-red-500"
                                  : "w-4 h-4 text-green-600"
                              }
                            />
                          </Button>
                          <Button variant="ghost" size="sm" onClick={() => handleEdit(tenant)}>
                            <Pencil className="w-4 h-4" />
                          </Button>
                          <Button
                            variant="ghost"
                            size="sm"
                            className="text-red-500"
                            onClick={() => setDeleteTarget(tenant)}
                          >
                            <Trash2 className="w-4 h-4" />
                          </Button>
                        </div>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </CardContent>
      </Card>

      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>{editingTenant ? "编辑租户" : "新增租户"}</DialogTitle>
          </DialogHeader>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-2">
              <label className="text-sm font-medium">
                租户名称 <span className="text-red-500">*</span>
              </label>
              <Input
                placeholder="请输入租户名称"
                value={formData.name}
                onChange={(e) => setFormData({ ...formData, name: e.target.value })}
              />
            </div>
            <div className="space-y-2">
              <label className="text-sm font-medium">
                租户代码 {!editingTenant && <span className="text-red-500">*</span>}
              </label>
              <Input
                placeholder="如：tenant-a"
                value={formData.code}
                disabled={!!editingTenant}
                onChange={(e) => setFormData({ ...formData, code: e.target.value })}
              />
            </div>
            <div className="grid grid-cols-3 gap-3">
              <div className="space-y-2">
                <label className="text-sm font-medium">停车场</label>
                <Input
                  type="number"
                  min="0"
                  value={formData.maxParkingLots}
                  onChange={(e) =>
                    setFormData({ ...formData, maxParkingLots: parseInt(e.target.value) || 0 })
                  }
                />
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium">设备</label>
                <Input
                  type="number"
                  min="0"
                  value={formData.maxDevices}
                  onChange={(e) =>
                    setFormData({ ...formData, maxDevices: parseInt(e.target.value) || 0 })
                  }
                />
              </div>
              <div className="space-y-2">
                <label className="text-sm font-medium">用户</label>
                <Input
                  type="number"
                  min="0"
                  value={formData.maxUsers}
                  onChange={(e) =>
                    setFormData({ ...formData, maxUsers: parseInt(e.target.value) || 0 })
                  }
                />
              </div>
            </div>
            {formError && <p className="text-sm text-red-500">{formError}</p>}
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setDialogOpen(false)}>
                取消
              </Button>
              <Button type="submit" disabled={createMutation.isPending || updateMutation.isPending}>
                {createMutation.isPending || updateMutation.isPending
                  ? "提交中..."
                  : editingTenant
                    ? "保存"
                    : "创建"}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      <Dialog open={!!deleteTarget} onOpenChange={(open) => !open && setDeleteTarget(null)}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>确认删除租户</DialogTitle>
          </DialogHeader>
          <div className="py-4">
            <p className="text-sm text-gray-600">
              确定删除租户 <strong>&quot;{deleteTarget?.name}&quot;</strong> 吗？此操作无法撤销。
            </p>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteTarget(null)}>
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
