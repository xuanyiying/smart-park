import { CheckCircle } from 'lucide-react';
import { Card } from '@/components/ui';
import { solutionsData } from '@/lib/constants';

/**
 * SolutionsSection - Solution scenarios showcase
 */
export function SolutionsSection() {
  return (
    <section id="solutions" className="py-24 px-4 sm:px-6 lg:px-8 bg-slate-50 dark:bg-slate-900/50">
      <div className="max-w-7xl mx-auto">
        <div className="text-center mb-16">
          <h2 className="text-3xl sm:text-4xl font-bold text-slate-900 dark:text-white">
            适用于多种场景
          </h2>
          <p className="mt-4 text-lg text-slate-600 dark:text-slate-400">
            无论您的停车场规模大小，我们都有合适的解决方案
          </p>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
          {solutionsData.map((solution, index) => (
            <Card
              key={solution.title}
              className={`p-8 relative ${
                index === 1 ? 'ring-2 ring-blue-500 shadow-lg shadow-blue-500/10' : ''
              }`}
            >
              {index === 1 && (
                <div className="absolute top-4 right-4 px-3 py-1 bg-blue-600 rounded-full text-xs font-medium text-white">
                  推荐
                </div>
              )}
              <div className="w-14 h-14 rounded-xl icon-bg flex items-center justify-center mb-6">
                <solution.icon className="w-7 h-7 text-white" />
              </div>
              <h3 className="text-xl font-semibold text-slate-900 dark:text-white mb-2">
                {solution.title}
              </h3>
              <p className="text-slate-600 dark:text-slate-400 mb-6">{solution.description}</p>
              <ul className="space-y-3">
                {solution.features.map((feature) => (
                  <li key={feature} className="flex items-center gap-3">
                    <CheckCircle className="w-5 h-5 text-blue-600 dark:text-blue-400 flex-shrink-0" />
                    <span className="text-slate-700 dark:text-slate-300">{feature}</span>
                  </li>
                ))}
              </ul>
            </Card>
          ))}
        </div>
      </div>
    </section>
  );
}