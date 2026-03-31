import { Clock, Mail, MessageSquare } from 'lucide-react';
import { Card, Input, Button } from '@/components/ui';
import { contactInfo } from '@/lib/constants';

/**
 * ContactSection - Contact form with bold gradient design
 */
export function ContactSection() {
  return (
    <section id="contact" className="py-24 px-4 sm:px-6 lg:px-8 bg-gradient-to-b from-white via-indigo-50/20 to-white dark:from-slate-950 dark:via-slate-900 dark:to-slate-950">
      <div className="max-w-7xl mx-auto">
        <div className="text-center mb-16">
          <h2 className="text-4xl sm:text-5xl font-bold text-slate-900 dark:text-white">
            联系我们
          </h2>
          <p className="mt-4 text-lg text-slate-600 dark:text-slate-400 max-w-2xl mx-auto">
            有任何问题或需求？我们的团队随时为您服务。
          </p>
          <div className="mt-4 w-24 h-1 mx-auto bg-gradient-to-r from-indigo-500 via-purple-500 to-pink-500 rounded-full" />
        </div>

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-12 max-w-6xl mx-auto">
          <div className="flex flex-col justify-center">
            <div className="space-y-8">
              {contactInfo.map((item, index) => {
                const icons = [Clock, Mail, MessageSquare];
                const Icon = icons[index];
                return (
                  <div
                    key={item.title}
                    className="flex items-start gap-5 p-6 rounded-2xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 hover:border-indigo-300 dark:hover:border-indigo-600 transition-all hover:shadow-lg hover:shadow-indigo-500/10 group"
                  >
                    <div className="w-14 h-14 rounded-2xl bg-gradient-to-br from-indigo-500 via-purple-500 to-pink-500 flex items-center justify-center flex-shrink-0 shadow-lg shadow-indigo-500/25 group-hover:shadow-indigo-500/40 transition-shadow">
                      <Icon className="w-6 h-6 text-white" />
                    </div>
                    <div>
                      <h4 className="font-bold text-slate-900 dark:text-white group-hover:text-indigo-600 dark:group-hover:text-indigo-400 transition-colors">
                        {item.title}
                      </h4>
                      <p className="text-slate-600 dark:text-slate-400 mt-1">{item.value}</p>
                    </div>
                  </div>
                );
              })}
            </div>
          </div>

          <Card className="p-8 relative overflow-hidden group">
            {/* 渐变装饰 */}
            <div className="absolute top-0 right-0 w-32 h-32 bg-gradient-to-br from-indigo-200 to-purple-200 dark:from-indigo-900/30 dark:to-purple-900/30 rounded-full blur-2xl -translate-y-1/2 translate-x-1/2" />
            <div className="absolute bottom-0 left-0 w-32 h-32 bg-gradient-to-br from-pink-200 to-indigo-200 dark:from-pink-900/30 dark:to-indigo-900/30 rounded-full blur-2xl translate-y-1/2 -translate-x-1/2" />

            <form className="space-y-6 relative">
              <Input
                label="姓名"
                placeholder="请输入您的姓名"
              />
              <Input
                label="邮箱"
                type="email"
                placeholder="请输入您的邮箱"
              />
              <div>
                <label htmlFor="message" className="block text-sm font-semibold text-slate-700 dark:text-slate-300 mb-2">
                  留言
                </label>
                <textarea
                  id="message"
                  rows={4}
                  className="w-full px-4 py-3 rounded-xl border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-800 text-slate-900 dark:text-white placeholder-slate-400 focus:ring-2 focus:ring-indigo-500 focus:border-transparent outline-none resize-none transition-all"
                  placeholder="请输入您的留言"
                />
              </div>
              <Button type="submit" variant="primary" className="w-full py-3.5 text-base font-semibold rounded-xl">
                提交留言
              </Button>
            </form>
          </Card>
        </div>
      </div>
    </section>
  );
}