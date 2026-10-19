"use client";

import { StatsCard } from "@/components/modules/stats-card";
import { TrendChart } from "@/components/modules/trend-chart";
import { formatCentsGrouped } from "@/lib/utils";
import { DollarSign, Car, Activity, Server, Zap, Battery } from "lucide-react";
import { useApiQuery } from "@/hooks/use-api";
import { getDashboardStats, getTrendData } from "@/services/dashboard";
import { chargingService, Session, Station } from "@/services/charging";

export default function DashboardPage() {
  const { data: stats, isLoading: statsLoading, error: statsError } = useApiQuery(
    ["dashboard-stats"],
    getDashboardStats
  );

  const { data: trendData, isLoading: trendLoading, error: trendError } = useApiQuery(
    ["dashboard-trend"],
    getTrendData
  );

  const { data: chargingSessions } = useApiQuery<Session[]>(
    ["dashboard-charging-sessions"],
    () => chargingService.listUserSessions("", 1, 100),
    { refetchOnWindowFocus: false }
  );

  const { data: chargingStations } = useApiQuery<Station[]>(
    ["dashboard-charging-stations"],
    () => chargingService.listStations(undefined, 1, 100),
    { refetchOnWindowFocus: false }
  );

  const chargingSessionsData = chargingSessions || [];
  const chargingStationsData = chargingStations || [];
  const activeChargingCount = chargingSessionsData.filter((s) => s.status === "charging").length;
  const totalChargingEnergy = chargingSessionsData.reduce((sum, s) => sum + s.chargedEnergy, 0);
  const totalChargingRevenue = chargingSessionsData.reduce((sum, s) => sum + s.totalAmount, 0);

  if (statsError || trendError) {
    return (
      <div className="space-y-6">
        <div>
          <h1 className="text-2xl font-semibold text-gray-900">数据概览</h1>
          <p className="text-gray-500 mt-1">实时掌握停车场运营情况</p>
        </div>
        <div className="bg-white rounded-xl p-6 border border-gray-100 shadow-sm">
          <p className="text-red-500">加载数据失败，请稍后重试</p>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold text-gray-900">数据概览</h1>
        <p className="text-gray-500 mt-1">实时掌握停车场运营情况</p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        {statsLoading ? (
          <>
            <StatsCardSkeleton />
            <StatsCardSkeleton />
            <StatsCardSkeleton />
            <StatsCardSkeleton />
          </>
        ) : stats ? (
          <>
            <StatsCard
              title="今日收入"
              value={`¥${formatCentsGrouped(stats.todayIncome)}`}
              change={stats.todayIncomeChange}
              trend={stats.todayIncomeChange >= 0 ? "up" : "down"}
              icon={<DollarSign className="w-6 h-6" />}
            />
            <StatsCard
              title="今日入场"
              value={stats.todayEntries}
              change={stats.todayEntriesChange}
              trend={stats.todayEntriesChange >= 0 ? "up" : "down"}
              icon={<Car className="w-6 h-6" />}
            />
            <StatsCard
              title="今日出场"
              value={stats.todayExits}
              change={stats.todayExitsChange}
              trend={stats.todayExitsChange >= 0 ? "up" : "down"}
              icon={<Activity className="w-6 h-6" />}
            />
            <StatsCard
              title="在线设备"
              value={stats.onlineDevices}
              icon={<Server className="w-6 h-6" />}
            />
          </>
        ) : null}
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <StatsCard
          title="充电桩总数"
          value={chargingStationsData.length}
          icon={<Zap className="w-6 h-6" />}
        />
        <StatsCard
          title="充电中"
          value={activeChargingCount}
          icon={<Battery className="w-6 h-6" />}
        />
        <StatsCard
          title="充电收入"
          value={`¥${formatCentsGrouped(totalChargingRevenue)}`}
          icon={<DollarSign className="w-6 h-6" />}
        />
      </div>

      <div className="bg-white rounded-xl p-6 border border-gray-100 shadow-sm">
        <h2 className="text-lg font-medium mb-4">7天趋势</h2>
        {trendLoading ? (
          <div className="h-[300px] flex items-center justify-center text-gray-400">
            加载中...
          </div>
        ) : trendData ? (
          <TrendChart data={trendData} />
        ) : null}
      </div>
    </div>
  );
}

function StatsCardSkeleton() {
  return (
    <div className="bg-white rounded-xl p-6 border border-gray-100 shadow-sm animate-pulse">
      <div className="flex items-center justify-between">
        <div>
          <div className="h-4 w-16 bg-gray-200 rounded mb-2"></div>
          <div className="h-8 w-24 bg-gray-200 rounded"></div>
        </div>
        <div className="w-12 h-12 bg-gray-200 rounded-lg"></div>
      </div>
    </div>
  );
}
