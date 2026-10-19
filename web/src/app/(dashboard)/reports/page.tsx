"use client";

import { toast } from "@/components/providers/toast-provider";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useApiQuery } from "@/hooks/use-api";
import { formatCentsGrouped } from "@/lib/utils";
import { adminService, type DailyReport, type MonthlyReport } from "@/services/admin";
import { ArrowRightCircle, Car, Download, FileText, Loader2, MapPin, RefreshCw, TrendingUp, Wallet, WifiOff } from "lucide-react";
import { useMemo, useState } from "react";

function getYearMonthFromDate(dateStr: string): { year: number; month: number } {
  const date = new Date(dateStr);
  return { year: date.getFullYear(), month: date.getMonth() + 1 };
}

function formatDateToMonth(year: number, month: number): string {
  return `${year}-${String(month).padStart(2, "0")}`;
}

function toCsv(headers: string[], rows: (string | number)[][]): string {
  const escape = (value: string | number) => {
    const s = String(value ?? "");
    return /[",\n]/.test(s) ? `"${s.replace(/"/g, '""')}"` : s;
  };
  return [headers, ...rows].map((row) => row.map(escape).join(",")).join("\n");
}

function downloadCsv(filename: string, content: string) {
  // 带 BOM，保证 Excel 直接打开中文不乱码
  const blob = new Blob(["\uFEFF" + content], {
    type: "text/csv;charset=utf-8;",
  });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
}


export default function ReportsPage() {
  const today = new Date().toISOString().split("T")[0];
  const firstDayOfMonth = new Date(new Date().getFullYear(), new Date().getMonth(), 1).toISOString().split("T")[0];

  const [dateRange, setDateRange] = useState({ start: firstDayOfMonth, end: today });
  const [selectedLotId, setSelectedLotId] = useState<string>("");
  const [activeTab, setActiveTab] = useState<string>("daily");

  const { data: parkingLotsData, isLoading: isLoadingLots, error: parkingLotsError, refetch: refetchLots } = useApiQuery(
    ["parking-lots"],
    () => adminService.listParkingLots(1, 100),
    { staleTime: 5 * 60 * 1000 }
  );

  const parkingLots = parkingLotsData?.list || [];
  const firstLotId = parkingLots.length > 0 ? parkingLots[0].id : "";
  const effectiveLotId = selectedLotId || firstLotId;

  const { year: startYear, month: startMonth } = getYearMonthFromDate(dateRange.start);
  const { year: endYear, month: endMonth } = getYearMonthFromDate(dateRange.end);

  const monthsToFetch = useMemo(() => {
    const months: { year: number; month: number }[] = [];
    let current = new Date(startYear, startMonth - 1, 1);
    const end = new Date(endYear, endMonth - 1, 1);
    while (current <= end) {
      months.push({ year: current.getFullYear(), month: current.getMonth() + 1 });
      current = new Date(current.getFullYear(), current.getMonth() + 1, 1);
    }
    return months;
  }, [startYear, startMonth, endYear, endMonth]);

  const monthlyReportQueries = useApiQuery(
    ["monthly-reports", effectiveLotId, monthsToFetch.map(m => `${m.year}-${m.month}`).join(",")],
    async () => {
      if (!effectiveLotId) return { dailyReports: [], monthlyReports: [] };

      const monthlyReports: MonthlyReport[] = [];
      const dailyReports: DailyReport[] = [];

      for (const { year, month } of monthsToFetch) {
        try {
          const data = await adminService.getMonthlyReport(effectiveLotId, year, month);
          monthlyReports.push(data);
          if (data.dailyReports) {
            dailyReports.push(...data.dailyReports);
          }
        } catch (error) {
          console.error(`Failed to fetch monthly report for ${year}-${month}:`, error);
        }
      }

      return { dailyReports, monthlyReports };
    },
    { enabled: !!effectiveLotId && monthsToFetch.length > 0 }
  );

  const isLoading = isLoadingLots || monthlyReportQueries.isLoading;
  const error = monthlyReportQueries.error;
  const dailyData = monthlyReportQueries.data?.dailyReports || [];
  const monthlyData = monthlyReportQueries.data?.monthlyReports || [];

  const filteredDailyData = useMemo(() => {
    return dailyData.filter((report) => {
      return report.date >= dateRange.start && report.date <= dateRange.end;
    }).sort((a, b) => b.date.localeCompare(a.date));
  }, [dailyData, dateRange.start, dateRange.end]);

  const totalEntries = filteredDailyData.reduce((sum, d) => sum + d.totalEntries, 0);
  const totalExits = filteredDailyData.reduce((sum, d) => sum + d.totalExits, 0);
  const totalAmount = filteredDailyData.reduce((sum, d) => sum + d.totalAmount, 0);
  const netAmount = filteredDailyData.reduce((sum, d) => sum + d.netAmount, 0);

  const summaryCards = [
    { label: "入场总数", value: totalEntries.toLocaleString(), icon: Car, color: "text-blue-500" },
    { label: "出场总数", value: totalExits.toLocaleString(), icon: ArrowRightCircle, color: "text-green-500" },
    { label: "收费总额", value: `¥${formatCentsGrouped(totalAmount)}`, icon: Wallet, color: "text-orange-500" },
    { label: "净收入", value: `¥${formatCentsGrouped(netAmount)}`, icon: TrendingUp, color: "text-purple-500" },
  ];

  const handleExport = () => {
    const lotName =
      parkingLots.find((lot) => lot.id === effectiveLotId)?.name || effectiveLotId;
    const fileSuffix = `${lotName}_${dateRange.start}_至_${dateRange.end}`;

    if (activeTab === "daily") {
      const csv = toCsv(
        ["日期", "入场数", "出场数", "收费金额(元)", "净收入(元)"],
        filteredDailyData.map((r) => [
          r.date,
          r.totalEntries,
          r.totalExits,
          (r.totalAmount / 100).toFixed(2),
          (r.netAmount / 100).toFixed(2),
        ]),
      );
      downloadCsv(`日报_${fileSuffix}.csv`, csv);
    } else {
      const csv = toCsv(
        ["月份", "入场数", "出场数", "收费金额(元)", "净收入(元)"],
        monthlyData.map((r) => [
          formatDateToMonth(r.year, r.month),
          r.totalEntries,
          r.totalExits,
          (r.totalAmount / 100).toFixed(2),
          (r.netAmount / 100).toFixed(2),
        ]),
      );
      downloadCsv(`月报_${fileSuffix}.csv`, csv);
    }
    toast.success("导出成功");
  };

  // 渲染错误状态
  const renderErrorState = () => {
    const errorMessage = (error as { message?: string })?.message || "";
    const isNetworkError = errorMessage.includes("Network Error") || errorMessage.includes("ECONNREFUSED");
    const isTimeout = errorMessage.includes("timeout");

    return (
      <Card>
        <CardContent className="py-16 text-center">
          <div className="w-16 h-16 rounded-full bg-red-50 flex items-center justify-center mx-auto mb-4">
            <WifiOff className="w-8 h-8 text-red-400" />
          </div>
          <h3 className="text-lg font-medium text-gray-900 mb-1">
            {isNetworkError ? "网络连接失败" : isTimeout ? "请求超时" : "加载失败"}
          </h3>
          <p className="text-gray-500 mb-4 max-w-sm mx-auto">
            {isNetworkError
              ? "无法连接到服务器，请检查网络连接或稍后重试"
              : isTimeout
                ? "服务器响应时间过长，请稍后重试"
                : "获取报表数据时出现问题，请稍后重试"}
          </p>
          <Button variant="outline" onClick={() => monthlyReportQueries.refetch()}>
            <RefreshCw className="w-4 h-4 mr-2" />
            重新加载
          </Button>
        </CardContent>
      </Card>
    );
  };

  // 渲染空状态
  const renderEmptyState = () => (
    <Card>
      <CardContent className="py-16 text-center">
        <div className="w-16 h-16 rounded-full bg-gray-100 flex items-center justify-center mx-auto mb-4">
          <MapPin className="w-8 h-8 text-gray-400" />
        </div>
        <h3 className="text-lg font-medium text-gray-900 mb-1">暂无停车场数据</h3>
        <p className="text-gray-500 mb-4">请先创建停车场后再查看报表</p>
      </CardContent>
    </Card>
  );

  // 渲染无数据状态
  const renderNoDataState = () => (
    <Card>
      <CardContent className="py-16 text-center">
        <div className="w-16 h-16 rounded-full bg-gray-100 flex items-center justify-center mx-auto mb-4">
          <FileText className="w-8 h-8 text-gray-400" />
        </div>
        <h3 className="text-lg font-medium text-gray-900 mb-1">暂无报表数据</h3>
        <p className="text-gray-500 mb-4">当前日期范围内没有运营数据</p>
      </CardContent>
    </Card>
  );

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold text-gray-900">报表统计</h1>
          <p className="text-gray-500 mt-1">查看停车场运营数据报表</p>
        </div>
        <Button
          onClick={handleExport}
          disabled={isLoading || (activeTab === "daily" ? filteredDailyData.length === 0 : monthlyData.length === 0)}
        >
          <Download className="w-4 h-4 mr-2" />
          导出
        </Button>
      </div>

      <Card>
        <CardHeader className="pb-3">
          <div className="flex items-center gap-4 flex-wrap">
            <div className="flex items-center gap-2">
              <label className="text-sm text-gray-600">停车场:</label>
              <Select value={selectedLotId || firstLotId} onValueChange={(value) => value && setSelectedLotId(value)}>
                <SelectTrigger className="w-48">
                  <SelectValue>
                    {parkingLots.find(lot => lot.id === (selectedLotId || firstLotId))?.name || "选择停车场"}
                  </SelectValue>
                </SelectTrigger>
                <SelectContent>
                  {parkingLots.map((lot) => (
                    <SelectItem key={lot.id} value={lot.id}>
                      {lot.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="flex items-center gap-2">
              <label className="text-sm text-gray-600">开始日期:</label>
              <input
                type="date"
                value={dateRange.start}
                onChange={(e) => setDateRange({ ...dateRange, start: e.target.value })}
                className="px-3 py-2 border rounded-md text-sm"
              />
            </div>
            <div className="flex items-center gap-2">
              <label className="text-sm text-gray-600">结束日期:</label>
              <input
                type="date"
                value={dateRange.end}
                onChange={(e) => setDateRange({ ...dateRange, end: e.target.value })}
                className="px-3 py-2 border rounded-md text-sm"
              />
            </div>
          </div>
        </CardHeader>
      </Card>

      {isLoading ? (
        <div className="flex items-center justify-center py-16">
          <Loader2 className="w-8 h-8 animate-spin text-gray-400" />
        </div>
      ) : error ? (
        renderErrorState()
      ) : !effectiveLotId ? (
        renderEmptyState()
      ) : filteredDailyData.length === 0 ? (
        renderNoDataState()
      ) : (
        <>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
            {summaryCards.map((card, index) => (
              <Card key={index}>
                <CardContent className="pt-6">
                  <div className="flex items-center justify-between">
                    <div>
                      <p className="text-sm text-gray-500">{card.label}</p>
                      <p className="text-2xl font-semibold mt-1">{card.value}</p>
                    </div>
                    <div className={`p-3 rounded-full bg-gray-100 ${card.color}`}>
                      <card.icon className="w-6 h-6" />
                    </div>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>

          <Tabs value={activeTab} onValueChange={setActiveTab} className="space-y-4">
            <TabsList>
              <TabsTrigger value="daily">日报</TabsTrigger>
              <TabsTrigger value="monthly">月报</TabsTrigger>
            </TabsList>

            <TabsContent value="daily">
              <Card>
                <CardHeader>
                  <h3 className="text-lg font-medium">日报明细</h3>
                </CardHeader>
                <CardContent>
                  <div className="border rounded-lg overflow-hidden">
                    <table className="w-full">
                      <thead className="bg-gray-50 border-b">
                        <tr>
                          <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">日期</th>
                          <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">入场数</th>
                          <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">出场数</th>
                          <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">收费金额</th>
                          <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">净收入</th>
                        </tr>
                      </thead>
                      <tbody>
                        {filteredDailyData.map((report) => (
                          <tr key={report.date} className="border-b last:border-0 hover:bg-gray-50">
                            <td className="px-4 py-3">{report.date}</td>
                            <td className="px-4 py-3">{report.totalEntries}</td>
                            <td className="px-4 py-3">{report.totalExits}</td>
                            <td className="px-4 py-3">¥{formatCentsGrouped(report.totalAmount)}</td>
                            <td className="px-4 py-3">¥{formatCentsGrouped(report.netAmount)}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                </CardContent>
              </Card>
            </TabsContent>

            <TabsContent value="monthly">
              <Card>
                <CardHeader>
                  <h3 className="text-lg font-medium">月报汇总</h3>
                </CardHeader>
                <CardContent>
                  <div className="border rounded-lg overflow-hidden">
                    <table className="w-full">
                      <thead className="bg-gray-50 border-b">
                        <tr>
                          <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">月份</th>
                          <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">入场数</th>
                          <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">出场数</th>
                          <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">收费金额</th>
                          <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">净收入</th>
                        </tr>
                      </thead>
                      <tbody>
                        {monthlyData.map((report) => (
                          <tr key={report.month} className="border-b last:border-0 hover:bg-gray-50">
                            <td className="px-4 py-3">{formatDateToMonth(report.year, report.month)}</td>
                            <td className="px-4 py-3">{report.totalEntries}</td>
                            <td className="px-4 py-3">{report.totalExits}</td>
                            <td className="px-4 py-3">¥{formatCentsGrouped(report.totalAmount)}</td>
                            <td className="px-4 py-3">¥{formatCentsGrouped(report.netAmount)}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                </CardContent>
              </Card>
            </TabsContent>
          </Tabs>
        </>
      )}
    </div>
  );
}
