import { Badge } from '@/components/ui';
import { ordersData } from '@/lib/constants';

/**
 * OrdersContent - Orders management tab
 */
export function OrdersContent() {
  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-xl font-semibold text-slate-900 dark:text-white">订单管理</h3>
        <button className="px-4 py-2 text-sm border border-slate-200 dark:border-slate-600 rounded-lg">导出报表</button>
      </div>
      <div className="card rounded-xl overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-slate-50 dark:bg-slate-800">
            <tr>
              <th className="text-left p-3 text-slate-600 dark:text-slate-400">订单号</th>
              <th className="text-left p-3 text-slate-600 dark:text-slate-400">车牌号</th>
              <th className="text-left p-3 text-slate-600 dark:text-slate-400">金额</th>
              <th className="text-left p-3 text-slate-600 dark:text-slate-400">支付方式</th>
              <th className="text-left p-3 text-slate-600 dark:text-slate-400">状态</th>
            </tr>
          </thead>
          <tbody>
            {ordersData.map((o) => (
              <tr key={o.id} className="border-t border-slate-100 dark:border-slate-700">
                <td className="p-3 font-medium text-slate-900 dark:text-white">{o.id}</td>
                <td className="p-3 text-slate-600 dark:text-slate-400">{o.plate}</td>
                <td className="p-3 font-medium text-slate-900 dark:text-white">{o.amount}</td>
                <td className="p-3 text-slate-600 dark:text-slate-400">{o.method}</td>
                <td className="p-3">
                  <Badge variant="success">{o.status}</Badge>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}