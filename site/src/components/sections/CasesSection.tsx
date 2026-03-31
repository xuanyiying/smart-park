import Link from 'next/link';
import { MapPin, Quote, Users, Building2, ArrowRight } from 'lucide-react';
import { Card } from '@/components/ui';
import { casesData } from '@/lib/constants';

/**
 * CasesSection - Customer cases showcase
 */
export function CasesSection() {
  return (
    <section id="cases" className="py-24 px-4 sm:px-6 lg:px-8">
      <div className="max-w-7xl mx-auto">
        <div className="text-center mb-16">
          <h2 className="text-3xl sm:text-4xl font-bold text-slate-900 dark:text-white">
            客户案例
          </h2>
          <p className="mt-4 text-lg text-slate-600 dark:text-slate-400">
            他们选择了 Smart Park，并取得了显著成效
          </p>
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
          {casesData.map((caseItem) => (
            <Card key={caseItem.company} className="p-6">
              {/* Header */}
              <div className="flex items-start justify-between mb-6">
                <div>
                  <h3 className="text-xl font-bold text-slate-900 dark:text-white">{caseItem.company}</h3>
                  <div className="flex items-center gap-2 mt-1 text-sm text-slate-500 dark:text-slate-400">
                    <span>{caseItem.industry}</span>
                    <span>·</span>
                    <MapPin className="w-3 h-3" />
                    <span>{caseItem.location}</span>
                  </div>
                </div>
                <div className="w-12 h-12 rounded-xl bg-blue-100 dark:bg-blue-900/30 flex items-center justify-center">
                  <Building2 className="w-6 h-6 text-blue-600 dark:text-blue-400" />
                </div>
              </div>

              {/* Stats */}
              <div className="grid grid-cols-3 gap-4 mb-6 p-4 bg-slate-50 dark:bg-slate-800/50 rounded-xl">
                {caseItem.stats.map((stat) => (
                  <div key={stat.label} className="text-center">
                    <div className="text-lg font-bold text-slate-900 dark:text-white">{stat.value}</div>
                    <div className="text-xs text-slate-500 dark:text-slate-400">{stat.label}</div>
                  </div>
                ))}
              </div>

              {/* Quote */}
              <div className="relative">
                <Quote className="absolute -top-2 -left-1 w-6 h-6 text-blue-200 dark:text-blue-800" />
                <p className="text-slate-600 dark:text-slate-400 text-sm leading-relaxed pl-6">
                  {caseItem.quote}
                </p>
              </div>

              {/* Author */}
              <div className="mt-6 pt-4 border-t border-slate-200 dark:border-slate-700 flex items-center gap-3">
                <div className="w-10 h-10 rounded-full bg-blue-100 dark:bg-blue-900/30 flex items-center justify-center">
                  <Users className="w-5 h-5 text-blue-600 dark:text-blue-400" />
                </div>
                <div>
                  <div className="text-sm font-medium text-slate-900 dark:text-white">{caseItem.author}</div>
                  <div className="text-xs text-slate-500 dark:text-slate-400">{caseItem.position}</div>
                </div>
              </div>
            </Card>
          ))}
        </div>

        {/* More cases CTA */}
        <div className="mt-12 text-center">
          <Link href="#contact">
            <button className="inline-flex items-center gap-2 text-blue-600 dark:text-blue-400 hover:text-blue-700 dark:hover:text-blue-300 transition-colors font-medium">
              查看更多案例
              <ArrowRight className="w-4 h-4" />
            </button>
          </Link>
        </div>
      </div>
    </section>
  );
}