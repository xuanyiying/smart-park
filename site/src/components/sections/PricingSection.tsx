import Link from 'next/link';
import { CheckCircle, Star } from 'lucide-react';
import { Button } from '@/components/ui';
import { pricingData } from '@/lib/constants';

/**
 * PricingSection - Bold gradient pricing cards
 */
export function PricingSection() {
  return (
    <section id="pricing" className="py-24 px-4 sm:px-6 lg:px-8 bg-gradient-to-b from-slate-50 via-indigo-50/30 to-slate-50 dark:from-slate-900 dark:via-slate-900 dark:to-slate-900">
      <div className="max-w-7xl mx-auto">
        <div className="text-center mb-16">
          <h2 className="text-4xl sm:text-5xl font-bold text-slate-900 dark:text-white">
            灵活的价格方案
          </h2>
          <p className="mt-4 text-lg text-slate-600 dark:text-slate-400 max-w-2xl mx-auto">
            按需选择，透明定价，无隐藏费用
          </p>
          <div className="mt-4 w-24 h-1 mx-auto bg-gradient-to-r from-indigo-500 via-purple-500 to-pink-500 rounded-full" />
        </div>

        <div className="grid grid-cols-1 md:grid-cols-3 gap-8 max-w-5xl mx-auto">
          {pricingData.map((plan, index) => (
            <div
              key={plan.name}
              className={`group relative rounded-2xl p-8 bg-white dark:bg-slate-800 border border-slate-200 dark:border-slate-700 hover:border-indigo-300 dark:hover:border-indigo-600 transition-all duration-300 hover:-translate-y-2 ${
                plan.highlighted
                  ? 'ring-2 ring-indigo-500 shadow-xl shadow-indigo-500/20 scale-105'
                  : 'shadow-lg hover:shadow-xl'
              }`}
            >
              {/* 渐变背景装饰 */}
              {plan.highlighted && (
                <div className="absolute inset-0 rounded-2xl bg-gradient-to-br from-indigo-500/5 via-purple-500/5 to-pink-500/5" />
              )}

              {plan.highlighted && (
                <div className="absolute -top-4 left-1/2 -translate-x-1/2 px-5 py-1.5 bg-gradient-to-r from-indigo-500 via-purple-500 to-pink-500 rounded-full text-sm font-semibold text-white shadow-lg shadow-indigo-500/30 flex items-center gap-1">
                  <Star className="w-3.5 h-3.5 fill-white" />
                  最受欢迎
                </div>
              )}

              <div className="relative">
                <h3 className="text-xl font-bold text-slate-900 dark:text-white group-hover:text-indigo-600 dark:group-hover:text-indigo-400 transition-colors">
                  {plan.name}
                </h3>
                <p className="mt-2 text-sm text-slate-500 dark:text-slate-400">
                  {plan.description}
                </p>

                <div className="mt-6">
                  <span className="text-4xl font-bold bg-gradient-to-r from-indigo-600 to-purple-600 dark:from-indigo-400 dark:to-purple-400 bg-clip-text text-transparent">
                    {plan.price}
                  </span>
                  <span className="text-slate-500 dark:text-slate-400 ml-1">{plan.period}</span>
                </div>

                <ul className="mt-8 space-y-4">
                  {plan.features.map((feature) => (
                    <li key={feature} className="flex items-center gap-3">
                      <div className="w-5 h-5 rounded-full bg-gradient-to-br from-indigo-500 to-purple-500 flex items-center justify-center flex-shrink-0 shadow-sm">
                        <CheckCircle className="w-3 h-3 text-white" />
                      </div>
                      <span className="text-slate-700 dark:text-slate-300">{feature}</span>
                    </li>
                  ))}
                </ul>

                <Link
                  href="https://github.com/yiying/smart-park"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="block mt-8"
                >
                  {plan.highlighted ? (
                    <Button variant="primary" className="w-full py-3.5 text-base font-semibold rounded-xl">
                      {plan.cta}
                    </Button>
                  ) : (
                    <Button variant="secondary" className="w-full py-3.5 text-base font-semibold rounded-xl border-2 border-indigo-200 dark:border-indigo-800 hover:border-indigo-400">
                      {plan.cta}
                    </Button>
                  )}
                </Link>
              </div>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}