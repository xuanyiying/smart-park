import Link from 'next/link';
import { ArrowRight, Play, Sparkles, ChevronRight } from 'lucide-react';
import { techStack } from '@/lib/constants';

/**
 * HeroSection - Main landing section with bold visual design
 */
export function HeroSection() {
  return (
    <section className="pt-24 pb-12 px-4 sm:px-6 lg:px-8 bg-gradient-to-b from-indigo-50 via-white to-white dark:from-slate-950 dark:via-slate-900 dark:to-slate-900 relative overflow-hidden">
      {/* 装饰性背景元素 */}
      <div className="absolute inset-0 overflow-hidden pointer-events-none">
        <div className="absolute -top-40 -right-40 w-80 h-80 bg-gradient-to-br from-indigo-300/30 to-purple-300/30 rounded-full blur-3xl" />
        <div className="absolute -bottom-40 -left-40 w-80 h-80 bg-gradient-to-br from-cyan-300/30 to-blue-300/30 rounded-full blur-3xl" />
      </div>

      <div className="max-w-7xl mx-auto relative">
        {/* Badge - 大胆渐变设计 */}
        <div className="flex justify-center mb-8">
          <div className="inline-flex items-center gap-2 px-5 py-2.5 rounded-full bg-gradient-to-r from-indigo-100 to-purple-100 dark:from-indigo-900/40 dark:to-purple-900/40 text-sm border border-indigo-200/50 dark:border-indigo-700/50">
            <Sparkles className="w-4 h-4 text-indigo-600 dark:text-indigo-400" />
            <span className="text-indigo-700 dark:text-indigo-300 font-medium">开源项目 · 微服务架构</span>
            <ChevronRight className="w-4 h-4 text-indigo-400" />
          </div>
        </div>

        {/* Main Heading - 大胆视觉 */}
        <div className="text-center max-w-4xl mx-auto">
          <h1 className="text-5xl sm:text-6xl lg:text-7xl font-bold tracking-tight text-slate-900 dark:text-white">
            智慧停车
            <br />
            <span className="gradient-text">从这里开始</span>
          </h1>
          <p className="mt-6 text-lg sm:text-xl text-slate-600 dark:text-slate-400 max-w-2xl mx-auto leading-relaxed">
            基于 Go + Kratos 微服务架构的新一代智慧停车场管理系统。
            <br className="hidden sm:block" />
            从车辆进场到支付结算，全流程自动化。
          </p>

          {/* CTA Buttons - 大胆设计 */}
          <div className="mt-10 flex flex-col sm:flex-row gap-4 justify-center">
            <Link href="https://github.com/xuanyiying/smart-park" target="_blank" rel="noopener noreferrer">
              <button className="group inline-flex items-center gap-2 px-8 py-4 text-base font-bold btn-primary rounded-2xl text-white shadow-lg shadow-indigo-500/25">
                <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 24 24"><path d="M12 0c-6.626 0-12 5.373-12 12 0 5.302 3.438 9.8 8.207 11.387.599.111.793-.261.793-.577v-2.234c-3.338.726-4.033-1.416-4.033-1.416-.546-1.387-1.333-1.756-1.333-1.756-1.089-.745.083-.729.083-.729 1.205.084 1.839 1.237 1.839 1.237 1.07 1.834 2.807 1.304 3.492.997.107-.775.418-1.305.762-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z"/></svg>
                GitHub 查看
                <ArrowRight className="w-5 h-5 group-hover:translate-x-1 transition-transform" />
              </button>
            </Link>
            <Link href="#dashboard">
              <button className="inline-flex items-center gap-2 px-8 py-4 text-base font-semibold border-2 border-indigo-200 dark:border-indigo-800 rounded-2xl text-indigo-700 dark:text-indigo-300 hover:bg-indigo-50 dark:hover:bg-indigo-900/30 transition-all hover:border-indigo-400">
                <Play className="w-5 h-5" />
                查看演示
              </button>
            </Link>
          </div>

          {/* Tech Stack - 渐变标签 */}
          <div className="mt-12 flex flex-wrap justify-center gap-3">
            {techStack.map((tech, index) => (
              <span
                key={tech}
                className="px-4 py-2 text-sm font-medium text-slate-700 dark:text-slate-300 rounded-xl bg-white/80 dark:bg-slate-800/80 border border-slate-200/50 dark:border-slate-700/50 shadow-sm hover:shadow-md hover:border-indigo-300 dark:hover:border-indigo-600 transition-all cursor-default"
                style={{ animationDelay: `${index * 50}ms` }}
              >
                {tech}
              </span>
            ))}
          </div>
        </div>
      </div>
    </section>
  );
}