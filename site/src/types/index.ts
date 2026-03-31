/**
 * TypeScript type definitions for the application
 */

import type { LucideIcon } from 'lucide-react';

// Navigation
export interface NavItem {
  label: string;
  href: string;
}

// Sidebar item for dashboard
export interface SidebarItem {
  id: string;
  icon: LucideIcon;
  label: string;
}

// Stats item
export interface StatsItem {
  value: string;
  label: string;
}

// Feature item
export interface FeatureItem {
  icon: LucideIcon;
  title: string;
  description: string;
}

// Solution item
export interface SolutionItem {
  icon: LucideIcon;
  title: string;
  description: string;
  features: readonly string[];
}

// Case item
export interface CaseItem {
  company: string;
  industry: string;
  location: string;
  stats: Array<{ label: string; value: string }>;
  quote: string;
  author: string;
  position: string;
}

// Pricing plan
export interface PricingPlan {
  name: string;
  price: string;
  period: string;
  description: string;
  features: readonly string[];
  cta: string;
  highlighted?: boolean;
}

// Contact info
export interface ContactInfo {
  title: string;
  value: string;
}

// Footer links section
export interface FooterSection {
  title: string;
  links: readonly string[];
}

// Dashboard stat
export interface DashboardStat {
  label: string;
  value: string;
  change: string;
  positive: boolean;
  icon: LucideIcon;
}

// Parking lot
export interface ParkingLot {
  name: string;
  address: string;
  spaces: number;
  used: number;
}

// Vehicle
export interface Vehicle {
  plate: string;
  type: string;
  time: string;
  lot: string;
  status: string;
}

// Order
export interface Order {
  id: string;
  plate: string;
  amount: string;
  method: string;
  status: string;
}

// Billing rule
export interface BillingRule {
  name: string;
  type: string;
  rule: string;
  status: string;
}

// Device
export interface Device {
  name: string;
  type: string;
  status: string;
  location: string;
}

// User
export interface User {
  name: string;
  role: string;
  perms: string;
  status: string;
}

// Report stat
export interface ReportStat {
  label: string;
  value: string;
}

// Report chart data
export interface ChartData {
  month: string;
  value: number;
}

// Button variants
export type ButtonVariant = 'primary' | 'secondary' | 'ghost';
export type ButtonSize = 'sm' | 'md' | 'lg';

// Input props
export interface InputProps extends React.InputHTMLAttributes<HTMLInputElement> {
  label?: string;
  error?: string;
}

// Card props
export interface CardProps extends React.HTMLAttributes<HTMLDivElement> {
  hoverable?: boolean;
}

// Badge variant
export type BadgeVariant = 'default' | 'success' | 'warning' | 'error' | 'info';