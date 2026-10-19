import { api } from "@/lib/api";

// ==================== Device ====================

export interface DeviceInfo {
  deviceId: string;
  status: string;
  lastHeartbeat: string;
  online: boolean;
  laneId: string;
  lotId: string;
  deviceType: string;
  deviceSecret: string;
  createdAt: string;
  updatedAt: string;
  manufacturer: string;
  model: string;
  firmwareVersion: string;
  faultInfo: string;
  heartbeatCount: number;
  offlineCount: number;
  lastOnline: string;
  vendorSpecificConfig: Record<string, string>;
}

export interface DeviceListData {
  list: DeviceInfo[];
  total: number;
  page: number;
  pageSize: number;
}

export interface DeviceStatus {
  deviceId: string;
  online: boolean;
  status: string;
  lastHeartbeat: string;
}

export interface CreateDeviceData {
  deviceId: string;
  deviceType: string;
  lotId: string;
  laneId: string;
  status: string;
  manufacturer?: string;
  model?: string;
  firmwareVersion?: string;
}

export interface UpdateDeviceData {
  deviceType?: string;
  lotId?: string;
  laneId?: string;
  status?: string;
  manufacturer?: string;
  model?: string;
  firmwareVersion?: string;
}

// ==================== Manufacturer ====================

export interface Manufacturer {
  id: string;
  name: string;
  website: string;
  contactInfo: string;
  description: string;
  createdAt: string;
  updatedAt: string;
}

export interface ManufacturerListResponse {
  list: Manufacturer[];
  total: number;
}

export interface CreateManufacturerData {
  name: string;
  website?: string;
  contactInfo?: string;
  description?: string;
}

export interface UpdateManufacturerData {
  name?: string;
  website?: string;
  contactInfo?: string;
  description?: string;
}

// ==================== Firmware ====================

export interface Firmware {
  id: string;
  firmwareId: string;
  manufacturer: string;
  model: string;
  version: string;
  url: string;
  size: number;
  md5: string;
  description: string;
  status: string;
  releaseDate: string;
  createdAt: string;
  updatedAt: string;
}

export interface FirmwareListResponse {
  list: Firmware[];
  total: number;
}

export interface CreateFirmwareData {
  firmwareId: string;
  manufacturer: string;
  model: string;
  version: string;
  url: string;
  size: number;
  md5: string;
  description?: string;
  status?: string;
}

export interface UpdateFirmwareData {
  manufacturer?: string;
  model?: string;
  version?: string;
  url?: string;
  size?: number;
  md5?: string;
  description?: string;
  status?: string;
}

// ==================== Device Fault ====================

export interface DeviceFault {
  id: string;
  deviceId: string;
  faultType: string;
  faultCode: string;
  description: string;
  severity: string;
  status: string;
  suggestion: string;
  detectedAt: string;
  resolvedAt: string;
  createdAt: string;
  updatedAt: string;
}

export interface DeviceFaultListResponse {
  list: DeviceFault[];
  total: number;
}

export interface CreateDeviceFaultData {
  deviceId: string;
  faultType: string;
  faultCode: string;
  description: string;
  severity: string;
}

// ==================== Device Log ====================

export interface DeviceLog {
  id: string;
  deviceId: string;
  level: string;
  logType: string;
  message: string;
  detail: string;
  createdAt: string;
}

export interface DeviceLogListResponse {
  list: DeviceLog[];
  total: number;
}

// ==================== Device Performance ====================

export interface DevicePerformance {
  id: string;
  deviceId: string;
  cpuUsage: number;
  memoryUsage: number;
  storageUsage: number;
  networkIn: number;
  networkOut: number;
  temperature: number;
  timestamp: string;
  createdAt: string;
}

// ==================== Device Upgrade ====================

export interface UpgradeStatusData {
  upgradeId: string;
  status: string;
  message: string;
}

export interface UpgradeStatusDetail {
  upgradeId: string;
  deviceId: string;
  fromVersion: string;
  toVersion: string;
  status: string;
  errorMessage: string;
  startTime: string;
  endTime: string;
}

// ==================== Device Stats ====================

export interface DeviceStatsData {
  totalDevices: number;
  onlineDevices: number;
  faultDevices: number;
  onlineRate: number;
}

// ==================== API Response ====================

export interface ApiResponse<T> {
  code: number;
  message: string;
  data: T;
}

// ==================== Service ====================

export const deviceService = {
  // --- Device CRUD ---
  listDevices: async (
    page = 1,
    pageSize = 10,
  ): Promise<DeviceListData> => {
    const res = await api.get<ApiResponse<DeviceListData>>(
      "/api/v1/devices",
      { params: { page, pageSize } },
    );
    return res.data;
  },

  getDevice: async (deviceId: string): Promise<DeviceInfo> => {
    const res = await api.get<ApiResponse<DeviceInfo>>(
      `/api/v1/devices/${deviceId}`,
    );
    return res.data;
  },

  createDevice: async (data: CreateDeviceData): Promise<DeviceInfo> => {
    const res = await api.post<ApiResponse<DeviceInfo>>(
      "/api/v1/devices",
      data,
    );
    return res.data;
  },

  updateDevice: async (
    deviceId: string,
    data: UpdateDeviceData,
  ): Promise<DeviceInfo> => {
    const res = await api.put<ApiResponse<DeviceInfo>>(
      `/api/v1/devices/${deviceId}`,
      data,
    );
    return res.data;
  },

  deleteDevice: async (deviceId: string): Promise<void> => {
    await api.delete<ApiResponse<null>>(`/api/v1/devices/${deviceId}`);
  },

  // --- Device Status & Command ---
  getDeviceStatus: async (deviceId: string): Promise<DeviceStatus> => {
    const res = await api.get<ApiResponse<DeviceStatus>>(
      `/api/v1/device/${deviceId}/status`,
    );
    return res.data;
  },

  getDeviceHeartbeat: async (deviceId: string): Promise<DeviceStatus> => {
    const res = await api.get<ApiResponse<DeviceStatus>>(
      `/api/v1/device/${deviceId}/heartbeat`,
    );
    return res.data;
  },

  sendCommand: async (
    deviceId: string,
    command: string,
    params?: Record<string, string>,
  ): Promise<void> => {
    await api.post<ApiResponse<null>>(
      `/api/v1/device/${deviceId}/command`,
      { command, params },
    );
  },

  // --- Manufacturer ---
  listManufacturers: async (
    page = 1,
    pageSize = 20,
  ): Promise<ManufacturerListResponse> => {
    const res = await api.get<ApiResponse<ManufacturerListResponse>>(
      "/api/v1/manufacturers",
      { params: { page, pageSize } },
    );
    return res.data;
  },

  createManufacturer: async (
    data: CreateManufacturerData,
  ): Promise<Manufacturer> => {
    const res = await api.post<ApiResponse<Manufacturer>>(
      "/api/v1/manufacturers",
      data,
    );
    return res.data;
  },

  updateManufacturer: async (
    id: string,
    data: UpdateManufacturerData,
  ): Promise<Manufacturer> => {
    const res = await api.put<ApiResponse<Manufacturer>>(
      `/api/v1/manufacturers/${id}`,
      data,
    );
    return res.data;
  },

  deleteManufacturer: async (id: string): Promise<void> => {
    await api.delete<ApiResponse<null>>(`/api/v1/manufacturers/${id}`);
  },

  // --- Firmware ---
  listFirmwares: async (
    manufacturer?: string,
    model?: string,
    page = 1,
    pageSize = 20,
  ): Promise<FirmwareListResponse> => {
    const res = await api.get<ApiResponse<FirmwareListResponse>>(
      "/api/v1/firmwares",
      {
        params: {
          manufacturer: manufacturer || "",
          model: model || "",
          page,
          pageSize,
        },
      },
    );
    return res.data;
  },

  getLatestFirmware: async (
    manufacturer: string,
    model: string,
  ): Promise<Firmware> => {
    const res = await api.get<ApiResponse<Firmware>>(
      "/api/v1/firmwares/latest",
      { params: { manufacturer, model } },
    );
    return res.data;
  },

  createFirmware: async (data: CreateFirmwareData): Promise<Firmware> => {
    const res = await api.post<ApiResponse<Firmware>>(
      "/api/v1/firmwares",
      data,
    );
    return res.data;
  },

  updateFirmware: async (
    id: string,
    data: UpdateFirmwareData,
  ): Promise<Firmware> => {
    const res = await api.put<ApiResponse<Firmware>>(
      `/api/v1/firmwares/${id}`,
      data,
    );
    return res.data;
  },

  deleteFirmware: async (id: string): Promise<void> => {
    await api.delete<ApiResponse<null>>(`/api/v1/firmwares/${id}`);
  },

  // --- Device Fault ---
  listDeviceFaults: async (
    deviceId?: string,
    status?: string,
    page = 1,
    pageSize = 20,
  ): Promise<DeviceFaultListResponse> => {
    const res = await api.get<ApiResponse<DeviceFaultListResponse>>(
      "/api/v1/device/faults",
      {
        params: {
          deviceId: deviceId || "",
          status: status || "",
          page,
          pageSize,
        },
      },
    );
    return res.data;
  },

  createDeviceFault: async (
    data: CreateDeviceFaultData,
  ): Promise<DeviceFault> => {
    const res = await api.post<ApiResponse<DeviceFault>>(
      "/api/v1/device/fault",
      data,
    );
    return res.data;
  },

  resolveDeviceFault: async (id: string): Promise<void> => {
    await api.put<ApiResponse<null>>(`/api/v1/device/fault/${id}/resolve`);
  },

  // --- Device Log ---
  getDeviceLogs: async (
    deviceId: string,
    page = 1,
    pageSize = 20,
  ): Promise<DeviceLogListResponse> => {
    const res = await api.get<ApiResponse<DeviceLogListResponse>>(
      `/api/v1/device/${deviceId}/logs`,
      { params: { page, pageSize } },
    );
    return res.data;
  },

  // --- Device Performance ---
  getDevicePerformance: async (
    deviceId: string,
    startTime?: string,
    endTime?: string,
  ): Promise<DevicePerformance[]> => {
    const res = await api.get<ApiResponse<DevicePerformance[]>>(
      `/api/v1/device/${deviceId}/performance`,
      {
        params: {
          startTime: startTime || "",
          endTime: endTime || "",
        },
      },
    );
    return res.data;
  },

  getDevicePerformanceLatest: async (
    deviceId: string,
  ): Promise<DevicePerformance> => {
    const res = await api.get<ApiResponse<DevicePerformance>>(
      `/api/v1/device/${deviceId}/performance/latest`,
    );
    return res.data;
  },

  // --- Device Stats ---
  getDeviceStats: async (deviceId: string): Promise<DeviceStatsData> => {
    const res = await api.get<ApiResponse<DeviceStatsData>>(
      `/api/v1/device/${deviceId}/stats`,
    );
    return res.data;
  },

  getDeviceStatsSummary: async (
    deviceId: string,
  ): Promise<Record<string, unknown>> => {
    const res = await api.get<ApiResponse<Record<string, unknown>>>(
      `/api/v1/device/${deviceId}/stats-summary`,
    );
    return res.data;
  },

  getDeviceUsageStats: async (
    deviceId: string,
    startTime: string,
    endTime: string,
  ): Promise<Record<string, unknown>> => {
    const res = await api.get<ApiResponse<Record<string, unknown>>>(
      `/api/v1/device/${deviceId}/usage-stats`,
      { params: { startTime, endTime } },
    );
    return res.data;
  },

  getDeviceFaultStats: async (
    deviceId: string,
    startTime: string,
    endTime: string,
  ): Promise<Record<string, unknown>> => {
    const res = await api.get<ApiResponse<Record<string, unknown>>>(
      `/api/v1/device/${deviceId}/fault-stats`,
      { params: { startTime, endTime } },
    );
    return res.data;
  },

  // --- Device Upgrade ---
  upgradeDevice: async (
    deviceId: string,
    firmwareVersion: string,
    firmwareUrl?: string,
  ): Promise<UpgradeStatusData> => {
    const res = await api.post<ApiResponse<UpgradeStatusData>>(
      "/api/v1/device/upgrade",
      { deviceId, firmwareVersion, firmwareUrl },
    );
    return res.data;
  },

  getDeviceUpgradeStatus: async (
    upgradeId: string,
  ): Promise<UpgradeStatusDetail> => {
    const res = await api.get<ApiResponse<UpgradeStatusDetail>>(
      `/api/v1/device/upgrade/${upgradeId}`,
    );
    return res.data;
  },

  // --- Device Config ---
  updateDeviceConfig: async (
    deviceId: string,
    config: Record<string, string>,
  ): Promise<void> => {
    await api.put<ApiResponse<null>>(
      `/api/v1/device/${deviceId}/config`,
      { deviceId, config },
    );
  },
};
