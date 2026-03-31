import { Clock, Shield, Users } from 'lucide-react';
import { Card, Input, Button } from '@/components/ui';
import { contactInfo } from '@/lib/constants';

/**
 * ContactSection - Contact form and info section
 */
export function ContactSection() {
  return (
    <section id="contact" className="py-24 px-4 sm:px-6 lg:px-8">
      <div className="max-w-7xl mx-auto">
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-12">
          <div>
            <h2 className="text-3xl sm:text-4xl font-bold text-slate-900 dark:text-white">
              联系我们
            </h2>
            <p className="mt-4 text-lg text-slate-600 dark:text-slate-400">
              有任何问题或需求？我们的团队随时为您服务。
            </p>

            <div className="mt-10 space-y-6">
              {contactInfo.map((item, index) => {
                const icons = [Clock, Shield, Users];
                const Icon = icons[index];
                return (
                  <div key={item.title} className="flex items-start gap-4">
                    <div className="w-12 h-12 rounded-xl bg-blue-100 dark:bg-blue-900/30 flex items-center justify-center flex-shrink-0">
                      <Icon className="w-5 h-5 text-blue-600 dark:text-blue-400" />
                    </div>
                    <div>
                      <h4 className="font-semibold text-slate-900 dark:text-white">{item.title}</h4>
                      <p className="text-slate-600 dark:text-slate-400 mt-1">{item.value}</p>
                    </div>
                  </div>
                );
              })}
            </div>
          </div>

          <Card className="p-8">
            <form className="space-y-6">
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
                <label htmlFor="message" className="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">
                  留言
                </label>
                <textarea
                  id="message"
                  rows={4}
                  className="w-full px-4 py-3 rounded-xl border border-slate-300 dark:border-slate-600 bg-white dark:bg-slate-800 text-slate-900 dark:text-white placeholder-slate-400 focus:ring-2 focus:ring-blue-500 focus:border-transparent outline-none resize-none transition-all"
                  placeholder="请输入您的留言"
                />
              </div>
              <Button type="submit" variant="primary" className="w-full">
                提交留言
              </Button>
            </form>
          </Card>
        </div>
      </div>
    </section>
  );
}