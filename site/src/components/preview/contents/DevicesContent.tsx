import { devicesData } from '@/lib/constants';

/**
 * DevicesContent - Device control tab
 */
export function DevicesContent() {
  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-xl font-semibold text-slate-900 dark:text-white">设备控制</h3>
        <button className="px-4 py-2 text-sm btn-primary rounded-lg text-white">添加设备</button>
      </div>
      <div className="grid grid-cols-2 gap-4">
        {devicesData.map((device) => (
          <div key={device.name} className="card rounded-xl p-4">
            <div className="flex items-center justify-between mb-3">
              <div className="font-medium text-slate-900 dark:text-white">{device.name}</div>
              <span className={`w-2 h-2 rounded-full ${device.status === '在线' ? 'bg-green-500' : 'bg-red-500'}`} />
            </div>
            <div className="text-sm text-slate-500 dark:text-slate-400">{device.type}</div>
            <div className="text-sm text-slate-500 dark:text-slate-400">{device.location}</div>
            <div className="mt-3 flex gap-2">
              <button className="px-3 py-1 text-xs border border-slate-200 dark:border-slate-600 rounded">开闸</button>
              <button className="px-3 py-1 text-xs border border-slate-200 dark:border-slate-600 rounded">关闸</button>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}