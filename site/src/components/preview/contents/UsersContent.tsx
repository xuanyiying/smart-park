import { Badge } from '@/components/ui';
import { usersData } from '@/lib/constants';

/**
 * UsersContent - User management tab
 */
export function UsersContent() {
  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-xl font-semibold text-slate-900 dark:text-white">用户权限</h3>
        <button className="px-4 py-2 text-sm btn-primary rounded-lg text-white">添加用户</button>
      </div>
      <div className="card rounded-xl overflow-hidden">
        <table className="w-full text-sm">
          <thead className="bg-slate-50 dark:bg-slate-800">
            <tr>
              <th className="text-left p-3 text-slate-600 dark:text-slate-400">用户</th>
              <th className="text-left p-3 text-slate-600 dark:text-slate-400">角色</th>
              <th className="text-left p-3 text-slate-600 dark:text-slate-400">权限</th>
              <th className="text-left p-3 text-slate-600 dark:text-slate-400">状态</th>
            </tr>
          </thead>
          <tbody>
            {usersData.map((u) => (
              <tr key={u.name} className="border-t border-slate-100 dark:border-slate-700">
                <td className="p-3 font-medium text-slate-900 dark:text-white">{u.name}</td>
                <td className="p-3 text-slate-600 dark:text-slate-400">{u.role}</td>
                <td className="p-3 text-slate-600 dark:text-slate-400">{u.perms}</td>
                <td className="p-3">
                  <Badge variant="success">{u.status}</Badge>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}