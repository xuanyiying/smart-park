"use client";

import { useState, useEffect } from "react";
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
import { useApiMutation } from "@/hooks/use-api";
import { DeviceInfo, deviceService } from "@/services/device";
import { Loader2, Plus, Trash2 } from "lucide-react";

interface ConfigEntry {
  key: string;
  value: string;
}

interface DeviceConfigDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  device: DeviceInfo;
}

export function DeviceConfigDialog({
  open,
  onOpenChange,
  device,
}: DeviceConfigDialogProps) {
  const [entries, setEntries] = useState<ConfigEntry[]>([]);

  useEffect(() => {
    if (open && device.vendorSpecificConfig) {
      const initialEntries = Object.entries(device.vendorSpecificConfig).map(
        ([key, value]) => ({ key, value })
      );
      setEntries(initialEntries.length > 0 ? initialEntries : []);
    } else if (open) {
      setEntries([]);
    }
  }, [open, device]);

  const updateMutation = useApiMutation(
    (data: { deviceId: string; config: Record<string, string> }) =>
      deviceService.updateDeviceConfig(data.deviceId, data.config),
    {
      successMessage: "配置更新成功",
    }
  );

  const handleAddEntry = () => {
    setEntries([...entries, { key: "", value: "" }]);
  };

  const handleRemoveEntry = (index: number) => {
    setEntries(entries.filter((_, i) => i !== index));
  };

  const handleKeyChange = (index: number, newKey: string) => {
    const updated = [...entries];
    updated[index] = { ...updated[index], key: newKey };
    setEntries(updated);
  };

  const handleValueChange = (index: number, newValue: string) => {
    const updated = [...entries];
    updated[index] = { ...updated[index], value: newValue };
    setEntries(updated);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const config: Record<string, string> = {};
    for (const entry of entries) {
      if (entry.key.trim()) {
        config[entry.key.trim()] = entry.value;
      }
    }
    await updateMutation.mutateAsync({ deviceId: device.deviceId, config });
    onOpenChange(false);
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[560px]">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>设备配置</DialogTitle>
            <DialogDescription>
              编辑设备 {device.deviceId} 的厂商特定配置
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-3 py-4">
            {entries.length === 0 ? (
              <div className="text-center py-6 text-gray-500">
                <p className="text-sm">暂无配置项</p>
                <p className="text-xs mt-1">点击下方按钮添加配置项</p>
              </div>
            ) : (
              <div className="space-y-2">
                {entries.map((entry, index) => (
                  <div key={index} className="flex items-center gap-2">
                    <Input
                      placeholder="键名"
                      value={entry.key}
                      onChange={(e) => handleKeyChange(index, e.target.value)}
                      className="flex-1"
                    />
                    <span className="text-gray-400">=</span>
                    <Input
                      placeholder="值"
                      value={entry.value}
                      onChange={(e) => handleValueChange(index, e.target.value)}
                      className="flex-1"
                    />
                    <Button
                      type="button"
                      variant="outline"
                      size="icon-sm"
                      onClick={() => handleRemoveEntry(index)}
                      className="text-red-600 hover:bg-red-50 shrink-0"
                    >
                      <Trash2 className="w-4 h-4" />
                    </Button>
                  </div>
                ))}
              </div>
            )}

            <Button
              type="button"
              variant="outline"
              size="sm"
              onClick={handleAddEntry}
              className="w-full"
            >
              <Plus className="w-4 h-4 mr-2" />
              添加配置项
            </Button>
          </div>

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
              disabled={updateMutation.isPending}
            >
              取消
            </Button>
            <Button type="submit" disabled={updateMutation.isPending}>
              {updateMutation.isPending && <Loader2 className="w-4 h-4 mr-2 animate-spin" />}
              保存配置
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
