import { CreditCard } from 'lucide-react';
import { Badge } from '@/components/ui';
import { billingRulesData } from '@/lib/constants';

/**
 * BillingContent - Billing rules management tab
 */
export function BillingContent() {
  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-xl font-semibold text-slate-900 dark:text-white">计费规则</h3>
        <button className="px-4 py-2 text-sm btn-primary rounded-lg text-white">添加规则</button>
      </div>
      <div className="grid gap-4">
        {billingRulesData.map((rule) => (
          <div key={rule.name} className="card rounded-xl p-4 flex items-center justify-between">
            <div className="flex items-center gap-4">
              <div className="w-10 h-10 rounded-lg bg-blue-100 dark:bg-blue-900/30 flex items-center justify-center">
                <CreditCard className="w-5 h-5 text-blue-600 dark:text-blue-400" />
              </div>
              <div>
                <div className="font-medium text-slate-900 dark:text-white">{rule.name}</div>
                <div className="text-sm text-slate-500 dark:text-slate-400">{rule.rule}</div>
              </div>
            </div>
            <Badge variant="success">{rule.status}</Badge>
          </div>
        ))}
      </div>
    </div>
  );
}