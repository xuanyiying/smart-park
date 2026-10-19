import Link from "next/link";
import { Button } from "@/components/ui/button";
import {
  Car,
  CreditCard,
  BarChart3,
  Shield,
  Zap,
  Clock,
  CheckCircle,
  ArrowRight,
  Smartphone,
  Users,
  TrendingUp,
  Building2,
} from "lucide-react";

export default function HomePage() {
  return (
    <div className="min-h-screen bg-white">
      {/* Header */}
      <header className="fixed top-0 left-0 right-0 z-50 bg-white/80 backdrop-blur-md border-b border-gray-100">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between h-16">
            <Link href="/" className="flex items-center gap-2">
              <div className="w-8 h-8 bg-blue-600 rounded-lg flex items-center justify-center">
                <Car className="w-5 h-5 text-white" />
              </div>
              <span className="text-xl font-bold text-gray-900">Smart Park</span>
            </Link>

            <nav className="hidden md:flex items-center gap-8">
              <Link
                href="#features"
                className="text-gray-600 hover:text-gray-900 transition-colors"
              >
                功能特性
              </Link>
              <Link
                href="#solutions"
                className="text-gray-600 hover:text-gray-900 transition-colors"
              >
                解决方案
              </Link>
              <Link
                href="#pricing"
                className="text-gray-600 hover:text-gray-900 transition-colors"
              >
                价格方案
              </Link>
              <Link
                href="#contact"
                className="text-gray-600 hover:text-gray-900 transition-colors"
              >
                联系我们
              </Link>
            </nav>

            <div className="flex items-center gap-4">
              <Link href="/login">
                <Button variant="ghost" size="sm">
                  登录
                </Button>
              </Link>
              <Link href="/login">
                <Button size="sm">免费试用</Button>
              </Link>
            </div>
          </div>
        </div>
      </header>

      {/* Hero Section */}
      <section className="pt-24 pb-16 px-4 sm:px-6 lg:px-8 bg-gradient-to-b from-blue-50 to-white">
        <div className="max-w-7xl mx-auto">
          <div className="text-center">
            <h1 className="text-4xl sm:text-5xl lg:text-6xl font-bold text-gray-900 tracking-tight">
              智慧停车
              <span className="text-blue-600">，</span>
              <br className="hidden sm:block" />
              从这里开始
            </h1>
            <p className="mt-6 text-lg sm:text-xl text-gray-600 max-w-3xl mx-auto">
              Smart Park 是新一代智慧停车场管理系统，提供车辆进出管理、自动计费、移动支付、数据统计等全方位解决方案，让停车场运营更高效、更智能。
            </p>
            <div className="mt-10 flex flex-col sm:flex-row gap-4 justify-center">
              <Link href="/login">
                <Button size="lg" className="gap-2">
                  开始免费试用
                  <ArrowRight className="w-4 h-4" />
                </Button>
              </Link>
              <Link href="#features">
                <Button size="lg" variant="outline">
                  了解更多
                </Button>
              </Link>
            </div>
          </div>

          {/* Hero Image/Dashboard Preview */}
          <div className="mt-16 relative">
            <div className="bg-white rounded-2xl shadow-2xl border border-gray-200 overflow-hidden">
              <div className="bg-gray-100 px-4 py-3 flex items-center gap-2">
                <div className="flex gap-1.5">
                  <div className="w-3 h-3 rounded-full bg-red-400"></div>
                  <div className="w-3 h-3 rounded-full bg-yellow-400"></div>
                  <div className="w-3 h-3 rounded-full bg-green-400"></div>
                </div>
                <div className="flex-1 text-center text-sm text-gray-500">
                  Smart Park 管理后台
                </div>
              </div>
              <div className="p-6 bg-gray-50">
                <div className="grid grid-cols-4 gap-4 mb-6">
                  {[
                    { label: "今日收入", value: "¥12,580" },
                    { label: "今日入场", value: "1,234" },
                    { label: "今日出场", value: "1,189" },
                    { label: "在线设备", value: "48" },
                  ].map((stat) => (
                    <div
                      key={stat.label}
                      className="bg-white rounded-lg p-4 border border-gray-200"
                    >
                      <div className="text-sm text-gray-500">{stat.label}</div>
                      <div className="text-2xl font-bold text-gray-900">
                        {stat.value}
                      </div>
                    </div>
                  ))}
                </div>
                <div className="bg-white rounded-lg p-4 border border-gray-200 h-48 flex items-center justify-center text-gray-400">
                  [ 收入趋势图表 ]
                </div>
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* Features Section */}
      <section id="features" className="py-20 px-4 sm:px-6 lg:px-8">
        <div className="max-w-7xl mx-auto">
          <div className="text-center mb-16">
            <h2 className="text-3xl sm:text-4xl font-bold text-gray-900">
              强大的功能特性
            </h2>
            <p className="mt-4 text-lg text-gray-600">
              全方位覆盖停车场运营的各个环节
            </p>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-8">
            {[
              {
                icon: Car,
                title: "车牌识别",
                description:
                  "支持高清摄像头车牌自动识别，识别率高达99.5%，支持多种车牌类型。",
              },
              {
                icon: CreditCard,
                title: "自动计费",
                description:
                  "灵活的计费规则配置，支持按时长、按时段、会员优惠等多种计费模式。",
              },
              {
                icon: Smartphone,
                title: "移动支付",
                description:
                  "集成微信支付、支付宝等主流支付方式，支持扫码缴费、无感支付。",
              },
              {
                icon: BarChart3,
                title: "数据统计",
                description:
                  "实时数据看板，多维度报表分析，助力运营决策优化。",
              },
              {
                icon: Shield,
                title: "安全可靠",
                description:
                  "企业级安全架构，数据加密存储，完善的权限管理体系。",
              },
              {
                icon: Zap,
                title: "高性能",
                description:
                  "微服务架构设计，支持高并发场景，系统稳定可靠。",
              },
            ].map((feature) => (
              <div
                key={feature.title}
                className="bg-white rounded-xl p-6 border border-gray-200 hover:border-blue-200 hover:shadow-lg transition-all"
              >
                <div className="w-12 h-12 bg-blue-100 rounded-lg flex items-center justify-center mb-4">
                  <feature.icon className="w-6 h-6 text-blue-600" />
                </div>
                <h3 className="text-xl font-semibold text-gray-900 mb-2">
                  {feature.title}
                </h3>
                <p className="text-gray-600">{feature.description}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Solutions Section */}
      <section
        id="solutions"
        className="py-20 px-4 sm:px-6 lg:px-8 bg-gray-50"
      >
        <div className="max-w-7xl mx-auto">
          <div className="text-center mb-16">
            <h2 className="text-3xl sm:text-4xl font-bold text-gray-900">
              适用于多种场景
            </h2>
            <p className="mt-4 text-lg text-gray-600">
              无论您的停车场规模大小，我们都有合适的解决方案
            </p>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
            {[
              {
                icon: Building2,
                title: "商业综合体",
                features: [
                  "多入口统一管理",
                  "商户优惠券核销",
                  "VIP会员管理",
                  "车位引导系统",
                ],
              },
              {
                icon: Users,
                title: "住宅小区",
                features: [
                  "业主车位管理",
                  "访客临时停车",
                  "月租续费提醒",
                  "车位共享功能",
                ],
              },
              {
                icon: TrendingUp,
                title: "公共停车场",
                features: [
                  "无人值守运营",
                  "电子发票开具",
                  "车位实时发布",
                  "运营数据分析",
                ],
              },
            ].map((solution) => (
              <div
                key={solution.title}
                className="bg-white rounded-xl p-8 border border-gray-200"
              >
                <div className="w-12 h-12 bg-blue-600 rounded-lg flex items-center justify-center mb-6">
                  <solution.icon className="w-6 h-6 text-white" />
                </div>
                <h3 className="text-xl font-semibold text-gray-900 mb-4">
                  {solution.title}
                </h3>
                <ul className="space-y-3">
                  {solution.features.map((feature) => (
                    <li key={feature} className="flex items-center gap-2">
                      <CheckCircle className="w-5 h-5 text-green-500" />
                      <span className="text-gray-600">{feature}</span>
                    </li>
                  ))}
                </ul>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* Pricing Section */}
      <section id="pricing" className="py-20 px-4 sm:px-6 lg:px-8">
        <div className="max-w-7xl mx-auto">
          <div className="text-center mb-16">
            <h2 className="text-3xl sm:text-4xl font-bold text-gray-900">
              灵活的价格方案
            </h2>
            <p className="mt-4 text-lg text-gray-600">
              按需选择，透明定价，无隐藏费用
            </p>
          </div>

          <div className="grid grid-cols-1 md:grid-cols-3 gap-8 max-w-5xl mx-auto">
            {[
              {
                name: "基础版",
                price: "免费",
                period: "",
                description: "适合小型停车场试用",
                features: [
                  "最多 1 个停车场",
                  "最多 2 个出入口",
                  "基础计费功能",
                  "数据保留 30 天",
                  "社区支持",
                ],
                highlighted: false,
              },
              {
                name: "专业版",
                price: "¥999",
                period: "/月",
                description: "适合中型停车场运营",
                features: [
                  "最多 5 个停车场",
                  "不限出入口数量",
                  "高级计费规则",
                  "数据保留 1 年",
                  "移动支付集成",
                  "优先技术支持",
                ],
                highlighted: true,
              },
              {
                name: "企业版",
                price: "联系我们",
                period: "",
                description: "适合大型连锁停车场",
                features: [
                  "不限停车场数量",
                  "定制化开发",
                  "私有化部署",
                  "数据永久保留",
                  "专属客户经理",
                  "SLA 服务保障",
                ],
                highlighted: false,
              },
            ].map((plan) => (
              <div
                key={plan.name}
                className={`rounded-2xl p-8 ${
                  plan.highlighted
                    ? "bg-blue-600 text-white ring-4 ring-blue-200 scale-105"
                    : "bg-white border border-gray-200"
                }`}
              >
                <h3
                  className={`text-xl font-semibold ${
                    plan.highlighted ? "text-white" : "text-gray-900"
                  }`}
                >
                  {plan.name}
                </h3>
                <p
                  className={`mt-2 text-sm ${
                    plan.highlighted ? "text-blue-100" : "text-gray-500"
                  }`}
                >
                  {plan.description}
                </p>
                <div className="mt-6">
                  <span
                    className={`text-4xl font-bold ${
                      plan.highlighted ? "text-white" : "text-gray-900"
                    }`}
                  >
                    {plan.price}
                  </span>
                  <span
                    className={`${
                      plan.highlighted ? "text-blue-100" : "text-gray-500"
                    }`}
                  >
                    {plan.period}
                  </span>
                </div>
                <ul className="mt-8 space-y-4">
                  {plan.features.map((feature) => (
                    <li key={feature} className="flex items-center gap-2">
                      <CheckCircle
                        className={`w-5 h-5 ${
                          plan.highlighted ? "text-blue-200" : "text-green-500"
                        }`}
                      />
                      <span
                        className={
                          plan.highlighted ? "text-blue-100" : "text-gray-600"
                        }
                      >
                        {feature}
                      </span>
                    </li>
                  ))}
                </ul>
                <Link href="/login" className="block mt-8">
                  <Button
                    className="w-full"
                    variant={plan.highlighted ? "secondary" : "default"}
                  >
                    {plan.price === "联系我们" ? "联系销售" : "立即开始"}
                  </Button>
                </Link>
              </div>
            ))}
          </div>
        </div>
      </section>

      {/* CTA Section */}
      <section className="py-20 px-4 sm:px-6 lg:px-8 bg-blue-600">
        <div className="max-w-4xl mx-auto text-center">
          <h2 className="text-3xl sm:text-4xl font-bold text-white">
            准备好开始了吗？
          </h2>
          <p className="mt-4 text-lg text-blue-100">
            立即注册，开启智慧停车管理之旅
          </p>
          <div className="mt-8 flex flex-col sm:flex-row gap-4 justify-center">
            <Link href="/login">
              <Button
                size="lg"
                variant="secondary"
                className="gap-2 bg-white text-blue-600 hover:bg-gray-100"
              >
                免费试用
                <ArrowRight className="w-4 h-4" />
              </Button>
            </Link>
            <Link href="#contact">
              <Button
                size="lg"
                variant="outline"
                className="border-white text-white hover:bg-blue-700"
              >
                预约演示
              </Button>
            </Link>
          </div>
        </div>
      </section>

      {/* Contact Section */}
      <section id="contact" className="py-20 px-4 sm:px-6 lg:px-8">
        <div className="max-w-7xl mx-auto">
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-12">
            <div>
              <h2 className="text-3xl sm:text-4xl font-bold text-gray-900">
                联系我们
              </h2>
              <p className="mt-4 text-lg text-gray-600">
                有任何问题或需求？我们的团队随时为您服务。
              </p>

              <div className="mt-8 space-y-6">
                <div className="flex items-start gap-4">
                  <div className="w-10 h-10 bg-blue-100 rounded-lg flex items-center justify-center flex-shrink-0">
                    <Clock className="w-5 h-5 text-blue-600" />
                  </div>
                  <div>
                    <h4 className="font-semibold text-gray-900">工作时间</h4>
                    <p className="text-gray-600">
                      周一至周五 9:00 - 18:00
                    </p>
                  </div>
                </div>
                <div className="flex items-start gap-4">
                  <div className="w-10 h-10 bg-blue-100 rounded-lg flex items-center justify-center flex-shrink-0">
                    <Shield className="w-5 h-5 text-blue-600" />
                  </div>
                  <div>
                    <h4 className="font-semibold text-gray-900">技术支持</h4>
                    <p className="text-gray-600">support@smartpark.com</p>
                  </div>
                </div>
                <div className="flex items-start gap-4">
                  <div className="w-10 h-10 bg-blue-100 rounded-lg flex items-center justify-center flex-shrink-0">
                    <Users className="w-5 h-5 text-blue-600" />
                  </div>
                  <div>
                    <h4 className="font-semibold text-gray-900">商务合作</h4>
                    <p className="text-gray-600">business@smartpark.com</p>
                  </div>
                </div>
              </div>
            </div>

            <div className="bg-gray-50 rounded-2xl p-8">
              <form className="space-y-6">
                <div>
                  <label
                    htmlFor="name"
                    className="block text-sm font-medium text-gray-700 mb-2"
                  >
                    姓名
                  </label>
                  <input
                    type="text"
                    id="name"
                    className="w-full px-4 py-3 rounded-lg border border-gray-300 focus:ring-2 focus:ring-blue-500 focus:border-transparent outline-none"
                    placeholder="请输入您的姓名"
                  />
                </div>
                <div>
                  <label
                    htmlFor="email"
                    className="block text-sm font-medium text-gray-700 mb-2"
                  >
                    邮箱
                  </label>
                  <input
                    type="email"
                    id="email"
                    className="w-full px-4 py-3 rounded-lg border border-gray-300 focus:ring-2 focus:ring-blue-500 focus:border-transparent outline-none"
                    placeholder="请输入您的邮箱"
                  />
                </div>
                <div>
                  <label
                    htmlFor="message"
                    className="block text-sm font-medium text-gray-700 mb-2"
                  >
                    留言
                  </label>
                  <textarea
                    id="message"
                    rows={4}
                    className="w-full px-4 py-3 rounded-lg border border-gray-300 focus:ring-2 focus:ring-blue-500 focus:border-transparent outline-none resize-none"
                    placeholder="请输入您的留言"
                  />
                </div>
                <Button className="w-full">提交留言</Button>
              </form>
            </div>
          </div>
        </div>
      </section>

      {/* Footer */}
      <footer className="bg-gray-900 text-gray-300">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-12">
          <div className="grid grid-cols-1 md:grid-cols-4 gap-8">
            <div>
              <div className="flex items-center gap-2 mb-4">
                <div className="w-8 h-8 bg-blue-600 rounded-lg flex items-center justify-center">
                  <Car className="w-5 h-5 text-white" />
                </div>
                <span className="text-xl font-bold text-white">Smart Park</span>
              </div>
              <p className="text-sm text-gray-400">
                新一代智慧停车场管理系统，让停车更简单、更智能。
              </p>
            </div>

            <div>
              <h4 className="text-white font-semibold mb-4">产品</h4>
              <ul className="space-y-2 text-sm">
                <li>
                  <Link href="#features" className="hover:text-white">
                    功能特性
                  </Link>
                </li>
                <li>
                  <Link href="#pricing" className="hover:text-white">
                    价格方案
                  </Link>
                </li>
                <li>
                  <Link href="#" className="hover:text-white">
                    更新日志
                  </Link>
                </li>
              </ul>
            </div>

            <div>
              <h4 className="text-white font-semibold mb-4">支持</h4>
              <ul className="space-y-2 text-sm">
                <li>
                  <Link href="#" className="hover:text-white">
                    帮助中心
                  </Link>
                </li>
                <li>
                  <Link href="#" className="hover:text-white">
                    API 文档
                  </Link>
                </li>
                <li>
                  <Link href="#contact" className="hover:text-white">
                    联系我们
                  </Link>
                </li>
              </ul>
            </div>

            <div>
              <h4 className="text-white font-semibold mb-4">关于</h4>
              <ul className="space-y-2 text-sm">
                <li>
                  <Link href="#" className="hover:text-white">
                    关于我们
                  </Link>
                </li>
                <li>
                  <Link href="#" className="hover:text-white">
                    隐私政策
                  </Link>
                </li>
                <li>
                  <Link href="#" className="hover:text-white">
                    服务条款
                  </Link>
                </li>
              </ul>
            </div>
          </div>

          <div className="border-t border-gray-800 mt-8 pt-8 text-center text-sm text-gray-400">
            <p>&copy; {new Date().getFullYear()} Smart Park. All rights reserved.</p>
          </div>
        </div>
      </footer>
    </div>
  );
}
