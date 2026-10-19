import { api } from "@/lib/api";

export interface Station {
  id: string;
  lotId: string;
  name: string;
  stationType: string;
  status: string;
  connectorType: string;
  maxPower: number;
  voltage: number;
  totalConnectors: number;
  availableConnectors: number;
  location: string;
  floor: string;
  createdAt: string;
  updatedAt: string;
}

export interface CreateStationRequest {
  lotId: string;
  name: string;
  stationType: string;
  connectorType: string;
  maxPower: number;
  voltage: number;
  totalConnectors: number;
  location: string;
  floor: string;
}

export interface UpdateStationRequest {
  name: string;
  status: string;
  maxPower: number;
  totalConnectors: number;
  location: string;
  floor: string;
}

export interface Connector {
  id: string;
  stationId: string;
  number: number;
  type: string;
  status: string;
  maxPower: number;
  voltage: number;
  current: number;
  faultCode: string;
  createdAt: string;
  updatedAt: string;
}

export interface CreateConnectorRequest {
  stationId: string;
  number: number;
  type: string;
  maxPower: number;
  voltage: number;
}

export interface Session {
  id: string;
  stationId: string;
  connectorId: string;
  userId: string;
  vehiclePlate: string;
  startTime: string;
  endTime: string;
  startEnergy: number;
  endEnergy: number;
  chargedEnergy: number;
  cost: number;
  serviceFee: number;
  totalAmount: number;
  status: string;
  paymentStatus: string;
  payTime: string;
  paymentMethod: string;
  transactionId: string;
  createdAt: string;
  updatedAt: string;
}

export interface ChargingSummary {
  sessionId: string;
  stationName: string;
  connectorNum: number;
  vehiclePlate: string;
  startTime: string;
  endTime: string;
  duration: number;
  chargedEnergy: number;
  cost: number;
  serviceFee: number;
  totalAmount: number;
  paymentStatus: string;
}

export interface Price {
  id: string;
  stationId: string;
  name: string;
  startHour: number;
  endHour: number;
  pricePerKwh: number;
  serviceFee: number;
  peakLoad: number;
  offPeakLoad: number;
  isPeakHours: boolean;
  effectiveAt: string;
  expiresAt: string;
  createdAt: string;
  updatedAt: string;
}

export interface CreatePriceRequest {
  stationId: string;
  name: string;
  startHour: number;
  endHour: number;
  pricePerKwh: number;
  serviceFee: number;
  peakLoad: number;
  offPeakLoad: number;
  isPeakHours: boolean;
  effectiveAt: string;
  expiresAt: string;
}

export const chargingService = {
  createStation: (data: CreateStationRequest) =>
    api.post<Station>("/api/v1/charging/stations", data),

  getStation: (id: string) =>
    api.get<Station>(`/api/v1/charging/stations/${id}`),

  updateStation: (id: string, data: UpdateStationRequest) =>
    api.put<Station>(`/api/v1/charging/stations/${id}`, data),

  deleteStation: (id: string) =>
    api.delete<null>(`/api/v1/charging/stations/${id}`),

  listStations: (lotId?: string, page = 1, pageSize = 20) =>
    api.get<Station[]>(`/api/v1/charging/stations?lotId=${lotId || ""}&page=${page}&pageSize=${pageSize}`),

  getAvailableStations: (lotId: string) =>
    api.get<Station[]>(`/api/v1/charging/stations/available?lotId=${lotId}`),

  createConnector: (data: CreateConnectorRequest) =>
    api.post<Connector>("/api/v1/charging/connectors", data),

  getConnector: (id: string) =>
    api.get<Connector>(`/api/v1/charging/connectors/${id}`),

  listConnectors: (stationId: string) =>
    api.get<Connector[]>(`/api/v1/charging/stations/${stationId}/connectors`),

  startCharging: (data: { stationId: string; connectorId: string; userId: string; vehiclePlate: string }) =>
    api.post<Session>("/api/v1/charging/sessions/start", data),

  stopCharging: (sessionId: string, userId: string) =>
    api.post<Session>(`/api/v1/charging/sessions/${sessionId}/stop`, { userId }),

  getSession: (sessionId: string) =>
    api.get<Session>(`/api/v1/charging/sessions/${sessionId}`),

  listUserSessions: (userId: string, page = 1, pageSize = 20) =>
    api.get<Session[]>(`/api/v1/charging/sessions?userId=${userId}&page=${page}&pageSize=${pageSize}`),

  getChargingSummary: (sessionId: string) =>
    api.get<ChargingSummary>(`/api/v1/charging/sessions/${sessionId}/summary`),

  createPrice: (data: CreatePriceRequest) =>
    api.post<Price>("/api/v1/charging/prices", data),

  getCurrentPrice: (stationId: string) =>
    api.get<Price>(`/api/v1/charging/stations/${stationId}/price`),

  listPrices: (stationId: string) =>
    api.get<Price[]>(`/api/v1/charging/stations/${stationId}/prices`),

  confirmPayment: (sessionId: string, data: { transactionId: string; paymentMethod: string; paidAmount: number }) =>
    api.post<Session>(`/api/v1/charging/sessions/${sessionId}/payment/confirm`, data),
};
