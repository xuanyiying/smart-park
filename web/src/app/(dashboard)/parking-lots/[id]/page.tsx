"use client";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { useParams } from "next/navigation";

export default function ParkingLotDetailPage() {
  const params = useParams();
  const id = params.id;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold text-gray-900">停车场详情</h1>
          <p className="text-gray-500 mt-1">ID: {id}</p>
        </div>
        <Button variant="outline">编辑</Button>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>基本信息</CardTitle>
        </CardHeader>
        <CardContent>
          <div className="text-gray-400">详情页面占位 - 后续实现CRUD功能</div>
        </CardContent>
      </Card>
    </div>
  );
}
