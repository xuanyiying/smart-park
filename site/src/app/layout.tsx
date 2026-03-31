import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "Smart Park - 智慧停车场管理系统",
  description:
    "Smart Park 是新一代智慧停车场管理系统，提供车辆进出管理、自动计费、移动支付、数据统计等全方位解决方案。",
  keywords: [
    "停车场管理",
    "智慧停车",
    "车牌识别",
    "自动计费",
    "移动支付",
    "停车系统",
  ],
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="zh-CN" className="h-full antialiased">
      <body className="min-h-full flex flex-col font-sans">{children}</body>
    </html>
  );
}
