import { useState } from 'react';
import { devicesData } from '@/lib/constants';

/**
 * DevicesContent - Device control tab
 */
export function DevicesContent() {
  const [activeTab, setActiveTab] = useState('list');
  const [selectedDevice, setSelectedDevice] = useState(devicesData[0]);

  const tabs = [
    { id: 'list', label: '设备列表' },
    { id: 'monitoring', label: '设备监控' },
    { id: 'firmware', label: '远程升级' },
    { id: 'diagnostic', label: '故障诊断' },
  ];

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-xl font-semibold text-slate-900 dark:text-white">设备管理</h3>
        <button className="px-4 py-2 text-sm btn-primary rounded-lg text-white">添加设备</button>
      </div>

      {/* Tab Navigation */}
      <div className="flex border-b border-slate-200 dark:border-slate-700">
        {tabs.map((tab) => (
          <button
            key={tab.id}
            onClick={() => setActiveTab(tab.id)}
            className={`px-4 py-2 text-sm font-medium ${activeTab === tab.id 
              ? 'border-b-2 border-blue-500 text-blue-600 dark:text-blue-400' 
              : 'text-slate-500 dark:text-slate-400 hover:text-slate-700 dark:hover:text-slate-300'}`}
          >
            {tab.label}
          </button>
        ))}
      </div>

      {/* Tab Content */}
      <div className="space-y-4">
        {/* Device List Tab */}
        {activeTab === 'list' && (
          <div className="grid grid-cols-2 gap-4">
            {devicesData.map((device) => (
              <div 
                key={device.name} 
                className="card rounded-xl p-4 cursor-pointer hover:shadow-md transition-shadow"
                onClick={() => setSelectedDevice(device)}
              >
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
        )}

        {/* Device Monitoring Tab */}
        {activeTab === 'monitoring' && (
          <div className="space-y-4">
            <div className="card rounded-xl p-4">
              <h4 className="font-medium text-slate-900 dark:text-white mb-3">{selectedDevice.name} - 实时状态</h4>
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <div className="flex justify-between">
                    <span className="text-sm text-slate-500 dark:text-slate-400">设备状态</span>
                    <span className={`text-sm font-medium ${selectedDevice.status === '在线' ? 'text-green-600 dark:text-green-400' : 'text-red-600 dark:text-red-400'}`}>
                      {selectedDevice.status}
                    </span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-sm text-slate-500 dark:text-slate-400">设备类型</span>
                    <span className="text-sm font-medium text-slate-900 dark:text-white">{selectedDevice.type}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-sm text-slate-500 dark:text-slate-400">位置</span>
                    <span className="text-sm font-medium text-slate-900 dark:text-white">{selectedDevice.location}</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-sm text-slate-500 dark:text-slate-400">固件版本</span>
                    <span className="text-sm font-medium text-slate-900 dark:text-white">1.0.0</span>
                  </div>
                </div>
                <div className="space-y-2">
                  <div className="flex justify-between">
                    <span className="text-sm text-slate-500 dark:text-slate-400">CPU 使用率</span>
                    <span className="text-sm font-medium text-slate-900 dark:text-white">25%</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-sm text-slate-500 dark:text-slate-400">内存使用率</span>
                    <span className="text-sm font-medium text-slate-900 dark:text-white">45%</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-sm text-slate-500 dark:text-slate-400">网络状态</span>
                    <span className="text-sm font-medium text-green-600 dark:text-green-400">正常</span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-sm text-slate-500 dark:text-slate-400">最后心跳</span>
                    <span className="text-sm font-medium text-slate-900 dark:text-white">2分钟前</span>
                  </div>
                </div>
              </div>
            </div>
          </div>
        )}

        {/* Firmware Update Tab */}
        {activeTab === 'firmware' && (
          <div className="space-y-4">
            <div className="card rounded-xl p-4">
              <h4 className="font-medium text-slate-900 dark:text-white mb-3">{selectedDevice.name} - 固件升级</h4>
              <div className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">当前版本</label>
                  <div className="p-2 border border-slate-200 dark:border-slate-700 rounded">1.0.0</div>
                </div>
                <div>
                  <label className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-1">可用版本</label>
                  <select className="w-full p-2 border border-slate-200 dark:border-slate-700 rounded bg-white dark:bg-slate-800 text-slate-900 dark:text-white">
                    <option value="1.1.0">1.1.0 - 修复bug，优化性能</option>
                    <option value="1.0.1">1.0.1 - 修复安全漏洞</option>
                  </select>
                </div>
                <div className="flex gap-2">
                  <button className="px-4 py-2 text-sm btn-primary rounded-lg text-white flex-1">立即升级</button>
                  <button className="px-4 py-2 text-sm border border-slate-200 dark:border-slate-600 rounded flex-1">查看历史</button>
                </div>
              </div>
            </div>
          </div>
        )}

        {/* Diagnostic Tab */}
        {activeTab === 'diagnostic' && (
          <div className="space-y-4">
            <div className="card rounded-xl p-4">
              <h4 className="font-medium text-slate-900 dark:text-white mb-3">{selectedDevice.name} - 故障诊断</h4>
              <div className="space-y-4">
                <div>
                  <h5 className="text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">健康状态</h5>
                  <div className="flex items-center gap-2">
                    <span className="w-2 h-2 rounded-full bg-green-500"></span>
                    <span className="text-sm font-medium text-green-600 dark:text-green-400">健康</span>
                  </div>
                </div>
                <div>
                  <h5 className="text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">最近告警</h5>
                  <div className="space-y-2">
                    <div className="p-2 border border-slate-200 dark:border-slate-700 rounded">
                      <div className="flex justify-between">
                        <span className="text-sm font-medium text-slate-900 dark:text-white">网络波动</span>
                        <span className="text-xs text-slate-500 dark:text-slate-400">10分钟前</span>
                      </div>
                      <div className="text-xs text-slate-500 dark:text-slate-400 mt-1">网络连接暂时中断，已自动恢复</div>
                    </div>
                  </div>
                </div>
                <div>
                  <h5 className="text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">预测分析</h5>
                  <div className="p-2 border border-slate-200 dark:border-slate-700 rounded">
                    <div className="text-sm font-medium text-slate-900 dark:text-white mb-1">设备状态正常，未检测到异常</div>
                    <div className="text-xs text-slate-500 dark:text-slate-400">建议：定期检查设备固件更新</div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}