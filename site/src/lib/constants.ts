/**
 * Constants and static data for the application
 */

import {
  Car,
  CreditCard,
  BarChart3,
  Shield,
  Zap,
  Smartphone,
  Users,
  TrendingUp,
  Building2,
  LayoutDashboard,
  Warehouse,
  FileText,
  Settings,
  DollarSign,
  ArrowUpRight,
  ArrowDownRight,
  Server,
} from 'lucide-react';
import type { LucideIcon } from 'lucide-react';

// Navigation items
export const navItems = [
  { label: '功能特性', href: '#features' },
  { label: '解决方案', href: '#solutions' },
  { label: '客户案例', href: '#cases' },
  { label: '价格', href: '#pricing' },
] as const;

// Tech stack badges
export const techStack = ['Go', 'Kratos', 'PostgreSQL', 'Redis', 'RocketMQ', 'Microservices', 'Docker', 'Kubernetes'] as const;

// Dashboard sidebar items
export const sidebarItems = [
  { id: 'dashboard', icon: LayoutDashboard, label: '数据概览' },
  { id: 'parking-lots', icon: Warehouse, label: '停车场' },
  { id: 'vehicles', icon: Car, label: '车辆管理' },
  { id: 'orders', icon: FileText, label: '订单管理' },
  { id: 'billing', icon: CreditCard, label: '计费规则' },
  { id: 'devices', icon: Settings, label: '设备控制' },
  { id: 'users', icon: Users, label: '用户权限' },
  { id: 'reports', icon: BarChart3, label: '报表统计' },
] as const;

// Stats data
export const statsData = [
  { value: '99.5%', label: '车牌识别率' },
  { value: '10ms', label: '平均响应时间' },
  { value: '1000+', label: '部署站点' },
  { value: '24/7', label: '技术支持' },
] as const;

// Features data
export const featuresData: Array<{
  icon: LucideIcon;
  title: string;
  description: string;
}> = [
  {
    icon: Car,
    title: '车牌识别',
    description: '高清摄像头自动识别，支持多种车牌类型，识别率高达99.5%',
  },
  {
    icon: CreditCard,
    title: '智能计费',
    description: '灵活的计费规则引擎，支持阶梯计费、时段计费、会员优惠',
  },
  {
    icon: Smartphone,
    title: '移动支付',
    description: '集成微信、支付宝等主流支付，支持扫码缴费和无感支付',
  },
  {
    icon: BarChart3,
    title: '数据分析',
    description: '实时数据看板，多维度报表分析，助力运营决策优化',
  },
  {
    icon: Shield,
    title: '安全可靠',
    description: '企业级安全架构，数据加密存储，完善的权限管理体系',
  },
  {
    icon: Zap,
    title: '高性能',
    description: '微服务架构设计，支持高并发场景，系统稳定可靠',
  },
];

// Solutions data
export const solutionsData = [
  {
    icon: Building2,
    title: '商业综合体',
    description: '多入口统一管理，商户优惠券核销，VIP会员管理',
    features: ['多入口统一管理', '商户优惠券核销', 'VIP会员管理', '车位引导系统'],
  },
  {
    icon: Users,
    title: '住宅小区',
    description: '业主车位管理，访客临时停车，月租续费提醒',
    features: ['业主车位管理', '访客临时停车', '月租续费提醒', '车位共享功能'],
  },
  {
    icon: TrendingUp,
    title: '公共停车场',
    description: '无人值守运营，电子发票开具，运营数据分析',
    features: ['无人值守运营', '电子发票开具', '车位实时发布', '运营数据分析'],
  },
] as const;

// Cases data
export const casesData = [
  {
    company: '万达广场',
    industry: '商业综合体',
    location: '北京朝阳',
    stats: [
      { label: '日均停车', value: '3,200+' },
      { label: '车位数量', value: '1,500' },
      { label: '效率提升', value: '45%' },
    ],
    quote: 'Smart Park 帮助我们实现了多入口统一管理，车辆通行效率提升了45%，商户优惠券核销也更加便捷。',
    author: '张经理',
    position: '物业运营总监',
  },
  {
    company: '万科城市之光',
    industry: '住宅小区',
    location: '上海浦东',
    stats: [
      { label: '住户数量', value: '2,800户' },
      { label: '车位数量', value: '2,100' },
      { label: '满意度', value: '98%' },
    ],
    quote: '业主车位管理、访客临时停车、月租续费提醒，一站式解决。业主满意度大幅提升，物业工作量减少一半。',
    author: '李主管',
    position: '物业管理处',
  },
  {
    company: '首都机场停车场',
    industry: '公共停车场',
    location: '北京顺义',
    stats: [
      { label: '日均停车', value: '8,500+' },
      { label: '车位数量', value: '5,000' },
      { label: '无人化率', value: '95%' },
    ],
    quote: '无人值守运营模式让我们的运营成本降低了60%，电子发票开具功能深受旅客好评。',
    author: '王总监',
    position: '停车场运营',
  },
] as const;

// Pricing data
export const pricingData: Array<{
  name: string;
  price: string;
  period: string;
  description: string;
  features: string[];
  cta: string;
  highlighted?: boolean;
}> = [
  {
    name: '基础版',
    price: '免费',
    period: '',
    description: '适合小型停车场试用',
    features: [
      '最多 1 个停车场',
      '最多 2 个出入口',
      '基础计费功能',
      '数据保留 30 天',
      '社区支持',
    ],
    cta: '立即开始',
    highlighted: false,
  },
  {
    name: '专业版',
    price: '¥999',
    period: '/月',
    description: '适合中型停车场运营',
    features: [
      '最多 5 个停车场',
      '不限出入口数量',
      '高级计费规则',
      '数据保留 1 年',
      '移动支付集成',
      '优先技术支持',
    ],
    highlighted: true,
    cta: '开始试用',
  },
  {
    name: '企业版',
    price: '联系我们',
    period: '',
    description: '适合大型连锁停车场',
    features: [
      '不限停车场数量',
      '定制化开发',
      '私有化部署',
      '数据永久保留',
      '专属客户经理',
      'SLA 服务保障',
    ],
    cta: '联系销售',
    highlighted: false,
  },
] as const;

// CTA Stats
export const ctaStats = [
  { value: '1000+', label: '企业用户' },
  { value: '50万+', label: '日均停车' },
  { value: '99.9%', label: '系统可用性' },
] as const;

// Contact info
export const contactInfo = [
  { title: '工作时间', value: '周一至周五 9:00 - 18:00' },
  { title: '技术支持', value: 'support@smartpark.com' },
  { title: '商务合作', value: 'business@smartpark.com' },
] as const;

// Footer links
export const footerLinks = [
  {
    title: '产品',
    links: ['功能特性', '价格方案', '更新日志', 'API 文档'],
  },
  {
    title: '支持',
    links: ['帮助中心', '技术文档', '联系我们', '社区论坛'],
  },
  {
    title: '关于',
    links: ['关于我们', '加入我们', '隐私政策', '服务条款'],
  },
] as const;

// Dashboard stats
export const dashboardStats = [
  { label: '今日收入', value: '¥12,580', change: '+12%', positive: true, icon: DollarSign },
  { label: '今日入场', value: '1,234', change: '+5%', positive: true, icon: ArrowUpRight },
  { label: '今日出场', value: '1,189', change: '-2%', positive: false, icon: ArrowDownRight },
  { label: '在线设备', value: '48', change: '100%', positive: true, icon: Server },
] as const;

// Parking lots data
export const parkingLotsData = [
  { name: '万达广场停车场', address: '北京市朝阳区建国路93号', spaces: 500, used: 423 },
  { name: '万科城市之光', address: '上海市浦东新区张江路', spaces: 800, used: 654 },
  { name: '首都机场T3航站楼', address: '北京市顺义区首都机场路', spaces: 2000, used: 1856 },
] as const;

// Vehicles data
export const vehiclesData = [
  { plate: '京A12345', type: '临时车', time: '10:23', lot: '万达广场', status: '在场' },
  { plate: '沪B67890', type: '月卡车', time: '08:15', lot: '万科城市之光', status: '在场' },
  { plate: '京C24680', type: '临时车', time: '14:30', lot: '首都机场T3', status: '已离场' },
] as const;

// Orders data
export const ordersData = [
  { id: 'ORD001', plate: '京A12345', amount: '¥15.00', method: '微信支付', status: '已完成' },
  { id: 'ORD002', plate: '沪B67890', amount: '¥8.00', method: '支付宝', status: '已完成' },
  { id: 'ORD003', plate: '京C24680', amount: '¥45.00', method: '微信支付', status: '已完成' },
] as const;

// Billing rules data
export const billingRulesData = [
  { name: '临时车计费', type: '按时长', rule: '首小时5元，之后每小时3元', status: '启用' },
  { name: '月卡计费', type: '包月', rule: '每月300元，不限次数', status: '启用' },
  { name: 'VIP会员', type: '优惠', rule: '临时车8折优惠', status: '启用' },
] as const;

// Devices data
export const devicesData = [
  { name: '入口道闸 #1', type: '道闸', status: '在线', location: '万达广场 A入口' },
  { name: '出口道闸 #1', type: '道闸', status: '在线', location: '万达广场 A出口' },
  { name: '车牌识别相机 #1', type: '相机', status: '在线', location: '万达广场 A入口' },
  { name: '车牌识别相机 #2', type: '相机', status: '离线', location: '万达广场 B入口' },
] as const;

// Users data
export const usersData = [
  { name: '张三', role: '管理员', perms: '全部权限', status: '正常' },
  { name: '李四', role: '运营', perms: '查看、导出', status: '正常' },
  { name: '王五', role: '财务', perms: '订单、报表', status: '正常' },
] as const;

// Reports data
export const reportsStats = [
  { label: '本月总收入', value: '¥358,420' },
  { label: '本月停车次数', value: '28,456' },
  { label: '平均车位利用率', value: '86%' },
] as const;

export const reportsChartData = [
  { month: '1月', value: 60 },
  { month: '2月', value: 75 },
  { month: '3月', value: 65 },
  { month: '4月', value: 85 },
  { month: '5月', value: 70 },
  { month: '6月', value: 90 },
] as const;