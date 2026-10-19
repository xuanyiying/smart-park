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
import { blacklistService, type BlacklistEntry } from "@/services/blacklist";
import { Loader2, Plus, RefreshCw, Search, ShieldBan, Trash2, WifiOff } from "lucide-react";
import { useState } from "react";

export default function BlacklistPage() {
  const [searchQuery, setSearchQuery] = useState("");
  const [addDialogOpen, setAddDialogOpen] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<BlacklistEntry | null>(null);
  const [formData, setFormData] = useState({ plateNumber: "", reason: "" });
  const [formError, setFormError] = useState("");

  const {
    data: response,
    isLoading,
    error,
    refetch,
  } = useApiQuery(["blacklist"], () => blacklistService.list(1, 200), {
    refetchOnWindowFocus: false,
  });

  const entries: BlacklistEntry[] = response?.list || [];
  const filteredEntries = entries.filter((entry) =>
    entry.plateNumber.toLowerCase().includes(searchQuery.toLowerCase()),
  );

  const addMutation = useApiMutation(
    (data: { plateNumber: string; reason: string }) => blacklistService.add(data),
    {
      successMessage: "已加入黑名单",
      invalidateKeys: [["blacklist"]],
    },
  );

  const removeMutation = useApiMutation(
    (plateNumber: string) => blacklistService.remove(plateNumber),
    {
      successMessage: "已移出黑名单",
      invalidateKeys: [["blacklist"]],
    },
  );

  const handleAdd = async (e: React.FormEvent) => {
    e.preventDefault();
    const plateNumber = formData.plateNumber.trim().toUpperCase();
    if (!plateNumber) {
      setFormError("请输入车牌号");
      return;
    }
    if (!formData.reason.trim()) {
      setFormError("请输入拉黑原因");
      return;
    }
    try {
      await addMutation.mutateAsync({ plateNumber, reason: formData.reason.trim() });
      setAddDialogOpen(false);
      setFormData({ plateNumber: "", reason: "" });
      setFormError("");
    } catch {
      // error handled by useApiMutation
    }
  };

  const handleConfirmDelete = async () => {
    if (deleteTarget) {
      await removeMutation.mutateAsync(deleteTarget.plateNumber);
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
          <p className="text-gray-500 mb-4">获取黑名单数据时出现问题，请稍后重试</p>
          <Button variant="outline" onClick={() => refetch()}>
            <RefreshCw className="w-4 h-4 mr-2" />
            重新加载
          </Button>
        </CardContent>
      </Card>
    );
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold text-gray-900">黑名单管理</h1>
          <p className="text-gray-500 mt-1">禁止指定车辆入场</p>
        </div>
        <Button onClick={() => setAddDialogOpen(true)}>
          <Plus className="w-4 h-4 mr-2" />
          加入黑名单
        </Button>
      </div>

      <Card>
        <CardHeader className="pb-3">
          <div className="flex items-center justify-between">
            <div className="text-lg font-medium">
              黑名单列表
              <span className="text-sm text-gray-400 ml-2">
                共 {response?.total ?? 0} 条
              </span>
            </div>
            <div className="relative w-64">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
              <Input
                placeholder="搜索车牌号"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="pl-9"
              />
            </div>
          </div>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="flex items-center justify-center py-16">
              <Loader2 className="w-8 h-8 animate-spin text-gray-400" />
            </div>
          ) : error ? (
            renderErrorState()
          ) : filteredEntries.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-16 text-center">
              <div className="w-16 h-16 rounded-full bg-gray-100 flex items-center justify-center mb-4">
                <ShieldBan className="w-8 h-8 text-gray-400" />
              </div>
              <h3 className="text-lg font-medium text-gray-900 mb-1">黑名单为空</h3>
              <p className="text-gray-500">当前没有车辆被加入黑名单</p>
            </div>
          ) : (
            <div className="border rounded-lg overflow-hidden">
              <table className="w-full">
                <thead className="bg-gray-50 border-b">
                  <tr>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">车牌号</th>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">原因</th>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">操作人</th>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">状态</th>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">加入时间</th>
                    <th className="text-right px-4 py-3 text-sm font-medium text-gray-500">操作</th>
                  </tr>
                </thead>
                <tbody>
                  {filteredEntries.map((entry) => (
                    <tr key={entry.plateNumber} className="border-b last:border-0 hover:bg-gray-50">
                      <td className="px-4 py-3 font-mono font-medium">{entry.plateNumber}</td>
                      <td className="px-4 py-3 text-gray-600">{entry.reason || "-"}</td>
                      <td className="px-4 py-3 text-gray-600">{entry.createdBy || "-"}</td>
                      <td className="px-4 py-3">
                        <Badge variant={entry.active ? "destructive" : "secondary"}>
                          {entry.active ? "生效中" : "已解除"}
                        </Badge>
                      </td>
                      <td className="px-4 py-3 text-gray-500 text-sm">
                        {entry.createdAt
                          ? new Date(entry.createdAt).toLocaleString("zh-CN")
                          : "-"}
                      </td>
                      <td className="px-4 py-3 text-right">
                        {entry.active && (
                          <Button
                            variant="ghost"
                            size="sm"
                            className="text-red-500"
                            onClick={() => setDeleteTarget(entry)}
                          >
                            <Trash2 className="w-4 h-4" />
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

      <Dialog open={addDialogOpen} onOpenChange={setAddDialogOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>加入黑名单</DialogTitle>
          </DialogHeader>
          <form onSubmit={handleAdd} className="space-y-4">
            <div className="space-y-2">
              <label className="text-sm font-medium">
                车牌号 <span className="text-red-500">*</span>
              </label>
              <Input
                placeholder="如：京A12345"
                value={formData.plateNumber}
                onChange={(e) => setFormData({ ...formData, plateNumber: e.target.value })}
              />
            </div>
            <div className="space-y-2">
              <label className="text-sm font-medium">
                拉黑原因 <span className="text-red-500">*</span>
              </label>
              <Input
                placeholder="请输入拉黑原因"
                value={formData.reason}
                onChange={(e) => setFormData({ ...formData, reason: e.target.value })}
              />
            </div>
            {formError && <p className="text-sm text-red-500">{formError}</p>}
            <DialogFooter>
              <Button type="button" variant="outline" onClick={() => setAddDialogOpen(false)}>
                取消
              </Button>
              <Button type="submit" disabled={addMutation.isPending}>
                {addMutation.isPending ? "提交中..." : "确认"}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>

      <Dialog open={!!deleteTarget} onOpenChange={(open) => !open && setDeleteTarget(null)}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>确认移出黑名单</DialogTitle>
          </DialogHeader>
          <div className="py-4">
            <p className="text-sm text-gray-600">
              确定将车牌 <strong>{deleteTarget?.plateNumber}</strong> 移出黑名单吗？该车辆此后可正常入场。
            </p>
          </div>
          <DialogFooter>
            <Button variant="outline" onClick={() => setDeleteTarget(null)}>
              取消
            </Button>
            <Button
              variant="destructive"
              onClick={handleConfirmDelete}
              disabled={removeMutation.isPending}
            >
              {removeMutation.isPending ? "处理中..." : "确认移出"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
