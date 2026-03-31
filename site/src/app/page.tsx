"use client";

import { useDarkMode } from '@/hooks';
import { Header, Footer } from '@/components/layout';
import {
  HeroSection,
  StatsSection,
  FeaturesSection,
  SolutionsSection,
  CasesSection,
  PricingSection,
  CTASection,
  ContactSection,
} from '@/components/sections';
import { DashboardPreview } from '@/components/preview';

export default function HomePage() {
  const { darkMode, setDarkMode } = useDarkMode();

  return (
    <div className="min-h-screen bg-white dark:bg-slate-900 text-slate-900 dark:text-slate-100 transition-colors">
      {/* Background */}
      <div className="fixed inset-0 grid-bg pointer-events-none" />

      <Header darkMode={darkMode} setDarkMode={setDarkMode} />
      <main className="relative">
        <HeroSection />
        <DashboardPreview />
        <StatsSection />
        <FeaturesSection />
        <SolutionsSection />
        <CasesSection />
        <PricingSection />
        <CTASection />
        <ContactSection />
      </main>
      <Footer />
    </div>
  );
}