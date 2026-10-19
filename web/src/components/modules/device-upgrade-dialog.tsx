"use client";

import { useState, useEffect, useCallback, useRef } from "react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { UPGRADE_STATUS_LABELS, UPGRADE_STATUS_VARIANT } from "@/constants/device";
import { useApiMutation, useApiQuery } from "@/hooks/use-api";
import { DeviceInfo, deviceService, UpgradeStatusDetail } from "@/services/device";
import { Loader2 } from "lucide-react";

interface DeviceUpgradeDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  device: DeviceInfo;
}

export function DeviceUpgradeDialog({
  open,
  onOpenChange,
  device,
}: DeviceUpgradeDialogProps) {
  const [selectedVersion, setSelectedVersion] = useState<string>("");
  const [upgradeId, setUpgradeId] = useState<string | null>(null);
  const [upgradeStatus, setUpgradeStatus] = useState<UpgradeStatusDetail | null>(null);
  const pollTimerRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const { data: firmwaresResponse, isLoading: isLoadingFirmwares } = useApiQuery(
    ["firmwares", device.manufacturer, device.model],
    () => deviceService.listFirmwares(device.manufacturer, device.model),
    {
      enabled: open && !!device.manufacturer && !!device.model,
      refetchOnWindowFocus: false,
    }
  );

  const firmwares = firmwaresResponse?.list ?? [];

  const upgradeMutation = useApiMutation(
    (data: { deviceId: string; firmwareVersion: string }) =>
      deviceService.upgradeDevice(data.deviceId, data.firmwareVersion),
    {
      successMessage: "升级指令已发送",
    }
  );

  const statusQuery = useApiQuery(
    ["upgrade-status", upgradeId],
    () => deviceService.getDeviceUpgradeStatus(upgradeId!),
    {
      enabled: !!upgradeId,
      refetchOnWindowFocus: false,
    }
  );

  const stopPolling = useCallback(() => {
    if (pollTimerRef.current) {
      clearInterval(pollTimerRef.current);
      pollTimerRef.current = null;
    }
  }, []);

  useEffect(() => {
    if (upgradeId && open) {
      pollTimerRef.current = setInterval(() => {
        statusQuery.refetch();
      }, 3000);
    }
    return () => stopPolling();
  }, [upgradeId, open, stopPolling, statusQuery]);

  useEffect(() => {
    if (statusQuery.data) {
      setUpgradeStatus(statusQuery.data);
      const status = statusQuery.data.status;
      if (status === "completed" || status === "failed" || status === "cancelled") {
        stopPolling();
      }
    }
  }, [statusQuery.data, stopPolling]);

  useEffect(() => {
    if (!open) {
      setSelectedVersion("");
      setUpgradeId(null);
      setUpgradeStatus(null);
      stopPolling();
    }
  }, [open, stopPolling]);

  const handleSubmit = async () => {
    if (!selectedVersion) return;
    const result = await upgradeMutation.mutateAsync({
      deviceId: device.deviceId,
      firmwareVersion: selectedVersion,
    });
    setUpgradeId(result.upgradeId);
  };

  const isUpgrading = !!upgradeId;
  const isTerminalState =
    upgradeStatus?.status === "completed" ||
    upgradeStatus?.status === "failed" ||
    upgradeStatus?.status === "cancelled";

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[500px]">
        <DialogHeader>
          <DialogTitle>设备升级</DialogTitle>
          <DialogDescription>
            为设备 {device.deviceId} 选择目标固件版本进行升级
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-4">
          {/* Current version info */}
          <div className="bg-gray-50 rounded-lg p-3 space-y-1">
            <div className="text-sm text-gray-500">当前信息</div>
            <div className="text-sm">
              <span className="font-medium">厂商:</span> {device.manufacturer || "-"}
            </div>
            <div className="text-sm">
              <span className="font-medium">型号:</span> {device.model || "-"}
            </div>
            <div className="text-sm">
              <span className="font-medium">当前版本:</span> {device.firmwareVersion || "-"}
            </div>
          </div>

          {!isUpgrading ? (
            <>
              <div className="grid gap-2">
                <Label>目标固件版本</Label>
                {isLoadingFirmwares ? (
                  <div className="flex items-center gap-2 text-sm text-gray-500">
                    <Loader2 className="w-4 h-4 animate-spin" />
                    加载可用固件...
                  </div>
                ) : firmwares.length === 0 ? (
                  <div className="text-sm text-gray-500">暂无可用固件</div>
                ) : (
                  <Select
                    value={selectedVersion}
                    onValueChange={(value) => setSelectedVersion(value ?? "")}
                  >
                    <SelectTrigger>
                      <SelectValue placeholder="请选择目标固件版本" />
                    </SelectTrigger>
                    <SelectContent>
                      {firmwares.map((fw) => (
                        <SelectItem key={fw.id} value={fw.version}>
                          {fw.version}
                          {fw.version === device.firmwareVersion ? " (当前版本)" : ""}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                )}
              </div>
            </>
          ) : (
            <>
              {/* Upgrade progress */}
              <div className="space-y-3">
                <div className="text-sm font-medium">升级状态</div>
                <div className="flex items-center gap-2">
                  <Badge variant={upgradeStatus ? (UPGRADE_STATUS_VARIANT[upgradeStatus.status] || "outline") : "outline"}>
                    {upgradeStatus ? (UPGRADE_STATUS_LABELS[upgradeStatus.status] || upgradeStatus.status) : "等待中"}
                  </Badge>
                  {(statusQuery.isFetching || (!isTerminalState && upgradeId)) && (
                    <Loader2 className="w-4 h-4 animate-spin text-gray-400" />
                  )}
                </div>
                {upgradeStatus && (
                  <div className="bg-gray-50 rounded-lg p-3 space-y-1 text-sm">
                    <div>
                      <span className="text-gray-500">升级版本:</span>{" "}
                      {upgradeStatus.fromVersion} -&gt; {upgradeStatus.toVersion}
                    </div>
                    {upgradeStatus.errorMessage && (
                      <div className="text-red-600">
                        <span className="text-gray-500">错误信息:</span> {upgradeStatus.errorMessage}
                      </div>
                    )}
                    {upgradeStatus.startTime && (
                      <div>
                        <span className="text-gray-500">开始时间:</span> {upgradeStatus.startTime}
                      </div>
                    )}
                    {upgradeStatus.endTime && (
                      <div>
                        <span className="text-gray-500">结束时间:</span> {upgradeStatus.endTime}
                      </div>
                    )}
                  </div>
                )}
              </div>
            </>
          )}
        </div>

        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => onOpenChange(false)}
            disabled={upgradeMutation.isPending}
          >
            {isTerminalState ? "关闭" : "取消"}
          </Button>
          {!isUpgrading && (
            <Button
              onClick={handleSubmit}
              disabled={!selectedVersion || upgradeMutation.isPending || firmwares.length === 0}
            >
              {upgradeMutation.isPending && <Loader2 className="w-4 h-4 mr-2 animate-spin" />}
              开始升级
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
