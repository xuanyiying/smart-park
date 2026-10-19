"use client";

import { useState, useEffect } from "react";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { ParkingLot } from "@/services/admin";

interface ParkingLotFormData {
  name: string;
  address: string;
  lanes: number;
  totalCapacity: number;
  status: string;
}

interface ParkingLotDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  parkingLot?: ParkingLot | null;
  onSubmit: (data: ParkingLotFormData) => void;
  isLoading?: boolean;
}

const statusOptions = [
  { value: "active", label: "运营中" },
  { value: "inactive", label: "已停用" },
];

const initialFormData: ParkingLotFormData = {
  name: "",
  address: "",
  lanes: 1,
  totalCapacity: 0,
  status: "active",
};

export function ParkingLotDialog({
  open,
  onOpenChange,
  parkingLot,
  onSubmit,
  isLoading = false,
}: ParkingLotDialogProps) {
  const [formData, setFormData] = useState<ParkingLotFormData>(initialFormData);
  const [errors, setErrors] = useState<Partial<Record<keyof ParkingLotFormData, string>>>({});

  const isEditMode = !!parkingLot;

  useEffect(() => {
    if (open) {
      if (parkingLot) {
        setFormData({
          name: parkingLot.name,
          address: parkingLot.address,
          lanes: parkingLot.lanes,
          totalCapacity: parkingLot.totalCapacity ?? 0,
          status: parkingLot.status,
        });
      } else {
        setFormData(initialFormData);
      }
      setErrors({});
    }
  }, [open, parkingLot]);

  const validateForm = (): boolean => {
    const newErrors: Partial<Record<keyof ParkingLotFormData, string>> = {};

    if (!formData.name.trim()) {
      newErrors.name = "请输入停车场名称";
    }

    if (!formData.address.trim()) {
      newErrors.address = "请输入停车场地址";
    }

    if (formData.lanes < 1) {
      newErrors.lanes = "车道数必须大于0";
    }

    if (formData.totalCapacity < 0) {
      newErrors.totalCapacity = "总车位数不能为负数";
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

  const handleInputChange = (field: keyof ParkingLotFormData, value: string | number) => {
    setFormData((prev) => ({ ...prev, [field]: value }));
    if (errors[field]) {
      setErrors((prev) => ({ ...prev, [field]: undefined }));
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{isEditMode ? "编辑停车场" : "新增停车场"}</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <label className="text-sm font-medium">
              停车场名称 <span className="text-red-500">*</span>
            </label>
            <Input
              placeholder="请输入停车场名称"
              value={formData.name}
              onChange={(e) => handleInputChange("name", e.target.value)}
              className={errors.name ? "border-red-500" : ""}
            />
            {errors.name && (
              <p className="text-sm text-red-500">{errors.name}</p>
            )}
          </div>

          <div className="space-y-2">
            <label className="text-sm font-medium">
              停车场地址 <span className="text-red-500">*</span>
            </label>
            <Input
              placeholder="请输入停车场地址"
              value={formData.address}
              onChange={(e) => handleInputChange("address", e.target.value)}
              className={errors.address ? "border-red-500" : ""}
            />
            {errors.address && (
              <p className="text-sm text-red-500">{errors.address}</p>
            )}
          </div>

          <div className="space-y-2">
            <label className="text-sm font-medium">
              车道数 <span className="text-red-500">*</span>
            </label>
            <Input
              type="number"
              min="1"
              placeholder="请输入车道数"
              value={formData.lanes}
              onChange={(e) => handleInputChange("lanes", parseInt(e.target.value) || 1)}
              className={errors.lanes ? "border-red-500" : ""}
            />
            {errors.lanes && (
              <p className="text-sm text-red-500">{errors.lanes}</p>
            )}
          </div>

          <div className="space-y-2">
            <label className="text-sm font-medium">总车位数</label>
            <Input
              type="number"
              min="0"
              placeholder="请输入总车位数"
              value={formData.totalCapacity}
              onChange={(e) =>
                handleInputChange("totalCapacity", parseInt(e.target.value) || 0)
              }
              className={errors.totalCapacity ? "border-red-500" : ""}
            />
            {errors.totalCapacity && (
              <p className="text-sm text-red-500">{errors.totalCapacity}</p>
            )}
          </div>

          {isEditMode && (
            <div className="space-y-2">
              <label className="text-sm font-medium">状态</label>
              <Select
                value={formData.status}
                onValueChange={(value) => value && handleInputChange("status", value)}
              >
                <SelectTrigger className="w-full">
                  <SelectValue placeholder="请选择状态" />
                </SelectTrigger>
                <SelectContent>
                  {statusOptions.map((option) => (
                    <SelectItem key={option.value} value={option.value}>
                      {option.label}
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
