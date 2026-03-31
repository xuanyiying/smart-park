import { Card } from '@/components/ui';
import { featuresData } from '@/lib/constants';

/**
 * FeaturesSection - Main features showcase with bold gradient design
 */
export function FeaturesSection() {
  return (
    <section id="features" className="py-24 px-4 sm:px-6 lg:px-8 relative">
      {/* 背景装饰 */}
      <div className="absolute inset-0 bg-gradient-to-b from-white via-indigo-50/30 to-white dark:from-slate-900 dark:via-slate-900 dark:to-slate-900 -z-10" />

      <div className="max-w-7xl mx-auto">
        <div className="text-center mb-16">
          <h2 className="text-4xl sm:text-5xl font-bold text-slate-900 dark:text-white">
            强大的功能特性
          </h2>
          <p className="mt-4 text-lg text-slate-600 dark:text-slate-400 max-w-2xl mx-auto">
            全方位覆盖停车场运营的各个环节
          </p>
          {/* 装饰线条 */}
          <div className="mt-4 w-24 h-1 mx-auto bg-gradient-to-r from-indigo-500 via-purple-500 to-pink-500 rounded-full" />
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-8">
          {featuresData.map((feature, index) => (
            <Card
              key={feature.title}
              hoverable
              className="p-8 cursor-pointer group relative overflow-hidden"
              style={{ animationDelay: `${index * 100}ms` }}
            >
              {/* 背景渐变 */}
              <div className="absolute inset-0 bg-gradient-to-br from-indigo-50/50 via-purple-50/30 to-pink-50/30 dark:from-indigo-950/20 dark:via-purple-950/20 dark:to-pink-950/20 opacity-0 group-hover:opacity-100 transition-opacity" />

              <div className="relative">
                {/* 图标 - 渐变背景 */}
                <div className="w-14 h-14 rounded-2xl bg-gradient-to-br from-indigo-500 via-purple-500 to-pink-500 flex items-center justify-center mb-5 shadow-lg shadow-indigo-500/25 group-hover:shadow-indigo-500/40 transition-shadow">
                  <feature.icon className="w-7 h-7 text-white" />
                </div>

                <h3 className="text-xl font-bold text-slate-900 dark:text-white mb-3 group-hover:text-indigo-600 dark:group-hover:text-indigo-400 transition-colors">
                  {feature.title}
                </h3>
                <p className="text-slate-600 dark:text-slate-400 leading-relaxed">
                  {feature.description}
                </p>
              </div>

              {/* 右侧箭头指示器 */}
              <div className="absolute top-8 right-6 opacity-0 group-hover:opacity-100 transform translate-x-2 group-hover:translate-x-0 transition-all">
                <svg className="w-5 h-5 text-indigo-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 8l4 4m0 0l-4 4m4-4H3" />
                </svg>
              </div>
            </Card>
          ))}
        </div>
      </div>
    </section>
  );
}