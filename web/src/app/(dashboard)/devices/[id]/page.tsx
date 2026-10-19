"use client";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import {
  DEVICE_STATUS_LABELS,
  DEVICE_TYPE_LABELS,
  FAULT_SEVERITY_LABELS,
  FAULT_SEVERITY_VARIANT,
  FAULT_STATUS_LABELS,
  FAULT_STATUS_VARIANT,
  LOG_LEVEL_LABELS,
  LOG_LEVEL_VARIANT,
} from "@/constants/device";
import { useApiMutation, useApiQuery } from "@/hooks/use-api";
import {
  DeviceFault,
  DeviceInfo,
  DeviceLog,
  DevicePerformance,
  deviceService,
} from "@/services/device";
import { ArrowLeft, CheckCircle, Loader2, Wifi, WifiOff } from "lucide-react";
import Link from "next/link";
import { useParams } from "next/navigation";
import { useState } from "react";
import {
  Area,
  AreaChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";

export default function DeviceDetailPage() {
  const params = useParams();
  const deviceId = params.id as string;
  const [activeTab, setActiveTab] = useState("overview");
  const [faultPage, setFaultPage] = useState(1);
  const [logPage, setLogPage] = useState(1);
  const pageSize = 10;

  // --- Device basic info (always fetched) ---
  const {
    data: device,
    isLoading: deviceLoading,
    error: deviceError,
    refetch: refetchDevice,
  } = useApiQuery<DeviceInfo>(
    ["device", deviceId],
    () => deviceService.getDevice(deviceId),
    { enabled: !!deviceId }
  );

  // --- Lazy tab data ---
  const isOverviewActive = activeTab === "overview";
  const isFaultsActive = activeTab === "faults";
  const isLogsActive = activeTab === "logs";
  const isPerformanceActive = activeTab === "performance";

  // Overview: stats summary
  const { data: statsSummary, isLoading: statsLoading } = useApiQuery<
    Record<string, unknown>
  >(
    ["device-stats-summary", deviceId],
    () => deviceService.getDeviceStatsSummary(deviceId),
    { enabled: isOverviewActive && !!deviceId }
  );

  // Faults
  const {
    data: faultsData,
    isLoading: faultsLoading,
  } = useApiQuery<{ list: DeviceFault[]; total: number }>(
    ["device-faults", deviceId, faultPage],
    () => deviceService.listDeviceFaults(deviceId, undefined, faultPage, pageSize),
    { enabled: isFaultsActive && !!deviceId }
  );

  // Logs
  const {
    data: logsData,
    isLoading: logsLoading,
  } = useApiQuery<{ list: DeviceLog[]; total: number }>(
    ["device-logs", deviceId, logPage],
    () => deviceService.getDeviceLogs(deviceId, logPage, pageSize),
    { enabled: isLogsActive && !!deviceId }
  );

  // Performance
  const {
    data: performanceData,
    isLoading: performanceLoading,
  } = useApiQuery<DevicePerformance[]>(
    ["device-performance", deviceId],
    () => deviceService.getDevicePerformance(deviceId),
    { enabled: isPerformanceActive && !!deviceId }
  );

  // Resolve fault mutation
  const resolveFaultMutation = useApiMutation(
    (id: string) => deviceService.resolveDeviceFault(id),
    {
      successMessage: "故障已标记为解决",
      invalidateKeys: [["device-faults", deviceId, faultPage]],
    }
  );

  // --- Loading & Error states ---
  if (deviceLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Loader2 className="w-8 h-8 animate-spin text-gray-400" />
      </div>
    );
  }

  if (deviceError || !device) {
    return (
      <div className="flex flex-col items-center justify-center py-16 text-center">
        <h3 className="text-lg font-medium text-gray-900 mb-1">加载失败</h3>
        <p className="text-gray-500 mb-4">获取设备信息时出现问题</p>
        <Button variant="outline" onClick={() => refetchDevice()}>
          重新加载
        </Button>
      </div>
    );
  }

  const faults = faultsData?.list ?? [];
  const faultsTotal = faultsData?.total ?? 0;
  const faultsTotalPages = Math.ceil(faultsTotal / pageSize);

  const logs = logsData?.list ?? [];
  const logsTotal = logsData?.total ?? 0;
  const logsTotalPages = Math.ceil(logsTotal / pageSize);

  // Performance chart data - format timestamp for display
  const chartData = (performanceData ?? []).map((p) => ({
    ...p,
    time: new Date(p.timestamp).toLocaleTimeString("zh-CN", {
      hour: "2-digit",
      minute: "2-digit",
    }),
  }));

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center gap-4">
        <Link href="/devices">
          <Button variant="ghost" size="sm">
            <ArrowLeft className="w-4 h-4 mr-1" />
            返回
          </Button>
        </Link>
        <div className="flex-1">
          <h1 className="text-2xl font-semibold text-gray-900">
            设备详情
          </h1>
          <p className="text-gray-500 mt-1">{device.deviceId}</p>
        </div>
      </div>

      {/* Device basic info card */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            基本信息
            <Badge
              variant={
                device.online
                  ? "default"
                  : "secondary"
              }
              className="ml-2"
            >
              {device.online ? (
                <span className="flex items-center gap-1">
                  <Wifi className="w-3 h-3" />
                  在线
                </span>
              ) : (
                <span className="flex items-center gap-1">
                  <WifiOff className="w-3 h-3" />
                  离线
                </span>
              )}
            </Badge>
            <Badge
              variant={
                device.status === "active" || device.status === "normal"
                  ? "default"
                  : device.status === "error" || device.status === "fault"
                    ? "destructive"
                    : "secondary"
              }
            >
              {DEVICE_STATUS_LABELS[device.status] || device.status}
            </Badge>
          </CardTitle>
        </CardHeader>
        <CardContent>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
            <div>
              <p className="text-sm text-gray-500">设备ID</p>
              <p className="font-medium">{device.deviceId}</p>
            </div>
            <div>
              <p className="text-sm text-gray-500">设备类型</p>
              <Badge variant="outline">
                {DEVICE_TYPE_LABELS[device.deviceType] || device.deviceType}
              </Badge>
            </div>
            <div>
              <p className="text-sm text-gray-500">厂商</p>
              <p className="font-medium">{device.manufacturer || "-"}</p>
            </div>
            <div>
              <p className="text-sm text-gray-500">型号</p>
              <p className="font-medium">{device.model || "-"}</p>
            </div>
            <div>
              <p className="text-sm text-gray-500">固件版本</p>
              <p className="font-medium">{device.firmwareVersion || "-"}</p>
            </div>
            <div>
              <p className="text-sm text-gray-500">最后心跳</p>
              <p className="font-medium text-sm">
                {device.lastHeartbeat || "-"}
              </p>
            </div>
            <div>
              <p className="text-sm text-gray-500">心跳次数</p>
              <p className="font-medium">{device.heartbeatCount ?? "-"}</p>
            </div>
            <div>
              <p className="text-sm text-gray-500">离线次数</p>
              <p className="font-medium">{device.offlineCount ?? "-"}</p>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* Tabs */}
      <Tabs value={activeTab} onValueChange={setActiveTab}>
        <TabsList>
          <TabsTrigger value="overview">概览</TabsTrigger>
          <TabsTrigger value="faults">故障</TabsTrigger>
          <TabsTrigger value="logs">日志</TabsTrigger>
          <TabsTrigger value="performance">性能</TabsTrigger>
        </TabsList>

        {/* Overview Tab */}
        <TabsContent value="overview" className="mt-4 space-y-6">
          {/* Vendor specific config */}
          <Card>
            <CardHeader>
              <CardTitle>厂商配置</CardTitle>
            </CardHeader>
            <CardContent>
              {statsLoading ? (
                <div className="flex items-center justify-center py-8">
                  <Loader2 className="w-6 h-6 animate-spin text-gray-400" />
                </div>
              ) : !device.vendorSpecificConfig ||
                Object.keys(device.vendorSpecificConfig).length === 0 ? (
                <p className="text-gray-500 text-center py-4">
                  暂无厂商配置数据
                </p>
              ) : (
                <div className="border rounded-lg">
                  <table className="w-full">
                    <thead className="bg-gray-50 border-b">
                      <tr>
                        <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">
                          配置项
                        </th>
                        <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">
                          配置值
                        </th>
                      </tr>
                    </thead>
                    <tbody>
                      {Object.entries<string>(device.vendorSpecificConfig).map(
                        ([key, value]) => (
                          <tr
                            key={key}
                            className="border-b last:border-0 hover:bg-gray-50"
                          >
                            <td className="px-4 py-3 font-medium text-sm">
                              {key}
                            </td>
                            <td className="px-4 py-3 text-sm text-gray-600">
                              {String(value)}
                            </td>
                          </tr>
                        )
                      )}
                    </tbody>
                  </table>
                </div>
              )}
            </CardContent>
          </Card>

          {/* Stats summary */}
          <Card>
            <CardHeader>
              <CardTitle>统计概览</CardTitle>
            </CardHeader>
            <CardContent>
              {statsLoading ? (
                <div className="flex items-center justify-center py-8">
                  <Loader2 className="w-6 h-6 animate-spin text-gray-400" />
                </div>
              ) : !statsSummary ||
                Object.keys(statsSummary).length === 0 ? (
                <p className="text-gray-500 text-center py-4">
                  暂无统计数据
                </p>
              ) : (
                <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
                  {Object.entries(statsSummary).map(([key, value]) => (
                    <div
                      key={key}
                      className="rounded-lg border p-4 text-center"
                    >
                      <p className="text-2xl font-semibold text-gray-900">
                        {typeof value === "number"
                          ? value.toLocaleString()
                          : String(value)}
                      </p>
                      <p className="text-sm text-gray-500 mt-1">{key}</p>
                    </div>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        {/* Faults Tab */}
        <TabsContent value="faults" className="mt-4">
          <Card>
            <CardHeader>
              <CardTitle>故障记录</CardTitle>
            </CardHeader>
            <CardContent>
              {faultsLoading ? (
                <div className="flex items-center justify-center py-12">
                  <Loader2 className="w-8 h-8 animate-spin text-gray-400" />
                </div>
              ) : faults.length === 0 ? (
                <p className="text-gray-500 text-center py-8">
                  暂无故障记录
                </p>
              ) : (
                <>
                  <div className="border rounded-lg">
                    <table className="w-full">
                      <thead className="bg-gray-50 border-b">
                        <tr>
                          <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">
                            故障类型
                          </th>
                          <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">
                            故障码
                          </th>
                          <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">
                            描述
                          </th>
                          <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">
                            严重级别
                          </th>
                          <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">
                            状态
                          </th>
                          <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">
                            检测时间
                          </th>
                          <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">
                            操作
                          </th>
                        </tr>
                      </thead>
                      <tbody>
                        {faults.map((fault) => (
                          <tr
                            key={fault.id}
                            className="border-b last:border-0 hover:bg-gray-50"
                          >
                            <td className="px-4 py-3 text-sm font-medium">
                              {fault.faultType}
                            </td>
                            <td className="px-4 py-3 text-sm font-mono">
                              {fault.faultCode}
                            </td>
                            <td className="px-4 py-3 text-sm text-gray-600 max-w-xs truncate">
                              {fault.description}
                            </td>
                            <td className="px-4 py-3">
                              <Badge
                                variant={
                                  FAULT_SEVERITY_VARIANT[fault.severity] ||
                                  "outline"
                                }
                              >
                                {FAULT_SEVERITY_LABELS[fault.severity] ||
                                  fault.severity}
                              </Badge>
                            </td>
                            <td className="px-4 py-3">
                              <Badge
                                variant={
                                  FAULT_STATUS_VARIANT[fault.status] ||
                                  "outline"
                                }
                              >
                                {FAULT_STATUS_LABELS[fault.status] ||
                                  fault.status}
                              </Badge>
                            </td>
                            <td className="px-4 py-3 text-sm text-gray-600">
                              {fault.detectedAt}
                            </td>
                            <td className="px-4 py-3">
                              {fault.status !== "resolved" &&
                                fault.status !== "closed" && (
                                  <Button
                                    variant="outline"
                                    size="sm"
                                    onClick={() =>
                                      resolveFaultMutation.mutate(fault.id)
                                    }
                                    disabled={resolveFaultMutation.isPending}
                                  >
                                    {resolveFaultMutation.isPending ? (
                                      <Loader2 className="w-3 h-3 mr-1 animate-spin" />
                                    ) : (
                                      <CheckCircle className="w-3 h-3 mr-1" />
                                    )}
                                    解决
                                  </Button>
                                )}
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>

                  {/* Pagination */}
                  {faultsTotalPages > 1 && (
                    <div className="flex items-center justify-between mt-4">
                      <p className="text-sm text-gray-500">
                        共 {faultsTotal} 条记录，第 {faultPage}/{faultsTotalPages} 页
                      </p>
                      <div className="flex items-center gap-2">
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() =>
                            setFaultPage((p) => Math.max(1, p - 1))
                          }
                          disabled={faultPage <= 1}
                        >
                          上一页
                        </Button>
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() =>
                            setFaultPage((p) =>
                              Math.min(faultsTotalPages, p + 1)
                            )
                          }
                          disabled={faultPage >= faultsTotalPages}
                        >
                          下一页
                        </Button>
                      </div>
                    </div>
                  )}
                </>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        {/* Logs Tab */}
        <TabsContent value="logs" className="mt-4">
          <Card>
            <CardHeader>
              <CardTitle>设备日志</CardTitle>
            </CardHeader>
            <CardContent>
              {logsLoading ? (
                <div className="flex items-center justify-center py-12">
                  <Loader2 className="w-8 h-8 animate-spin text-gray-400" />
                </div>
              ) : logs.length === 0 ? (
                <p className="text-gray-500 text-center py-8">
                  暂无日志记录
                </p>
              ) : (
                <>
                  <div className="border rounded-lg">
                    <table className="w-full">
                      <thead className="bg-gray-50 border-b">
                        <tr>
                          <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">
                            级别
                          </th>
                          <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">
                            日志类型
                          </th>
                          <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">
                            消息
                          </th>
                          <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">
                            时间
                          </th>
                        </tr>
                      </thead>
                      <tbody>
                        {logs.map((log) => (
                          <tr
                            key={log.id}
                            className="border-b last:border-0 hover:bg-gray-50"
                          >
                            <td className="px-4 py-3">
                              <Badge
                                variant={
                                  LOG_LEVEL_VARIANT[log.level] || "outline"
                                }
                              >
                                {LOG_LEVEL_LABELS[log.level] || log.level}
                              </Badge>
                            </td>
                            <td className="px-4 py-3 text-sm">
                              {log.logType}
                            </td>
                            <td className="px-4 py-3 text-sm text-gray-600 max-w-md truncate">
                              {log.message}
                            </td>
                            <td className="px-4 py-3 text-sm text-gray-500">
                              {log.createdAt}
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>

                  {/* Pagination */}
                  {logsTotalPages > 1 && (
                    <div className="flex items-center justify-between mt-4">
                      <p className="text-sm text-gray-500">
                        共 {logsTotal} 条记录，第 {logPage}/{logsTotalPages} 页
                      </p>
                      <div className="flex items-center gap-2">
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() =>
                            setLogPage((p) => Math.max(1, p - 1))
                          }
                          disabled={logPage <= 1}
                        >
                          上一页
                        </Button>
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() =>
                            setLogPage((p) =>
                              Math.min(logsTotalPages, p + 1)
                            )
                          }
                          disabled={logPage >= logsTotalPages}
                        >
                          下一页
                        </Button>
                      </div>
                    </div>
                  )}
                </>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        {/* Performance Tab */}
        <TabsContent value="performance" className="mt-4">
          <Card>
            <CardHeader>
              <CardTitle>性能趋势</CardTitle>
            </CardHeader>
            <CardContent>
              {performanceLoading ? (
                <div className="flex items-center justify-center py-12">
                  <Loader2 className="w-8 h-8 animate-spin text-gray-400" />
                </div>
              ) : !performanceData || performanceData.length === 0 ? (
                <p className="text-gray-500 text-center py-8">
                  暂无性能数据
                </p>
              ) : (
                <div className="space-y-8">
                  {/* CPU / Memory / Temperature chart */}
                  <div>
                    <h3 className="text-sm font-medium text-gray-700 mb-3">
                      CPU / 内存 / 温度
                    </h3>
                    <ResponsiveContainer width="100%" height={320}>
                      <AreaChart
                        data={chartData}
                        margin={{ top: 5, right: 30, left: 0, bottom: 5 }}
                      >
                        <CartesianGrid
                          strokeDasharray="3 3"
                          stroke="#f0f0f0"
                        />
                        <XAxis
                          dataKey="time"
                          tick={{ fill: "#6b7280", fontSize: 12 }}
                          axisLine={{ stroke: "#e5e7eb" }}
                        />
                        <YAxis
                          tick={{ fill: "#6b7280", fontSize: 12 }}
                          axisLine={{ stroke: "#e5e7eb" }}
                          unit="%"
                        />
                        <Tooltip
                          contentStyle={{
                            backgroundColor: "white",
                            border: "1px solid #e5e7eb",
                            borderRadius: "8px",
                            boxShadow:
                              "0 4px 6px -1px rgb(0 0 0 / 0.1)",
                          }}
                        />
                        <Area
                          type="monotone"
                          dataKey="cpuUsage"
                          name="CPU 使用率"
                          stroke="#3b82f6"
                          fill="#3b82f6"
                          fillOpacity={0.1}
                          strokeWidth={2}
                        />
                        <Area
                          type="monotone"
                          dataKey="memoryUsage"
                          name="内存使用率"
                          stroke="#10b981"
                          fill="#10b981"
                          fillOpacity={0.1}
                          strokeWidth={2}
                        />
                        <Area
                          type="monotone"
                          dataKey="temperature"
                          name="温度"
                          stroke="#f59e0b"
                          fill="#f59e0b"
                          fillOpacity={0.1}
                          strokeWidth={2}
                        />
                      </AreaChart>
                    </ResponsiveContainer>
                  </div>

                  {/* Network chart */}
                  <div>
                    <h3 className="text-sm font-medium text-gray-700 mb-3">
                      网络流量
                    </h3>
                    <ResponsiveContainer width="100%" height={260}>
                      <AreaChart
                        data={chartData}
                        margin={{ top: 5, right: 30, left: 0, bottom: 5 }}
                      >
                        <CartesianGrid
                          strokeDasharray="3 3"
                          stroke="#f0f0f0"
                        />
                        <XAxis
                          dataKey="time"
                          tick={{ fill: "#6b7280", fontSize: 12 }}
                          axisLine={{ stroke: "#e5e7eb" }}
                        />
                        <YAxis
                          tick={{ fill: "#6b7280", fontSize: 12 }}
                          axisLine={{ stroke: "#e5e7eb" }}
                        />
                        <Tooltip
                          contentStyle={{
                            backgroundColor: "white",
                            border: "1px solid #e5e7eb",
                            borderRadius: "8px",
                            boxShadow:
                              "0 4px 6px -1px rgb(0 0 0 / 0.1)",
                          }}
                        />
                        <Area
                          type="monotone"
                          dataKey="networkIn"
                          name="入站流量"
                          stroke="#8b5cf6"
                          fill="#8b5cf6"
                          fillOpacity={0.1}
                          strokeWidth={2}
                        />
                        <Area
                          type="monotone"
                          dataKey="networkOut"
                          name="出站流量"
                          stroke="#ec4899"
                          fill="#ec4899"
                          fillOpacity={0.1}
                          strokeWidth={2}
                        />
                      </AreaChart>
                    </ResponsiveContainer>
                  </div>

                  {/* Latest snapshot */}
                  {performanceData.length > 0 && (
                    <div>
                      <h3 className="text-sm font-medium text-gray-700 mb-3">
                        最新快照
                      </h3>
                      <div className="grid grid-cols-2 md:grid-cols-5 gap-4">
                        {[
                          {
                            label: "CPU 使用率",
                            value: performanceData[0].cpuUsage,
                            unit: "%",
                          },
                          {
                            label: "内存使用率",
                            value: performanceData[0].memoryUsage,
                            unit: "%",
                          },
                          {
                            label: "存储使用率",
                            value: performanceData[0].storageUsage,
                            unit: "%",
                          },
                          {
                            label: "温度",
                            value: performanceData[0].temperature,
                            unit: "\u00b0C",
                          },
                          {
                            label: "网络入站",
                            value: performanceData[0].networkIn,
                            unit: "KB",
                          },
                        ].map((item) => (
                          <div
                            key={item.label}
                            className="rounded-lg border p-4 text-center"
                          >
                            <p className="text-2xl font-semibold text-gray-900">
                              {item.value}
                              <span className="text-sm font-normal text-gray-500 ml-1">
                                {item.unit}
                              </span>
                            </p>
                            <p className="text-sm text-gray-500 mt-1">
                              {item.label}
                            </p>
                          </div>
                        ))}
                      </div>
                    </div>
                  )}
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  );
}
