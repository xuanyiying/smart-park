import { api } from "@/lib/api";

// ==================== Blacklist ====================

export interface BlacklistEntry {
  plateNumber: string;
  reason: string;
  createdBy: string;
  active: boolean;
  createdAt: string;
}

export interface BlacklistListData {
  list: BlacklistEntry[];
  total: number;
}

export interface AddBlacklistData {
  plateNumber: string;
  reason: string;
  createdBy?: string;
}

// ==================== Service ====================

export const blacklistService = {
  list: (page = 1, pageSize = 20): Promise<BlacklistListData> =>
    api.get<BlacklistListData>(`/api/v1/blacklist?page=${page}&pageSize=${pageSize}`),

  add: (data: AddBlacklistData): Promise<BlacklistEntry> =>
    api.post<BlacklistEntry>("/api/v1/blacklist", {
      ...data,
      createdBy: data.createdBy || "admin",
    }),

  remove: (plateNumber: string): Promise<void> =>
    api.delete<void>(`/api/v1/blacklist/${encodeURIComponent(plateNumber)}`),
};
