import Link from 'next/link';
import { CheckCircle } from 'lucide-react';
import { Button } from '@/components/ui';
import { pricingData } from '@/lib/constants';

/**
 * PricingSection - Pricing plans showcase
 */
export function PricingSection() {
  return (
    <section id="pricing" className="py-24 px-4 sm:px-6 lg:px-8 bg-slate-50 dark:bg-slate-900/50">
      <div className="max-w-7xl mx-auto">
        <div className="text-center mb-16">
          <h2 className="text-3xl sm:text-4xl font-bold text-slate-900 dark:text-white">
            灵活的价格方案
          </h2>
          <p className="mt-4 text-lg text-slate-600 dark:text-slate-400">
            按需选择，透明定价，无隐藏费用
          </p>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-3 gap-8 max-w-5xl mx-auto">
          {pricingData.map((plan) => (
            <div
              key={plan.name}
              className={`card rounded-2xl p-8 relative bg-white dark:bg-slate-800 ${
                plan.highlighted
                  ? 'ring-2 ring-blue-500 shadow-lg shadow-blue-500/10 scale-105'
                  : ''
              }`}
            >
              {plan.highlighted && (
                <div className="absolute -top-4 left-1/2 -translate-x-1/2 px-4 py-1 bg-blue-600 rounded-full text-sm font-medium text-white">
                  最受欢迎
                </div>
              )}
              <h3 className="text-xl font-semibold text-slate-900 dark:text-white">
                {plan.name}
              </h3>
              <p className="mt-2 text-sm text-slate-500 dark:text-slate-400">
                {plan.description}
              </p>
              <div className="mt-6">
                <span className="text-4xl font-bold text-slate-900 dark:text-white">
                  {plan.price}
                </span>
                <span className="text-slate-500 dark:text-slate-400">{plan.period}</span>
              </div>
              <ul className="mt-8 space-y-4">
                {plan.features.map((feature) => (
                  <li key={feature} className="flex items-center gap-3">
                    <CheckCircle className="w-5 h-5 text-blue-600 dark:text-blue-400 flex-shrink-0" />
                    <span className="text-slate-700 dark:text-slate-300">{feature}</span>
                  </li>
                ))}
              </ul>
              <Link href="https://github.com/yiying/smart-park" target="_blank" rel="noopener noreferrer" className="block mt-8">
                <Button
                  variant={plan.highlighted ? 'primary' : 'secondary'}
                  className="w-full"
                >
                  {plan.cta}
                </Button>
              </Link>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}