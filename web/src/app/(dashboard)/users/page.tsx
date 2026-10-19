"use client";

import { UserDialog } from "@/components/modules/user-dialog";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { USER_ROLE_LABELS, USER_ROLE_OPTIONS_WITH_ALL } from "@/constants/user";
import { useApiMutation, useApiQuery } from "@/hooks/use-api";
import { userManagementService, type CreateUserData, type UpdateUserData, type User, type UserRole } from "@/services/user-management";
import { Pencil, Plus, Search, Trash2 } from "lucide-react";
import { useState } from "react";

const roleColors: Record<UserRole, string> = {
  admin: "bg-purple-100 text-purple-800",
  manager: "bg-blue-100 text-blue-800",
  operator: "bg-gray-100 text-gray-800",
};

export default function UsersPage() {
  const [roleFilter, setRoleFilter] = useState<string>("all");
  const [searchQuery, setSearchQuery] = useState("");
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingUser, setEditingUser] = useState<User | null>(null);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [deletingUserId, setDeletingUserId] = useState<string | null>(null);

  const { data, isLoading } = useApiQuery(
    ["users"],
    () => userManagementService.listUsers()
  );

  const users = data?.list || [];

  const createMutation = useApiMutation(
    (data: CreateUserData) => userManagementService.createUser(data),
    {
      successMessage: "用户创建成功",
      invalidateKeys: [["users"]],
    }
  );

  const updateMutation = useApiMutation(
    ({ id, data }: { id: string; data: UpdateUserData }) =>
      userManagementService.updateUser(id, data),
    {
      successMessage: "用户更新成功",
      invalidateKeys: [["users"]],
    }
  );

  const deleteMutation = useApiMutation(
    (id: string) => userManagementService.deleteUser(id),
    {
      successMessage: "用户删除成功",
      invalidateKeys: [["users"]],
    }
  );

  const filteredUsers = users.filter((user) => {
    const matchesRole = roleFilter === "all" || user.role === roleFilter;
    const matchesSearch = user.username.toLowerCase().includes(searchQuery.toLowerCase()) ||
      user.name.toLowerCase().includes(searchQuery.toLowerCase());
    return matchesRole && matchesSearch;
  });

  const handleRoleChange = (value: string | null) => {
    if (value) setRoleFilter(value);
  };

  const handleAddUser = () => {
    setEditingUser(null);
    setDialogOpen(true);
  };

  const handleEditUser = (user: User) => {
    setEditingUser(user);
    setDialogOpen(true);
  };

  const handleDeleteClick = (userId: string) => {
    setDeletingUserId(userId);
    setDeleteDialogOpen(true);
  };

  const handleConfirmDelete = () => {
    if (deletingUserId) {
      deleteMutation.mutate(deletingUserId, {
        onSuccess: () => {
          setDeleteDialogOpen(false);
          setDeletingUserId(null);
        },
      });
    }
  };

  const handleDialogSubmit = (formData: CreateUserData | UpdateUserData) => {
    if (editingUser) {
      updateMutation.mutate(
        { id: editingUser.id, data: formData as UpdateUserData },
        {
          onSuccess: () => {
            setDialogOpen(false);
          },
        }
      );
    } else {
      createMutation.mutate(formData as CreateUserData, {
        onSuccess: () => {
          setDialogOpen(false);
        },
      });
    }
  };

  const isSubmitting = createMutation.isPending || updateMutation.isPending;

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-semibold text-gray-900">用户管理</h1>
          <p className="text-gray-500 mt-1">管理系统用户账号与权限分配</p>
        </div>
        <Button onClick={handleAddUser}>
          <Plus className="w-4 h-4 mr-1" />
          新增用户
        </Button>
      </div>

      <Card>
        <CardHeader className="pb-3">
          <div className="flex items-center gap-4">
            <div className="relative flex-1 max-w-sm">
              <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
              <Input
                placeholder="搜索用户名或姓名..."
                className="pl-10"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
              />
            </div>
            <Select value={roleFilter} onValueChange={handleRoleChange}>
              <SelectTrigger className="w-[140px]">
                <SelectValue>{USER_ROLE_LABELS[roleFilter as UserRole | "all"]}</SelectValue>
              </SelectTrigger>
              <SelectContent>
                {USER_ROLE_OPTIONS_WITH_ALL.map((option) => (
                  <SelectItem key={option.value} value={option.value}>
                    {option.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </CardHeader>
        <CardContent>
          <div className="border rounded-lg">
            <table className="w-full">
              <thead className="bg-gray-50 border-b">
                <tr>
                  <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">用户名</th>
                  <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">姓名</th>
                  <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">角色</th>
                  <th className="text-left px-4 py-3 text-sm font-medium text-gray-500">操作</th>
                </tr>
              </thead>
              <tbody>
                {isLoading ? (
                  <tr>
                    <td colSpan={4} className="px-4 py-8 text-center text-gray-500">
                      加载中...
                    </td>
                  </tr>
                ) : filteredUsers.length === 0 ? (
                  <tr>
                    <td colSpan={4} className="px-4 py-8 text-center text-gray-500">
                      暂无数据
                    </td>
                  </tr>
                ) : (
                  filteredUsers.map((user) => (
                    <tr key={user.id} className="border-b last:border-0 hover:bg-gray-50">
                      <td className="px-4 py-3 font-medium">{user.username}</td>
                      <td className="px-4 py-3 text-gray-600">{user.name}</td>
                      <td className="px-4 py-3">
                        <span className={`inline-flex px-2 py-1 rounded-full text-xs font-medium ${roleColors[user.role as UserRole]}`}>
                          {USER_ROLE_LABELS[user.role as UserRole] || user.role}
                        </span>
                      </td>
                      <td className="px-4 py-3">
                        <div className="flex items-center gap-2">
                          <Button variant="outline" size="sm" onClick={() => handleEditUser(user)}>
                            <Pencil className="w-4 h-4" />
                          </Button>
                          <Button variant="outline" size="sm" onClick={() => handleDeleteClick(user.id)}>
                            <Trash2 className="w-4 h-4" />
                          </Button>
                        </div>
                      </td>
                    </tr>
                  ))
                )}
              </tbody>
            </table>
          </div>
        </CardContent>
      </Card>

      <UserDialog
        open={dialogOpen}
        onOpenChange={setDialogOpen}
        user={editingUser}
        onSubmit={handleDialogSubmit}
        isLoading={isSubmitting}
      />

      <Dialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
        <DialogContent className="sm:max-w-sm">
          <DialogHeader>
            <DialogTitle>确认删除</DialogTitle>
          </DialogHeader>
          <div className="py-4">
            <p className="text-gray-600">
              确定要删除用户 <span className="font-medium">{deletingUserId}</span> 吗？此操作不可撤销。
            </p>
          </div>
          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => setDeleteDialogOpen(false)}
            >
              取消
            </Button>
            <Button
              type="button"
              variant="destructive"
              onClick={handleConfirmDelete}
              disabled={deleteMutation.isPending}
            >
              {deleteMutation.isPending ? "删除中..." : "确认删除"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
