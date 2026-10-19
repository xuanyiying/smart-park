"use client";

import { BillingRuleDialog } from "@/components/modules/billing-rule-dialog";
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
import { RULE_PRIORITY_LABELS, RULE_TYPE_LABELS } from "@/constants/billing";
import { useApiMutation, useApiQuery } from "@/hooks/use-api";
import {
  BillingRule,
  billingService,
  CreateBillingRuleRequest,
} from "@/services/billing";
import { FileText, Loader2, Pencil, Plus, RefreshCw, Search, Trash2, WifiOff } from "lucide-react";
import { useState } from "react";

export default function BillingPage() {
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingRule, setEditingRule] = useState<BillingRule | null>(null);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [deletingRule, setDeletingRule] = useState<BillingRule | null>(null);

  const {
    data: response,
    isLoading,
    error,
    refetch,
  } = useApiQuery<BillingRule[]>(
    ["billing-rules"],
    () => billingService.getBillingRules("default"),
    {
      refetchOnWindowFocus: false,
    }
  );

  const rules: BillingRule[] = response || [];

  const createMutation = useApiMutation(
    (data: CreateBillingRuleRequest) => billingService.createBillingRule(data),
    {
      successMessage: "创建规则成功",
      invalidateKeys: [["billing-rules"]],
    }
  );

  const updateMutation = useApiMutation(
    ({ id, data }: { id: string; data: Omit<CreateBillingRuleRequest, "lotId"> }) =>
      billingService.updateBillingRule(id, data),
    {
      successMessage: "更新规则成功",
      invalidateKeys: [["billing-rules"]],
    }
  );

  const deleteMutation = useApiMutation(
    (id: string) => billingService.deleteBillingRule(id),
    {
      successMessage: "删除规则成功",
      invalidateKeys: [["billing-rules"]],
    }
  );

  const handleAddRule = () => {
    setEditingRule(null);
    setDialogOpen(true);
  };

  const handleEditRule = (rule: BillingRule) => {
    setEditingRule(rule);
    setDialogOpen(true);
  };

  const handleDeleteClick = (rule: BillingRule) => {
    setDeletingRule(rule);
    setDeleteDialogOpen(true);
  };

  const handleConfirmDelete = async () => {
    if (deletingRule) {
      await deleteMutation.mutateAsync(deletingRule.id);
      setDeleteDialogOpen(false);
      setDeletingRule(null);
    }
  };

  const handleSubmit = async (data: CreateBillingRuleRequest) => {
    if (editingRule) {
      await updateMutation.mutateAsync({
        id: editingRule.id,
        data: {
          ruleName: data.ruleName,
          ruleType: data.ruleType,
          conditionsJson: data.conditionsJson,
          actionsJson: data.actionsJson,
          priority: data.priority,
          isActive: data.isActive,
        },
      });
    } else {
      await createMutation.mutateAsync(data);
    }
    setDialogOpen(false);
  };

  const isSubmitting =
    createMutation.isPending || updateMutation.isPending;

  // 渲染空状态
  const renderEmptyState = () => (
    <div className="flex flex-col items-center justify-center py-16 text-center">
      <div className="w-16 h-16 rounded-full bg-gray-100 flex items-center justify-center mb-4">
        <FileText className="w-8 h-8 text-gray-400" />
      </div>
      <h3 className="text-lg font-medium text-gray-900 mb-1">暂无计费规则</h3>
      <p className="text-gray-500 mb-4">您还没有创建任何计费规则，点击下方按钮添加第一个规则</p>
      <Button onClick={handleAddRule}>
        <Plus className="w-4 h-4 mr-2" />
        新增规则
      </Button>
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
              : "获取计费规则时出现问题，请稍后重试"}
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
          <h1 className="text-2xl font-semibold text-gray-900">计费规则管理</h1>
          <p className="text-gray-500 mt-1">配置停车费用计算规则</p>
        </div>
        <Button onClick={handleAddRule}>
          <Plus className="w-4 h-4 mr-2" />
          新增规则
        </Button>
      </div>

      <Card>
        <CardHeader className="pb-3">
          <div className="text-lg font-medium">计费规则列表</div>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className="flex items-center justify-center py-16">
              <Loader2 className="w-8 h-8 animate-spin text-gray-400" />
            </div>
          ) : error ? (
            renderErrorState()
          ) : rules.length === 0 ? (
            renderEmptyState()
          ) : (
            <div className="space-y-4">
              {rules.map((rule) => (
                <div
                  key={rule.id}
                  className="border rounded-lg p-4 hover:bg-gray-50"
                >
                  <div className="flex items-start justify-between">
                    <div className="space-y-2">
                      <div className="flex items-center gap-3">
                        <h3 className="font-medium">{rule.ruleName}</h3>
                        <Badge variant={rule.isActive ? "default" : "secondary"}>
                          {rule.isActive ? "启用" : "停用"}
                        </Badge>
                        <Badge variant="outline">
                          {RULE_TYPE_LABELS[rule.ruleType] || rule.ruleType}
                        </Badge>
                      </div>
                      <div className="text-sm text-gray-500 space-x-4">
                        <span>优先级: {RULE_PRIORITY_LABELS[rule.priority] || rule.priority}</span>
                        <span>条件: {rule.conditionsJson}</span>
                        <span>动作: {rule.actionsJson}</span>
                      </div>
                      <div className="text-xs text-gray-400">
                        创建时间: {rule.createdAt}
                      </div>
                    </div>
                    <div className="flex gap-2">
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => handleEditRule(rule)}
                      >
                        <Pencil className="w-4 h-4" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        className="text-red-500"
                        onClick={() => handleDeleteClick(rule)}
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

      <BillingRuleDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        onSubmit={handleSubmit}
        editingRule={editingRule}
        loading={isSubmitting}
      />

      <Dialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>确认删除</DialogTitle>
          </DialogHeader>
          <div className="py-4">
            <p className="text-sm text-gray-600">
              确定要删除规则 <strong>&quot;{deletingRule?.ruleName}&quot;</strong> 吗？此操作无法撤销。
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
