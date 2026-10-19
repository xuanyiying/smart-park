"use client";

import { Button } from "@/components/ui/button";
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
import {
  ACTION_FIELD_CONFIG,
  CONDITION_FIELD_CONFIG,
  RULE_PRIORITY_OPTIONS,
  RULE_TYPE_CONFIG,
} from "@/constants/billing";
import { BILLING_VEHICLE_TYPE_OPTIONS } from "@/constants/vehicle";
import { BillingRule, CreateBillingRuleRequest } from "@/services/billing";
import { useEffect, useState } from "react";

interface BillingRuleDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onSubmit: (data: CreateBillingRuleRequest) => void;
  editingRule?: BillingRule | null;
  loading?: boolean;
}

export function BillingRuleDialog({
  open,
  onOpenChange,
  onSubmit,
  editingRule,
  loading,
}: BillingRuleDialogProps) {
  const [formData, setFormData] = useState<CreateBillingRuleRequest>({
    lotId: "default",
    ruleName: "",
    ruleType: "base",
    conditionsJson: "{}",
    actionsJson: "{}",
    priority: 2,
    isActive: true,
  });

  // 可视化表单状态
  const [visualConditions, setVisualConditions] = useState<Record<string, any>>({});
  const [visualActions, setVisualActions] = useState<Record<string, any>>({});

  useEffect(() => {
    if (editingRule) {
      setFormData({
        lotId: editingRule.lotId,
        ruleName: editingRule.ruleName,
        ruleType: editingRule.ruleType,
        conditionsJson: editingRule.conditionsJson,
        actionsJson: editingRule.actionsJson,
        priority: editingRule.priority,
        isActive: editingRule.isActive,
      });
      // 解析 JSON 到可视化表单
      try {
        setVisualConditions(JSON.parse(editingRule.conditionsJson || "{}"));
        setVisualActions(JSON.parse(editingRule.actionsJson || "{}"));
      } catch {
        setVisualConditions({});
        setVisualActions({});
      }
    } else {
      const defaultType = "base";
      setFormData({
        lotId: "default",
        ruleName: "",
        ruleType: defaultType,
        conditionsJson: "{}",
        actionsJson: "{}",
        priority: 2,
        isActive: true,
      });
      setVisualConditions({});
      setVisualActions({});
    }
  }, [editingRule, open]);

  // 当可视化表单变化时，更新 JSON
  useEffect(() => {
    setFormData(prev => ({
      ...prev,
      conditionsJson: JSON.stringify(visualConditions),
      actionsJson: JSON.stringify(visualActions),
    }));
  }, [visualConditions, visualActions]);

  const handleRuleTypeChange = (value: string | null) => {
    if (!value) return;
    setFormData(prev => ({
      ...prev,
      ruleType: value,
    }));
    // 清空当前配置，让用户重新填写
    setVisualConditions({});
    setVisualActions({});
  };

  const handleConditionChange = (field: string, value: any) => {
    setVisualConditions(prev => ({
      ...prev,
      [field]: value,
    }));
  };

  const handleActionChange = (field: string, value: any) => {
    setVisualActions(prev => ({
      ...prev,
      [field]: value,
    }));
  };

  const handleSubmit = () => {
    if (!formData.ruleName.trim()) {
      return;
    }
    onSubmit(formData);
  };

  const currentConfig = RULE_TYPE_CONFIG[formData.ruleType];

  const renderConditionField = (field: string) => {
    const config = CONDITION_FIELD_CONFIG[field];
    if (!config) return null;

    const value = visualConditions[field];

    if (config.type === "select") {
      const selectedLabel = BILLING_VEHICLE_TYPE_OPTIONS.find(t => t.value === value)?.label;
      return (
        <div key={field} className="space-y-2">
          <label className="text-sm font-medium">{config.label}</label>
          <Select
            value={value || ""}
            onValueChange={(v) => handleConditionChange(field, v)}
          >
            <SelectTrigger className="w-full">
              <SelectValue>{selectedLabel || config.placeholder}</SelectValue>
            </SelectTrigger>
            <SelectContent>
              {BILLING_VEHICLE_TYPE_OPTIONS.map((type) => (
                <SelectItem key={type.value} value={type.value}>
                  {type.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      );
    }

    return (
      <div key={field} className="space-y-2">
        <label className="text-sm font-medium">{config.label}</label>
        <Input
          type={config.type}
          value={value || ""}
          onChange={(e) => {
            const val = config.type === "number"
              ? (e.target.value === "" ? "" : Number(e.target.value))
              : e.target.value;
            handleConditionChange(field, val);
          }}
          placeholder={config.placeholder}
          min={config.min}
          max={config.max}
        />
      </div>
    );
  };

  const renderActionField = (field: string) => {
    const config = ACTION_FIELD_CONFIG[field];
    if (!config) return null;

    const value = visualActions[field];

    return (
      <div key={field} className="space-y-2">
        <label className="text-sm font-medium">{config.label}</label>
        <div className="flex items-center gap-2">
          <Input
            type={config.type}
            value={value || ""}
            onChange={(e) => {
              const val = config.type === "number"
                ? (e.target.value === "" ? "" : Number(e.target.value))
                : e.target.value;
              handleActionChange(field, val);
            }}
            placeholder={config.placeholder}
            min={config.min}
            max={config.max}
            step={(config as any).step}
            className="flex-1"
          />
          {config.suffix && (
            <span className="text-sm text-gray-500 whitespace-nowrap">{config.suffix}</span>
          )}
        </div>
      </div>
    );
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>
            {editingRule ? "编辑计费规则" : "新增计费规则"}
          </DialogTitle>
        </DialogHeader>

        <div className="space-y-6 py-4">
          {/* 基本信息 */}
          <div className="space-y-4">
            <h3 className="text-sm font-semibold text-gray-900 border-b pb-2">基本信息</h3>

            <div className="space-y-2">
              <label className="text-sm font-medium">规则名称</label>
              <Input
                value={formData.ruleName}
                onChange={(e) =>
                  setFormData((prev) => ({ ...prev, ruleName: e.target.value }))
                }
                placeholder="例如: 工作日白天收费"
              />
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium">规则类型</label>
              <Select
                value={formData.ruleType}
                onValueChange={handleRuleTypeChange}
              >
                <SelectTrigger className="w-full">
                  <SelectValue>
                    <div className="flex flex-col">
                      <span>{RULE_TYPE_CONFIG[formData.ruleType]?.label}</span>
                    </div>
                  </SelectValue>
                </SelectTrigger>
                <SelectContent>
                  {Object.entries(RULE_TYPE_CONFIG).map(([key, config]) => (
                    <SelectItem key={key} value={key}>
                      <div className="flex flex-col">
                        <span>{config.label}</span>
                        <span className="text-xs text-gray-400">{config.description}</span>
                      </div>
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium">优先级</label>
              <Select
                value={String(formData.priority)}
                onValueChange={(value) =>
                  setFormData((prev) => ({ ...prev, priority: Number(value) }))
                }
              >
                <SelectTrigger className="w-full">
                  <SelectValue>
                    {RULE_PRIORITY_OPTIONS.find(p => p.value === formData.priority)?.label}
                  </SelectValue>
                </SelectTrigger>
                <SelectContent>
                  {RULE_PRIORITY_OPTIONS.map((p) => (
                    <SelectItem key={p.value} value={String(p.value)}>
                      {p.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>

          {/* 条件配置 */}
          <div className="space-y-4">
            <h3 className="text-sm font-semibold text-gray-900 border-b pb-2">
              适用条件
              <span className="text-xs font-normal text-gray-400 ml-2">满足以下条件时规则生效</span>
            </h3>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {currentConfig?.conditionFields.map(renderConditionField)}
            </div>
          </div>

          {/* 计费动作 */}
          <div className="space-y-4">
            <h3 className="text-sm font-semibold text-gray-900 border-b pb-2">
              计费设置
              <span className="text-xs font-normal text-gray-400 ml-2">满足条件时执行的动作</span>
            </h3>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              {currentConfig?.actionFields.map(renderActionField)}
            </div>
          </div>

          {/* 启用状态 */}
          <div className="flex items-center gap-2 pt-2">
            <input
              type="checkbox"
              id="is_active"
              checked={formData.isActive}
              onChange={(e) =>
                setFormData((prev) => ({ ...prev, isActive: e.target.checked }))
              }
              className="w-4 h-4 rounded border-gray-300"
            />
            <label htmlFor="is_active" className="text-sm font-medium">
              启用规则
            </label>
          </div>

          {/* JSON 预览（可选，用于调试） */}
          <div className="space-y-2 pt-4 border-t">
            <label className="text-xs text-gray-400">配置预览 (JSON)</label>
            <div className="bg-gray-50 rounded-lg p-3 text-xs font-mono text-gray-600 space-y-1">
              <div>条件: {formData.conditionsJson}</div>
              <div>动作: {formData.actionsJson}</div>
            </div>
          </div>
        </div>

        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => onOpenChange(false)}
            disabled={loading}
          >
            取消
          </Button>
          <Button onClick={handleSubmit} disabled={loading}>
            {loading ? "保存中..." : "保存"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
