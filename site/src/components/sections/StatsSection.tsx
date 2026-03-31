import { statsData } from '@/lib/constants';

/**
 * StatsSection - Statistics showcase with bold gradient cards
 */
export function StatsSection() {
  return (
    <section className="py-20 px-4 sm:px-6 lg:px-8 bg-gradient-to-r from-indigo-50/50 via-purple-50/50 to-cyan-50/50 dark:from-slate-900 dark:via-slate-900 dark:to-slate-900">
      <div className="max-w-7xl mx-auto">
        <div className="grid grid-cols-2 lg:grid-cols-4 gap-6">
          {statsData.map((stat, index) => (
            <div
              key={stat.label}
              className="group relative p-6 rounded-2xl bg-white dark:bg-slate-900 border border-slate-200/50 dark:border-slate-800 hover:border-indigo-300 dark:hover:border-indigo-600 transition-all duration-300 hover:-translate-y-1 shadow-sm hover:shadow-xl"
            >
              {/* 装饰性渐变光晕 */}
              <div className="absolute inset-0 rounded-2xl bg-gradient-to-br from-indigo-50/50 to-purple-50/50 opacity-0 group-hover:opacity-100 transition-opacity" />

              <div className="relative text-center">
                <div className="text-4xl sm:text-5xl font-bold gradient-text mb-2">
                  {stat.value}
                </div>
                <div className="text-sm font-medium text-slate-500 dark:text-slate-400">
                  {stat.label}
                </div>
              </div>

              {/* 底部渐变线条 */}
              <div className="absolute bottom-0 left-1/2 -translate-x-1/2 w-0 group-hover:w-full h-1 bg-gradient-to-r from-indigo-500 via-purple-500 to-pink-500 rounded-full transition-all duration-300" />
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}