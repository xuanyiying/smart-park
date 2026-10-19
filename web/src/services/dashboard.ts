import { adminService } from "./admin";
import { deviceService } from "./device";

export interface DashboardStats {
  todayIncome: number;
  todayEntries: number;
  todayExits: number;
  onlineDevices: number;
  totalDevices: number;
  todayIncomeChange: number;
  todayEntriesChange: number;
  todayExitsChange: number;
}

export interface TrendData {
  date: string;
  entries: number;
  exits: number;
  amount: number;
}

/** 计算环比涨跌幅（百分比，保留整数）；前期为 0 时无法计算，返回 null。 */
function percentChange(current: number, previous: number): number | null {
  if (previous === 0) {
    return current > 0 ? 100 : null;
  }
  return Math.round(((current - previous) / previous) * 100);
}

/**
 * Get dashboard statistics.
 * 所有数据均来自后端真实接口：
 * - 收入/出入场：admin 服务的今日与昨日日报（用于环比）
 * - 设备在线数：vehicle 服务的设备列表（online 字段）
 */
export async function getDashboardStats(): Promise<DashboardStats> {
  const empty: DashboardStats = {
    todayIncome: 0,
    todayEntries: 0,
    todayExits: 0,
    onlineDevices: 0,
    totalDevices: 0,
    todayIncomeChange: 0,
    todayEntriesChange: 0,
    todayExitsChange: 0,
  };

  try {
    const todayDate = new Date();
    const yesterdayDate = new Date(todayDate);
    yesterdayDate.setDate(yesterdayDate.getDate() - 1);
    const today = todayDate.toISOString().split("T")[0];
    const yesterday = yesterdayDate.toISOString().split("T")[0];

    const [parkingLots, devices] = await Promise.all([
      adminService.listParkingLots(1, 100),
      deviceService.listDevices(1, 500).catch(() => null),
    ]);

    let todayIncome = 0;
    let todayEntries = 0;
    let todayExits = 0;
    let yesterdayIncome = 0;
    let yesterdayEntries = 0;
    let yesterdayExits = 0;

    await Promise.all(
      parkingLots.list.map(async (lot) => {
        const [todayReport, yesterdayReport] = await Promise.all([
          adminService.getDailyReport(lot.id, today).catch(() => null),
          adminService.getDailyReport(lot.id, yesterday).catch(() => null),
        ]);
        if (todayReport) {
          todayIncome += todayReport.netAmount || 0;
          todayEntries += todayReport.totalEntries || 0;
          todayExits += todayReport.totalExits || 0;
        }
        if (yesterdayReport) {
          yesterdayIncome += yesterdayReport.netAmount || 0;
          yesterdayEntries += yesterdayReport.totalEntries || 0;
          yesterdayExits += yesterdayReport.totalExits || 0;
        }
      }),
    );

    const onlineDevices = devices
      ? devices.list.filter((d) => d.online).length
      : 0;

    return {
      todayIncome,
      todayEntries,
      todayExits,
      onlineDevices,
      totalDevices: devices?.total ?? 0,
      todayIncomeChange: percentChange(todayIncome, yesterdayIncome) ?? 0,
      todayEntriesChange: percentChange(todayEntries, yesterdayEntries) ?? 0,
      todayExitsChange: percentChange(todayExits, yesterdayExits) ?? 0,
    };
  } catch (error) {
    console.error("Failed to fetch dashboard stats:", error);
    return empty;
  }
}

/**
 * Get trend data for the past 7 days
 */
export async function getTrendData(): Promise<TrendData[]> {
  try {
    const parkingLots = await adminService.listParkingLots(1, 100);
    const today = new Date();
    const data: TrendData[] = [];

    for (let i = 6; i >= 0; i--) {
      const date = new Date(today);
      date.setDate(date.getDate() - i);
      const dateStr = date.toISOString().split("T")[0];
      const displayDate = `${date.getMonth() + 1}/${date.getDate()}`;

      let dayEntries = 0;
      let dayExits = 0;
      let dayAmount = 0;

      // Aggregate data from all parking lots
      for (const lot of parkingLots.list) {
        try {
          const report = await adminService.getDailyReport(lot.id, dateStr);
          dayEntries += report.totalEntries || 0;
          dayExits += report.totalExits || 0;
          dayAmount += report.netAmount || 0;
        } catch (error) {
          console.error(`Failed to fetch report for lot ${lot.id} on ${dateStr}:`, error);
        }
      }

      data.push({
        date: displayDate,
        entries: dayEntries,
        exits: dayExits,
        amount: dayAmount,
      });
    }

    return data;
  } catch (error) {
    console.error("Failed to fetch trend data:", error);
    return [];
  }
}
