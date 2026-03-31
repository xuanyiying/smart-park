"use client";

import { useState } from 'react';
import Image from 'next/image';
import { sidebarItems } from '@/lib/constants';
import {
  DashboardContent,
  ParkingLotsContent,
  VehiclesContent,
  OrdersContent,
  BillingContent,
  DevicesContent,
  UsersContent,
  ReportsContent,
} from './contents';

type TabId = typeof sidebarItems[number]['id'];

/**
 * DashboardPreview - Main dashboard preview with sidebar navigation
 */
export function DashboardPreview() {
  const [activeTab, setActiveTab] = useState<TabId>('dashboard');

  const renderContent = () => {
    switch (activeTab) {
      case 'dashboard':
        return <DashboardContent />;
      case 'parking-lots':
        return <ParkingLotsContent />;
      case 'vehicles':
        return <VehiclesContent />;
      case 'orders':
        return <OrdersContent />;
      case 'billing':
        return <BillingContent />;
      case 'devices':
        return <DevicesContent />;
      case 'users':
        return <UsersContent />;
      case 'reports':
        return <ReportsContent />;
      default:
        return <DashboardContent />;
    }
  };

  return (
    <section id="dashboard" className="py-16 px-4 sm:px-6 lg:px-8">
      <div className="max-w-7xl mx-auto">
        <div className="text-center mb-12">
          <h2 className="text-3xl sm:text-4xl font-bold text-slate-900 dark:text-white">
            管理后台预览
          </h2>
          <p className="mt-4 text-lg text-slate-600 dark:text-slate-400">
            专业、直观、功能强大的管理界面
          </p>
        </div>

        {/* Dashboard Preview */}
        <div className="card rounded-2xl overflow-hidden shadow-xl">
          {/* Top Bar */}
          <div className="bg-slate-100 dark:bg-slate-800 px-4 py-3 flex items-center gap-2">
            <div className="flex gap-1.5">
              <div className="w-3 h-3 rounded-full bg-red-400"></div>
              <div className="w-3 h-3 rounded-full bg-yellow-400"></div>
              <div className="w-3 h-3 rounded-full bg-green-400"></div>
            </div>
            <div className="flex-1 text-center text-sm text-slate-500 dark:text-slate-400">
              Smart Park 管理后台
            </div>
          </div>

          <div className="flex">
            {/* Sidebar */}
            <div className="hidden md:block w-56 bg-slate-50 dark:bg-slate-800/50 border-r border-slate-200 dark:border-slate-700 min-h-[500px]">
              <div className="p-4 border-b border-slate-200 dark:border-slate-700">
                <div className="flex items-center gap-2">
                  <Image src="/logo-icon.svg" alt="Smart Park" width={28} height={28} />
                  <span className="font-semibold text-slate-900 dark:text-white">Smart Park</span>
                </div>
              </div>
              <nav className="p-2">
                {sidebarItems.map((item) => (
                  <button
                    key={item.id}
                    onClick={() => setActiveTab(item.id)}
                    className={`sidebar-item w-full flex items-center gap-3 px-3 py-2 rounded-lg text-sm text-left ${
                      activeTab === item.id
                        ? 'active'
                        : 'text-slate-600 dark:text-slate-400'
                    }`}
                  >
                    <item.icon className="w-5 h-5" />
                    {item.label}
                  </button>
                ))}
              </nav>
            </div>

            {/* Main Content */}
            <div className="flex-1 p-6 dashboard-preview min-h-[500px]">
              {renderContent()}
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}