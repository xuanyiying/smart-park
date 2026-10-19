import { api } from "@/lib/api";

// ==================== Tenant ====================

export interface TenantConfig {
  maxParkingLots: number;
  maxDevices: number;
  maxUsers: number;
  storageQuota: number;
  features: string[];
  customDomain: string;
  timezone: string;
  currency: string;
  language: string;
}

export interface Tenant {
  id: string;
  name: string;
  code: string;
  status: string; // active, disabled, expired
  config: TenantConfig;
  createdAt: string;
  updatedAt: string;
  expiredAt: string;
}

export interface CreateTenantData {
  name: string;
  code: string;
  config: Partial<TenantConfig>;
}

export interface UpdateTenantData {
  name: string;
  config: Partial<TenantConfig>;
}

// ==================== Service ====================

export const tenantService = {
  list: (page = 1, pageSize = 20): Promise<Tenant[]> =>
    api.get<Tenant[]>(`/api/v1/admin/tenants?page=${page}&pageSize=${pageSize}`),

  get: (tenantId: string): Promise<Tenant> =>
    api.get<Tenant>(`/api/v1/admin/tenants/${tenantId}`),

  create: (data: CreateTenantData): Promise<Tenant> =>
    api.post<Tenant>("/api/v1/admin/tenants", data),

  update: (tenantId: string, data: UpdateTenantData): Promise<Tenant> =>
    api.put<Tenant>(`/api/v1/admin/tenants/${tenantId}`, data),

  remove: (tenantId: string): Promise<void> =>
    api.delete<void>(`/api/v1/admin/tenants/${tenantId}`),

  enable: (tenantId: string): Promise<void> =>
    api.post<void>(`/api/v1/admin/tenants/${tenantId}/enable`),

  disable: (tenantId: string): Promise<void> =>
    api.post<void>(`/api/v1/admin/tenants/${tenantId}/disable`),
};
