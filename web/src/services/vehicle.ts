import { api } from "@/lib/api";

export interface EntryRequest {
  deviceId: string;
  plateNumber: string;
  plateImageUrl?: string;
  confidence?: number;
  vehicleType?: string;
  timestamp?: string;
}

export interface EntryData {
  recordId: string;
  plateNumber: string;
  allowed: boolean;
  gateOpen: boolean;
  displayMessage: string;
}

export interface EntryResponse {
  code: number;
  message: string;
  data: EntryData;
}

export interface ExitRequest {
  deviceId: string;
  plateNumber: string;
  plateImageUrl?: string;
  confidence?: number;
  timestamp?: string;
}

export interface ExitData {
  recordId: string;
  plateNumber: string;
  parkingDuration: number;
  amount: number;
  discountAmount: number;
  finalAmount: number;
  allowed: boolean;
  gateOpen: boolean;
  displayMessage: string;
}

export interface ExitResponse {
  code: number;
  message: string;
  data: ExitData;
}

export interface HeartbeatRequest {
  deviceId: string;
  status: string;
  timestamp?: string;
}

export interface HeartbeatResponse {
  code: number;
  message: string;
}

export interface DeviceStatus {
  deviceId: string;
  online: boolean;
  status: string;
  lastHeartbeat: string;
}

export interface GetDeviceStatusResponse {
  code: number;
  message: string;
  data: DeviceStatus;
}

export interface SendCommandRequest {
  deviceId: string;
  command: string;
  params?: Record<string, string>;
}

export interface CommandData {
  commandId: string;
  status: string;
}

export interface SendCommandResponse {
  code: number;
  message: string;
  data: CommandData;
}

export interface VehicleInfo {
  plateNumber: string;
  vehicleType: string;
  ownerName: string;
  ownerPhone: string;
  monthlyValidUntil: string;
}

export interface GetVehicleInfoResponse {
  code: number;
  message: string;
  data: VehicleInfo;
}

export const vehicleService = {
  entry: (data: EntryRequest) =>
    api.post<EntryResponse>("/api/v1/device/entry", data),

  exit: (data: ExitRequest) =>
    api.post<ExitResponse>("/api/v1/device/exit", data),

  heartbeat: (data: HeartbeatRequest) =>
    api.post<HeartbeatResponse>("/api/v1/device/heartbeat", data),

  getDeviceStatus: (deviceId: string) =>
    api.get<GetDeviceStatusResponse>(`/api/v1/device/${deviceId}/status`),

  sendCommand: (deviceId: string, data: { command: string; params?: Record<string, string> }) =>
    api.post<SendCommandResponse>(`/api/v1/device/${deviceId}/command`, data),

  getVehicleInfo: (plateNumber: string) =>
    api.get<GetVehicleInfoResponse>(`/api/v1/vehicle/${plateNumber}`),
};
