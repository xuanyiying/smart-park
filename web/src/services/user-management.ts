import { api } from "@/lib/api";
import { authService } from "./admin";

export type UserRole = "admin" | "manager" | "operator";
export type UserStatus = "active" | "inactive";

// 注意：proto 的 User 消息只返回 id/username/name/role/avatar，
// email/status 仅作为创建/更新请求体字段，响应中不存在。
export interface CreateUserData {
  username: string;
  password: string;
  role: UserRole;
  email: string;
  status: UserStatus;
}

export interface UpdateUserData {
  username?: string;
  role?: UserRole;
  email?: string;
  status?: UserStatus;
}

export interface ApiResponse<T> {
  code: number;
  message: string;
  data: T;
}

export interface ListResponse<T> {
  list: T[];
  total: number;
  page: number;
  pageSize: number;
}

export interface User {
  id: string;
  username: string;
  name: string;
  role: string;
  avatar: string;
}

export interface UserListData {
  list: User[];
  total: number;
  page: number;
  pageSize: number;
}

/**
 * User Management Service
 *
 * All methods call real backend API endpoints.
 */
export const userManagementService = {
  /**
   * List all users with pagination
   */
  listUsers: async (page = 1, pageSize = 10): Promise<UserListData> => {
    return api.get<UserListData>(`/api/v1/admin/users?page=${page}&pageSize=${pageSize}`);
  },

  /**
   * Create a new user
   */
  createUser: async (data: CreateUserData): Promise<User> => {
    return api.post<User>("/api/v1/admin/users", {
      username: data.username,
      password: data.password,
      name: data.username, // Use username as name if not provided
      role: data.role,
      email: data.email,
      status: data.status,
    });
  },

  /**
   * Update an existing user
   */
  updateUser: async (id: string, data: UpdateUserData): Promise<User> => {
    return api.put<User>(`/api/v1/admin/users/${id}`, {
      username: data.username,
      name: data.username,
      role: data.role,
      email: data.email,
      status: data.status,
    });
  },

  /**
   * Delete a user
   */
  deleteUser: async (id: string): Promise<void> => {
    return api.delete<void>(`/api/v1/admin/users/${id}`);
  },

  /**
   * Get current logged-in user info
   */
  getCurrentUser: async () => {
    return authService.getCurrentUser();
  },
};
