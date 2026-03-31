import Link from 'next/link';
import { MapPin, Quote, Users, Building2, ArrowRight } from 'lucide-react';
import { Card } from '@/components/ui';
import { casesData } from '@/lib/constants';

/**
 * CasesSection - Customer cases with bold visual design
 */
export function CasesSection() {
  return (
    <section id="cases" className="py-24 px-4 sm:px-6 lg:px-8 bg-gradient-to-b from-white via-indigo-50/20 to-white dark:from-slate-950 dark:via-slate-900 dark:to-slate-950">
      <div className="max-w-7xl mx-auto">
        <div className="text-center mb-16">
          <h2 className="text-4xl sm:text-5xl font-bold text-slate-900 dark:text-white">
            客户案例
          </h2>
          <p className="mt-4 text-lg text-slate-600 dark:text-slate-400 max-w-2xl mx-auto">
            他们选择了 Smart Park，并取得了显著成效
          </p>
          <div className="mt-4 w-24 h-1 mx-auto bg-gradient-to-r from-indigo-500 via-purple-500 to-pink-500 rounded-full" />
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
          {casesData.map((caseItem, index) => (
            <Card key={caseItem.company} className="p-6 group relative overflow-hidden">
              {/* 渐变装饰 */}
              <div className="absolute top-0 left-0 w-full h-1 bg-gradient-to-r from-indigo-500 via-purple-500 to-pink-500" />

              {/* 背景悬浮效果 */}
              <div className="absolute inset-0 bg-gradient-to-br from-indigo-50/50 to-purple-50/50 dark:from-indigo-950/20 dark:to-purple-950/20 opacity-0 group-hover:opacity-100 transition-opacity" />

              <div className="relative">
                {/* Header */}
                <div className="flex items-start justify-between mb-6">
                  <div>
                    <h3 className="text-xl font-bold text-slate-900 dark:text-white group-hover:text-indigo-600 dark:group-hover:text-indigo-400 transition-colors">
                      {caseItem.company}
                    </h3>
                    <div className="flex items-center gap-2 mt-1 text-sm text-slate-500 dark:text-slate-400">
                      <span>{caseItem.industry}</span>
                      <span>·</span>
                      <MapPin className="w-3 h-3" />
                      <span>{caseItem.location}</span>
                    </div>
                  </div>
                  <div className="w-12 h-12 rounded-xl bg-gradient-to-br from-indigo-500 to-purple-500 flex items-center justify-center shadow-lg shadow-indigo-500/25">
                    <Building2 className="w-6 h-6 text-white" />
                  </div>
                </div>

                {/* Stats - 渐变背景 */}
                <div className="grid grid-cols-3 gap-4 mb-6 p-4 bg-gradient-to-br from-slate-50 to-indigo-50 dark:from-slate-800/50 dark:to-indigo-950/30 rounded-xl">
                  {caseItem.stats.map((stat) => (
                    <div key={stat.label} className="text-center">
                      <div className="text-lg font-bold gradient-text">{stat.value}</div>
                      <div className="text-xs text-slate-500 dark:text-slate-400">{stat.label}</div>
                    </div>
                  ))}
                </div>

                {/* Quote */}
                <div className="relative">
                  <Quote className="absolute -top-2 -left-1 w-6 h-6 text-indigo-200 dark:text-indigo-800" />
                  <p className="text-slate-600 dark:text-slate-400 text-sm leading-relaxed pl-6">
                    {caseItem.quote}
                  </p>
                </div>

                {/* Author */}
                <div className="mt-6 pt-4 border-t border-slate-200 dark:border-slate-700 flex items-center gap-3">
                  <div className="w-10 h-10 rounded-full bg-gradient-to-br from-indigo-500 to-purple-500 flex items-center justify-center shadow-md">
                    <Users className="w-5 h-5 text-white" />
                  </div>
                  <div>
                    <div className="text-sm font-semibold text-slate-900 dark:text-white">
                      {caseItem.author}
                    </div>
                    <div className="text-xs text-slate-500 dark:text-slate-400">
                      {caseItem.position}
                    </div>
                  </div>
                </div>
              </div>
            </Card>
          ))}
        </div>

        {/* More cases CTA */}
        <div className="mt-12 text-center">
          <Link href="#contact">
            <button className="inline-flex items-center gap-2 text-indigo-600 dark:text-indigo-400 hover:text-indigo-700 dark:hover:text-indigo-300 transition-colors font-semibold group">
              查看更多案例
              <ArrowRight className="w-4 h-4 group-hover:translate-x-1 transition-transform" />
            </button>
          </Link>
        </div>
      </div>
    </section>
  );
}