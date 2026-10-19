"use client";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import type { CreateUserData, UpdateUserData, User, UserRole } from "@/services/user-management";
import { useEffect, useState } from "react";

interface UserDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  user?: User | null;
  onSubmit: (data: CreateUserData | UpdateUserData) => void;
  isLoading?: boolean;
}

const roleOptions: { value: UserRole; label: string }[] = [
  { value: "admin", label: "管理员" },
  { value: "manager", label: "经理" },
  { value: "operator", label: "操作员" },
];

export function UserDialog({
  open,
  onOpenChange,
  user,
  onSubmit,
  isLoading,
}: UserDialogProps) {
  const isEdit = !!user;
  const [formData, setFormData] = useState<CreateUserData>({
    username: "",
    password: "",
    role: "operator",
    email: "",
    status: "active",
  });

  useEffect(() => {
    if (user) {
      setFormData({
        username: user.username,
        password: "",
        role: user.role as UserRole,
        email: "",
        status: "active",
      });
    } else {
      setFormData({
        username: "",
        password: "",
        role: "operator",
        email: "",
        status: "active",
      });
    }
  }, [user, open]);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (isEdit) {
      const updateData: UpdateUserData = {
        username: formData.username,
        role: formData.role,
        email: formData.email,
        status: formData.status,
      };
      onSubmit(updateData);
    } else {
      onSubmit(formData);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{isEdit ? "编辑用户" : "新增用户"}</DialogTitle>
        </DialogHeader>
        <form onSubmit={handleSubmit}>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <label className="text-sm font-medium">用户名</label>
              <Input
                value={formData.username}
                onChange={(e) =>
                  setFormData({ ...formData, username: e.target.value })
                }
                placeholder="请输入用户名"
                required
              />
            </div>
            {!isEdit && (
              <div className="space-y-2">
                <label className="text-sm font-medium">密码</label>
                <Input
                  type="password"
                  value={formData.password}
                  onChange={(e) =>
                    setFormData({ ...formData, password: e.target.value })
                  }
                  placeholder="请输入密码"
                  required
                />
              </div>
            )}
            <div className="space-y-2">
              <label className="text-sm font-medium">角色</label>
              <Select
                value={formData.role}
                onValueChange={(value) =>
                  setFormData({ ...formData, role: value as UserRole })
                }
              >
                <SelectTrigger>
                  <SelectValue placeholder="请选择角色" />
                </SelectTrigger>
                <SelectContent>
                  {roleOptions.map((option) => (
                    <SelectItem key={option.value} value={option.value}>
                      {option.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>
          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => onOpenChange(false)}
            >
              取消
            </Button>
            <Button type="submit" disabled={isLoading}>
              {isLoading ? "保存中..." : "保存"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
