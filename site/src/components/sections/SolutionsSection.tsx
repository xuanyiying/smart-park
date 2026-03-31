import { CheckCircle } from 'lucide-react';
import { Card } from '@/components/ui';
import { solutionsData } from '@/lib/constants';

/**
 * SolutionsSection - Solution scenarios with bold gradient cards
 */
export function SolutionsSection() {
  return (
    <section id="solutions" className="py-24 px-4 sm:px-6 lg:px-8 bg-gradient-to-b from-slate-50 to-white dark:from-slate-900 dark:to-slate-950">
      <div className="max-w-7xl mx-auto">
        <div className="text-center mb-16">
          <h2 className="text-4xl sm:text-5xl font-bold text-slate-900 dark:text-white">
            适用于多种场景
          </h2>
          <p className="mt-4 text-lg text-slate-600 dark:text-slate-400 max-w-2xl mx-auto">
            无论您的停车场规模大小，我们都有合适的解决方案
          </p>
          <div className="mt-4 w-24 h-1 mx-auto bg-gradient-to-r from-indigo-500 via-purple-500 to-pink-500 rounded-full" />
        </div>

        <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
          {solutionsData.map((solution, index) => (
            <Card
              key={solution.title}
              className={`p-8 relative group overflow-hidden ${
                index === 1 ? 'ring-2 ring-indigo-500 shadow-xl shadow-indigo-500/15' : ''
              }`}
            >
              {/* 悬浮时渐变背景 */}
              <div className="absolute inset-0 bg-gradient-to-br from-indigo-50 via-purple-50 to-pink-50 dark:from-indigo-950/30 dark:via-purple-950/30 dark:to-pink-950/30 opacity-0 group-hover:opacity-100 transition-opacity" />

              {index === 1 && (
                <div className="absolute top-4 right-4 px-3 py-1 bg-gradient-to-r from-indigo-500 to-purple-500 rounded-full text-xs font-medium text-white shadow-md">
                  推荐
                </div>
              )}

              <div className="relative">
                {/* 图标 - 渐变背景 */}
                <div className="w-14 h-14 rounded-2xl bg-gradient-to-br from-indigo-500 via-purple-500 to-pink-500 flex items-center justify-center mb-6 shadow-lg">
                  <solution.icon className="w-7 h-7 text-white" />
                </div>

                <h3 className="text-xl font-bold text-slate-900 dark:text-white mb-3 group-hover:text-indigo-600 dark:group-hover:text-indigo-400 transition-colors">
                  {solution.title}
                </h3>
                <p className="text-slate-600 dark:text-slate-400 mb-6 leading-relaxed">
                  {solution.description}
                </p>

                <ul className="space-y-3">
                  {solution.features.map((feature, i) => (
                    <li key={feature} className="flex items-center gap-3">
                      <div className="w-5 h-5 rounded-full bg-gradient-to-br from-indigo-500 to-purple-500 flex items-center justify-center flex-shrink-0">
                        <CheckCircle className="w-3 h-3 text-white" />
                      </div>
                      <span className="text-slate-700 dark:text-slate-300">{feature}</span>
                    </li>
                  ))}
                </ul>
              </div>
            </Card>
          ))}
        </div>
      </div>
    </section>
  );
}