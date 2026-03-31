import { Badge } from '@/components/ui';
import { vehiclesData } from '@/lib/constants';

/**
 * VehiclesContent - Vehicle management tab
 */
export function VehiclesContent() {
  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-xl font-semibold text-slate-900 dark:text-white">车辆管理</h3>
        <div className="flex gap-2">
          <input type="text" placeholder="搜索车牌号..." className="px-4 py-2 text-sm rounded-lg border border-slate-200 dark:border-slate-600 bg-white dark:bg-slate-800" />
          <button className="px-4 py-2 text-sm btn-primary rounded-lg text-white">查询</button>
        </div>
      </div>
      <div className="card rounded-xl overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-slate-50 dark:bg-slate-800">
            <tr>
              <th className="text-left p-3 text-slate-600 dark:text-slate-400">车牌号</th>
              <th className="text-left p-3 text-slate-600 dark:text-slate-400">类型</th>
              <th className="text-left p-3 text-slate-600 dark:text-slate-400">入场时间</th>
              <th className="text-left p-3 text-slate-600 dark:text-slate-400">停车场</th>
              <th className="text-left p-3 text-slate-600 dark:text-slate-400">状态</th>
            </tr>
          </thead>
          <tbody>
            {vehiclesData.map((v) => (
              <tr key={v.plate} className="border-t border-slate-100 dark:border-slate-700">
                <td className="p-3 font-medium text-slate-900 dark:text-white">{v.plate}</td>
                <td className="p-3 text-slate-600 dark:text-slate-400">{v.type}</td>
                <td className="p-3 text-slate-600 dark:text-slate-400">{v.time}</td>
                <td className="p-3 text-slate-600 dark:text-slate-400">{v.lot}</td>
                <td className="p-3">
                  <Badge variant={v.status === '在场' ? 'success' : 'default'}>
                    {v.status}
                  </Badge>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}