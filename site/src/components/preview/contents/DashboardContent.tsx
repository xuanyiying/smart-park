import {
  Search,
  Bell,
  DollarSign,
  ArrowUpRight,
  ArrowDownRight,
  Server,
} from 'lucide-react';
import { dashboardStats } from '@/lib/constants';

/**
 * DashboardContent - Dashboard tab content with stats and chart
 */
export function DashboardContent() {
  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-xl font-semibold text-slate-900 dark:text-white">数据概览</h3>
          <p className="text-sm text-slate-500 dark:text-slate-400">实时掌握停车场运营情况</p>
        </div>
        <div className="flex items-center gap-2">
          <div className="relative">
            <Search className="w-4 h-4 absolute left-3 top-1/2 -translate-y-1/2 text-slate-400" />
            <input type="text" placeholder="搜索..." className="pl-9 pr-4 py-2 text-sm rounded-lg border border-slate-200 dark:border-slate-600 bg-white dark:bg-slate-800" />
          </div>
          <button className="p-2 rounded-lg border border-slate-200 dark:border-slate-600 bg-white dark:bg-slate-800">
            <Bell className="w-4 h-4 text-slate-600 dark:text-slate-400" />
          </button>
        </div>
      </div>

      {/* Stats Cards */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        {dashboardStats.map((stat) => (
          <div key={stat.label} className="card rounded-xl p-4">
            <div className="flex items-center justify-between">
              <div className="text-sm text-slate-500 dark:text-slate-400">{stat.label}</div>
              <stat.icon className="w-4 h-4 text-slate-400" />
            </div>
            <div className="text-2xl font-bold text-slate-900 dark:text-white mt-2">{stat.value}</div>
            <div className={`text-xs mt-1 ${stat.positive ? 'text-green-600 dark:text-green-400' : 'text-red-600 dark:text-red-400'}`}>
              {stat.change} 较昨日
            </div>
          </div>
        ))}
      </div>

      {/* Chart Placeholder */}
      <div className="card rounded-xl p-6">
        <h4 className="font-medium text-slate-900 dark:text-white mb-4">收入趋势</h4>
        <div className="h-48 flex items-center justify-center">
          <div className="flex items-end gap-2 h-32">
            {[40, 65, 45, 80, 55, 90, 70].map((h, i) => (
              <div key={i} className="w-8 bg-sky-500 rounded-t" style={{ height: `${h}%` }} />
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}