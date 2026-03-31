import { reportsStats, reportsChartData } from '@/lib/constants';

/**
 * ReportsContent - Reports and statistics tab
 */
export function ReportsContent() {
  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-xl font-semibold text-slate-900 dark:text-white">报表统计</h3>
        <div className="flex gap-2">
          <select className="px-4 py-2 text-sm rounded-lg border border-slate-200 dark:border-slate-600 bg-white dark:bg-slate-800">
            <option>本月</option>
            <option>上月</option>
            <option>近三月</option>
          </select>
          <button className="px-4 py-2 text-sm btn-primary rounded-lg text-white">导出</button>
        </div>
      </div>
      <div className="grid grid-cols-3 gap-4">
        {reportsStats.map((stat) => (
          <div key={stat.label} className="card rounded-xl p-4 text-center">
            <div className="text-3xl font-bold text-slate-900 dark:text-white">{stat.value}</div>
            <div className="text-sm text-slate-500 dark:text-slate-400 mt-1">{stat.label}</div>
          </div>
        ))}
      </div>
      <div className="card rounded-xl p-6">
        <h4 className="font-medium text-slate-900 dark:text-white mb-4">月度趋势</h4>
        <div className="h-40 flex items-end justify-between gap-4">
          {reportsChartData.map((data) => (
            <div key={data.month} className="flex-1 flex flex-col items-center">
              <div className="w-full bg-blue-500 rounded-t" style={{ height: `${data.value}%` }} />
              <div className="text-xs text-slate-500 dark:text-slate-400 mt-2">{data.month}</div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}