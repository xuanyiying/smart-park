"use client";

import { Button } from "@/components/ui/button";
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
import {
  DEVICE_STATUS_OPTIONS,
  DEVICE_TYPE_OPTIONS,
  DEVICE_TYPES,
} from "@/constants/device";
import { useApiQuery } from "@/hooks/use-api";
import { adminService } from "@/services/admin";
import { DeviceInfo } from "@/services/device";
import { Loader2 } from "lucide-react";
import { useEffect, useState } from "react";

interface DeviceDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  device?: DeviceInfo | null;
  onSubmit: (data: {
    deviceId: string;
    deviceType: string;
    lotId: string;
    laneId: string;
    status: string;
  }) => void;
  isSubmitting?: boolean;
}

export function DeviceDialog({
  open,
  onOpenChange,
  device,
  onSubmit,
  isSubmitting,
}: DeviceDialogProps) {
  const [formData, setFormData] = useState<{
    deviceId: string;
    deviceType: string;
    lotId: string;
    laneId: string;
    status: string;
  }>({
    deviceId: "",
    deviceType: DEVICE_TYPES.CAMERA,
    lotId: "",
    laneId: "",
    status: "active",
  });

  const { data: lotsResponse, isLoading: isLoadingLots } = useApiQuery(
    ["parking-lots"],
    () => adminService.listParkingLots(1, 100),
    {
      refetchOnWindowFocus: false,
    }
  );

  const lots = lotsResponse?.list || [];

  useEffect(() => {
    if (device) {
      setFormData({
        deviceId: device.deviceId,
        deviceType: device.deviceType,
        lotId: device.lotId,
        laneId: device.laneId,
        status: device.status,
      });
    } else {
      setFormData({
        deviceId: "",
        deviceType: DEVICE_TYPES.CAMERA,
        lotId: "",
        laneId: "",
        status: "active",
      });
    }
  }, [device, open]);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onSubmit(formData);
  };

  const handleValueChange = (value: string | null, field: string) => {
    if (value !== null) {
      setFormData({ ...formData, [field]: value });
    }
  };

  const isEditing = !!device;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[500px]">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>{isEditing ? "编辑设备" : "新增设备"}</DialogTitle>
            <DialogDescription>
              {isEditing ? "修改设备信息" : "添加新设备到系统"}
            </DialogDescription>
          </DialogHeader>

          <div className="grid gap-4 py-4">
            <div className="grid gap-2">
              <Label htmlFor="deviceId">设备ID</Label>
              <Input
                id="deviceId"
                value={formData.deviceId}
                onChange={(e) =>
                  setFormData({ ...formData, deviceId: e.target.value })
                }
                placeholder="请输入设备ID"
                disabled={isEditing}
                required
              />
            </div>

            <div className="grid gap-2">
              <Label htmlFor="deviceType">设备类型</Label>
              <Select
                value={formData.deviceType}
                onValueChange={(value) => handleValueChange(value, "deviceType")}
              >
                <SelectTrigger>
                  <SelectValue placeholder="选择设备类型" />
                </SelectTrigger>
                <SelectContent>
                  {DEVICE_TYPE_OPTIONS.map((option) => (
                    <SelectItem key={option.value} value={option.value}>
                      {option.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            <div className="grid gap-2">
              <Label htmlFor="lotId">所属停车场</Label>
              <Select
                value={formData.lotId}
                onValueChange={(value) => handleValueChange(value, "lotId")}
              >
                <SelectTrigger>
                  <SelectValue placeholder="选择停车场" />
                </SelectTrigger>
                <SelectContent>
                  {isLoadingLots ? (
                    <SelectItem value="loading" disabled>
                      <Loader2 className="w-4 h-4 animate-spin" />
                      加载中...
                    </SelectItem>
                  ) : (
                    lots.map((lot) => (
                      <SelectItem key={lot.id} value={lot.id}>
                        {lot.name}
                      </SelectItem>
                    ))
                  )}
                </SelectContent>
              </Select>
            </div>

            <div className="grid gap-2">
              <Label htmlFor="laneId">所属车道</Label>
              <Input
                id="laneId"
                value={formData.laneId}
                onChange={(e) =>
                  setFormData({ ...formData, laneId: e.target.value })
                }
                placeholder="请输入车道ID"
              />
            </div>

            <div className="grid gap-2">
              <Label htmlFor="status">设备状态</Label>
              <Select
                value={formData.status}
                onValueChange={(value) => handleValueChange(value, "status")}
              >
                <SelectTrigger>
                  <SelectValue placeholder="选择设备状态" />
                </SelectTrigger>
                <SelectContent>
                  {DEVICE_STATUS_OPTIONS.map((option) => (
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
              onClick={() => onOpenChange(false)}
              disabled={isSubmitting}
            >
              取消
            </Button>
            <Button type="submit" disabled={isSubmitting}>
              {isSubmitting && (
                <Loader2 className="w-4 h-4 mr-2 animate-spin" />
              )}
              {isEditing ? "保存" : "创建"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
