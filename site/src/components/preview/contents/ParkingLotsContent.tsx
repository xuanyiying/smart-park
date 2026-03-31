import { Warehouse } from 'lucide-react';
import { parkingLotsData } from '@/lib/constants';

/**
 * ParkingLotsContent - Parking lots management tab
 */
export function ParkingLotsContent() {
  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-xl font-semibold text-slate-900 dark:text-white">停车场管理</h3>
        <button className="px-4 py-2 text-sm btn-primary rounded-lg text-white">添加停车场</button>
      </div>
      <div className="grid gap-4">
        {parkingLotsData.map((lot) => (
          <div key={lot.name} className="card rounded-xl p-4 flex items-center justify-between">
            <div className="flex items-center gap-4">
              <div className="w-12 h-12 rounded-lg bg-sky-100 dark:bg-sky-900/30 flex items-center justify-center">
                <Warehouse className="w-6 h-6 text-sky-600 dark:text-sky-400" />
              </div>
              <div>
                <div className="font-medium text-slate-900 dark:text-white">{lot.name}</div>
                <div className="text-sm text-slate-500 dark:text-slate-400">{lot.address}</div>
              </div>
            </div>
            <div className="text-right">
              <div className="text-sm text-slate-900 dark:text-white">{lot.used}/{lot.spaces} 车位</div>
              <div className="text-xs text-slate-500 dark:text-slate-400">使用率 {Math.round(lot.used/lot.spaces*100)}%</div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}