"use client";

import { ParkingLotDialog } from "@/components/modules/parking-lot-dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { useApiMutation } from "@/hooks/use-api";
import { adminService, ParkingLot } from "@/services/admin";
import { Loader2, Plus, Search } from "lucide-react";
import Link from "next/link";
import { useEffect, useState } from "react";

export default function ParkingLotsPage() {
  const [search, setSearch] = useState("");
  const [parkingLots, setParkingLots] = useState<ParkingLot[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingParkingLot, setEditingParkingLot] = useState<ParkingLot | null>(null);

  const createMutation = useApiMutation(
    (data: { name: string; address: string; lanes: number }) =>
      adminService.createParkingLot(data),
    {
      successMessage: "停车场添加成功",
      onSuccess: () => {
        setDialogOpen(false);
        fetchParkingLots();
      },
    }
  );

  const updateMutation = useApiMutation(
    (data: { id: string; data: { name: string; address: string; lanes: number; status: string } }) =>
      adminService.updateParkingLot(data.id, data.data),
    {
      successMessage: "停车场更新成功",
      onSuccess: () => {
        setDialogOpen(false);
        fetchParkingLots();
      },
    }
  );

  useEffect(() => {
    fetchParkingLots();
  }, []);

  const fetchParkingLots = async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await adminService.listParkingLots(1, 100);
      setParkingLots(data.list || []);
    } catch (err) {
      setError("网络错误，请稍后重试");
      console.error("Failed to fetch parking lots:", err);
    } finally {
      setLoading(false);
    }
  };

  const filteredLots = parkingLots.filter((lot) => {
    const name = lot.name || "";
    const address = lot.address || "";
    const searchTerm = search.toLowerCase();
    return name.toLowerCase().includes(searchTerm) ||
      address.toLowerCase().includes(searchTerm);
  });

  const handleAddParkingLot = () => {
    setEditingParkingLot(null);
    setDialogOpen(true);
  };

  const handleEditParkingLot = (lot: ParkingLot) => {
    setEditingParkingLot(lot);
    setDialogOpen(true);
  };

  const handleSubmit = async (data: { name: string; address: string; lanes: number; status: string }) => {
    if (editingParkingLot) {
      await updateMutation.mutateAsync({ id: editingParkingLot.id, data });
    } else {
      await createMutation.mutateAsync({ name: data.name, address: data.address, lanes: data.lanes });
    }
  };

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold text-gray-900">停车场管理</h1>
          <p className="text-gray-500 mt-1">管理停车场信息</p>
        </div>
        <Button onClick={handleAddParkingLot}>
          <Plus className="w-4 h-4 mr-2" />
          新增停车场
        </Button>
      </div>

      <Card>
        <CardHeader className="pb-3">
          <div className="flex items-center gap-4">
            <div className="relative flex-1 max-w-sm">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
              <Input
                placeholder="搜索停车场..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="pl-10"
              />
            </div>
          </div>
        </CardHeader>
        <CardContent>
          {loading ? (
            <div className="flex items-center justify-center py-12">
              <Loader2 className="w-8 h-8 animate-spin text-gray-400" />
            </div>
          ) : error ? (
            <div className="text-center py-12">
              <p className="text-red-500">{error}</p>
              <Button variant="outline" onClick={fetchParkingLots} className="mt-4">
                重试
              </Button>
            </div>
          ) : (
            <div className="border rounded-lg">
              <table className="w-full">
                <thead className="bg-gray-50 border-b">
                  <tr>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">名称</th>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">地址</th>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">车道数</th>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">状态</th>
                    <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">操作</th>
                  </tr>
                </thead>
                <tbody>
                  {filteredLots.length === 0 ? (
                    <tr>
                      <td colSpan={5} className="px-4 py-8 text-center text-gray-500">
                        暂无停车场数据
                      </td>
                    </tr>
                  ) : (
                    filteredLots.map((lot) => (
                      <tr key={lot.id} className="border-b last:border-0 hover:bg-gray-50">
                        <td className="px-4 py-3 font-medium">{lot.name}</td>
                        <td className="px-4 py-3 text-gray-600">{lot.address}</td>
                        <td className="px-4 py-3">{lot.lanes}</td>
                        <td className="px-4 py-3">
                          <Badge variant={lot.status === "active" ? "default" : "secondary"}>
                            {lot.status === "active" ? "运营中" : "已停用"}
                          </Badge>
                        </td>
                        <td className="px-4 py-3">
                          <div className="flex items-center gap-2">
                            <Button
                              variant="ghost"
                              size="sm"
                              onClick={() => handleEditParkingLot(lot)}
                            >
                              编辑
                            </Button>
                            <Link href={`/parking-lots/${lot.id}`}>
                              <Button variant="ghost" size="sm">查看</Button>
                            </Link>
                          </div>
                        </td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </div>
          )}
        </CardContent>
      </Card>

      <ParkingLotDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        parkingLot={editingParkingLot}
        onSubmit={handleSubmit}
        isLoading={createMutation.isPending || updateMutation.isPending}
      />
    </div>
  );
}
