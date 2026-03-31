import Link from 'next/link';
import { ArrowRight, Sparkles } from 'lucide-react';
import { ctaStats } from '@/lib/constants';

/**
 * CTASection - Refined gradient call to action matching app colors
 */
export function CTASection() {
  return (
    <section className="py-24 px-4 sm:px-6 lg:px-8 bg-gradient-to-br from-slate-900 via-indigo-950 to-slate-900 relative overflow-hidden">
      {/* 装饰性背景 - 使用应用主色 */}
      <div className="absolute inset-0 overflow-hidden pointer-events-none">
        <div className="absolute top-0 left-1/4 w-[500px] h-[500px] bg-indigo-600/20 rounded-full blur-[100px]" />
        <div className="absolute bottom-0 right-1/4 w-[500px] h-[500px] bg-purple-600/20 rounded-full blur-[100px]" />
        <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 w-[800px] h-[800px] bg-indigo-500/10 rounded-full blur-[120px]" />
      </div>

      {/* 装饰线条 */}
      <div className="absolute top-0 left-0 right-0 h-px bg-gradient-to-r from-transparent via-indigo-500/50 to-transparent" />
      <div className="absolute bottom-0 left-0 right-0 h-px bg-gradient-to-r from-transparent via-purple-500/50 to-transparent" />

      <div className="max-w-4xl mx-auto text-center relative">
        <h2 className="text-4xl sm:text-5xl font-bold text-white">
          准备好开始了吗？
        </h2>
        <p className="mt-4 text-lg text-slate-300 max-w-xl mx-auto">
          立即访问 GitHub，开始使用 Smart Park 构建您的智慧停车场系统
        </p>

        <div className="mt-10 flex flex-col sm:flex-row gap-4 justify-center">
          <Link href="https://github.com/xuanyiying/smart-park" target="_blank" rel="noopener noreferrer">
            <button className="inline-flex items-center gap-2 px-8 py-4 bg-gradient-to-r from-indigo-500 via-purple-500 to-indigo-500 text-white font-bold rounded-2xl hover:opacity-90 transition-all hover:scale-105 shadow-lg shadow-indigo-500/25">
              <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 24 24"><path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z"/></svg>
              GitHub 仓库
              <ArrowRight className="w-5 h-5" />
            </button>
          </Link>
          <Link href="#contact">
            <button className="inline-flex items-center gap-2 px-8 py-4 border-2 border-slate-600 text-slate-200 font-semibold rounded-2xl hover:bg-slate-800 transition-all">
              <Sparkles className="w-5 h-5" />
              预约演示
            </button>
          </Link>
        </div>

        {/* Trust badges - 玻璃态卡片 */}
        <div className="mt-16 flex flex-wrap justify-center gap-6">
          {ctaStats.map((stat) => (
            <div
              key={stat.label}
              className="px-6 py-4 bg-white/5 backdrop-blur-sm rounded-2xl border border-white/10 hover:bg-white/10 transition-all"
            >
              <div className="text-2xl font-bold bg-gradient-to-r from-indigo-400 to-purple-400 bg-clip-text text-transparent">
                {stat.value}
              </div>
              <div className="text-sm text-slate-400">{stat.label}</div>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}