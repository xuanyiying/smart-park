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
import { Vehicle } from "@/services/admin";
import { useEffect, useState } from "react";

interface VehicleFormData {
  plateNumber: string;
  vehicleType: string;
  ownerName: string;
  ownerPhone: string;
  monthlyValidUntil: string;
}

interface VehicleDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  vehicle?: Vehicle | null;
  onSubmit: (data: VehicleFormData) => void;
  isLoading?: boolean;
}

const vehicleTypes = [
  { value: "temporary", label: "临时车" },
  { value: "monthly", label: "月租车" },
  { value: "vip", label: "VIP" },
];

const initialFormData: VehicleFormData = {
  plateNumber: "",
  vehicleType: "temporary",
  ownerName: "",
  ownerPhone: "",
  monthlyValidUntil: "",
};

export function VehicleDialog({
  open,
  onOpenChange,
  vehicle,
  onSubmit,
  isLoading = false,
}: VehicleDialogProps) {
  const [formData, setFormData] = useState<VehicleFormData>(initialFormData);
  const [errors, setErrors] = useState<Partial<Record<keyof VehicleFormData, string>>>({});

  const isEditMode = !!vehicle;

  useEffect(() => {
    if (open) {
      if (vehicle) {
        setFormData({
          plateNumber: vehicle.plateNumber,
          vehicleType: vehicle.vehicleType,
          ownerName: vehicle.ownerName,
          ownerPhone: vehicle.ownerPhone,
          monthlyValidUntil: vehicle.monthlyValidUntil || "",
        });
      } else {
        setFormData(initialFormData);
      }
      setErrors({});
    }
  }, [open, vehicle]);

  const validateForm = (): boolean => {
    const newErrors: Partial<Record<keyof VehicleFormData, string>> = {};

    if (!formData.plateNumber.trim()) {
      newErrors.plateNumber = "请输入车牌号";
    } else if (!/^[京津沪渝冀豫云辽黑湘皖鲁新苏浙赣鄂桂甘晋蒙陕吉闽贵粤青藏川宁琼使领][A-Z][A-Z0-9]{5,6}$/.test(formData.plateNumber.trim())) {
      newErrors.plateNumber = "请输入正确的车牌号格式";
    }

    if (!formData.ownerName.trim()) {
      newErrors.ownerName = "请输入车主姓名";
    }

    if (!formData.ownerPhone.trim()) {
      newErrors.ownerPhone = "请输入联系方式";
    } else if (!/^1[3-9]\d{9}$/.test(formData.ownerPhone.trim())) {
      newErrors.ownerPhone = "请输入正确的手机号码";
    }

    if (formData.vehicleType === "monthly" && !formData.monthlyValidUntil) {
      newErrors.monthlyValidUntil = "月租车需填写有效期";
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (validateForm()) {
      onSubmit(formData);
    }
  };

  const handleInputChange = (field: keyof VehicleFormData, value: string) => {
    setFormData((prev) => ({ ...prev, [field]: value }));
    if (errors[field]) {
      setErrors((prev) => ({ ...prev, [field]: undefined }));
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{isEditMode ? "编辑车辆" : "新增车辆"}</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <label className="text-sm font-medium">
              车牌号 <span className="text-red-500">*</span>
            </label>
            <Input
              placeholder="请输入车牌号，如：京A12345"
              value={formData.plateNumber}
              onChange={(e) => handleInputChange("plateNumber", e.target.value)}
              className={errors.plateNumber ? "border-red-500" : ""}
            />
            {errors.plateNumber && (
              <p className="text-sm text-red-500">{errors.plateNumber}</p>
            )}
          </div>

          <div className="space-y-2">
            <label className="text-sm font-medium">
              车辆类型 <span className="text-red-500">*</span>
            </label>
            <Select
              value={formData.vehicleType}
              onValueChange={(value) => value && handleInputChange("vehicleType", value)}
            >
              <SelectTrigger className="w-full">
                <SelectValue placeholder="请选择车辆类型" />
              </SelectTrigger>
              <SelectContent>
                {vehicleTypes.map((type) => (
                  <SelectItem key={type.value} value={type.value}>
                    {type.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-2">
            <label className="text-sm font-medium">
              车主姓名 <span className="text-red-500">*</span>
            </label>
            <Input
              placeholder="请输入车主姓名"
              value={formData.ownerName}
              onChange={(e) => handleInputChange("ownerName", e.target.value)}
              className={errors.ownerName ? "border-red-500" : ""}
            />
            {errors.ownerName && (
              <p className="text-sm text-red-500">{errors.ownerName}</p>
            )}
          </div>

          <div className="space-y-2">
            <label className="text-sm font-medium">
              联系方式 <span className="text-red-500">*</span>
            </label>
            <Input
              placeholder="请输入手机号码"
              value={formData.ownerPhone}
              onChange={(e) => handleInputChange("ownerPhone", e.target.value)}
              className={errors.ownerPhone ? "border-red-500" : ""}
            />
            {errors.ownerPhone && (
              <p className="text-sm text-red-500">{errors.ownerPhone}</p>
            )}
          </div>

          <div className="space-y-2">
            <label className="text-sm font-medium">
              月租有效期
              {formData.vehicleType === "monthly" && (
                <span className="text-red-500"> *</span>
              )}
            </label>
            <Input
              type="date"
              value={formData.monthlyValidUntil}
              onChange={(e) => handleInputChange("monthlyValidUntil", e.target.value)}
              className={errors.monthlyValidUntil ? "border-red-500" : ""}
            />
            {errors.monthlyValidUntil && (
              <p className="text-sm text-red-500">{errors.monthlyValidUntil}</p>
            )}
          </div>

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={isLoading}
            >
              取消
            </Button>
            <Button type="submit" disabled={isLoading}>
              {isLoading ? "提交中..." : isEditMode ? "保存" : "新增"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
