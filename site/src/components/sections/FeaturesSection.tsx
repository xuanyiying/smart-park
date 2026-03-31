import { Card } from '@/components/ui';
import { featuresData } from '@/lib/constants';

/**
 * FeaturesSection - Main features showcase
 */
export function FeaturesSection() {
  return (
    <section id="features" className="py-24 px-4 sm:px-6 lg:px-8">
      <div className="max-w-7xl mx-auto">
        <div className="text-center mb-16">
          <h2 className="text-3xl sm:text-4xl font-bold text-slate-900 dark:text-white">
            强大的功能特性
          </h2>
          <p className="mt-4 text-lg text-slate-600 dark:text-slate-400">
            全方位覆盖停车场运营的各个环节
          </p>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {featuresData.map((feature) => (
            <Card key={feature.title} hoverable className="p-6 cursor-pointer">
              <div className="w-12 h-12 rounded-xl icon-bg flex items-center justify-center mb-4">
                <feature.icon className="w-6 h-6 text-white" />
              </div>
              <h3 className="text-xl font-semibold text-slate-900 dark:text-white mb-2">
                {feature.title}
              </h3>
              <p className="text-slate-600 dark:text-slate-400">{feature.description}</p>
            </Card>
          ))}
        </div>
      </div>
    </section>
  );
}