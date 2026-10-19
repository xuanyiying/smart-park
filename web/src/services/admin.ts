import { api } from "@/lib/api";

export interface ParkingLot {
  id: string;
  name: string;
  address: string;
  lanes: number;
  totalCapacity: number;
  status: string;
  createdAt: string;
  updatedAt: string;
}

export interface Vehicle {
  id: string;
  plateNumber: string;
  vehicleType: string;
  ownerName: string;
  ownerPhone: string;
  monthlyValidUntil: string;
  createdAt: string;
}

export interface Order {
  id: string;
  recordId: string;
  lotId: string;
  plateNumber: string;
  amount: number;
  discountAmount: number;
  finalAmount: number;
  status: string;
  payTime: string;
  payMethod: string;
}

export interface ParkingRecord {
  id: string;
  lotId: string;
  plateNumber: string;
  entryTime: number;
  exitTime: number;
  parkingDuration: number;
  status: string;
}

export interface DailyReport {
  lotId: string;
  date: string;
  totalEntries: number;
  totalExits: number;
  totalVehicles: number;
  totalAmount: number;
  totalDiscount: number;
  netAmount: number;
}

export interface MonthlyReport {
  lotId: string;
  year: number;
  month: number;
  totalEntries: number;
  totalExits: number;
  totalVehicles: number;
  totalAmount: number;
  totalDiscount: number;
  netAmount: number;
  dailyReports: DailyReport[];
}

export interface User {
  id: string;
  username: string;
  name: string;
  role: string;
  avatar: string;
}

export interface LoginData {
  token: string;
  user: User;
  expiresAt: string;
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

// Auth service
export const authService = {
  login: (username: string, password: string) =>
    api.post<LoginData>("/api/v1/admin/login", { username, password }),

  getCurrentUser: () =>
    api.get<User>("/api/v1/admin/user"),
};

// Admin service
export const adminService = {
  listParkingLots: (page = 1, pageSize = 10) =>
    api.get<ListResponse<ParkingLot>>(`/api/v1/admin/lots?page=${page}&pageSize=${pageSize}`),

  getParkingLot: (id: string) =>
    api.get<ParkingLot>(`/api/v1/admin/lots/${id}`),

  createParkingLot: (data: { name: string; address: string; lanes: number; totalCapacity?: number }) =>
    api.post<ParkingLot>("/api/v1/admin/lots", data),

  updateParkingLot: (id: string, data: { name: string; address: string; lanes: number; totalCapacity?: number; status: string }) =>
    api.put<null>(`/api/v1/admin/lots/${id}`, data),

  listVehicles: (page = 1, pageSize = 10, vehicleType?: string) =>
    api.get<ListResponse<Vehicle>>(`/api/v1/admin/vehicles?page=${page}&pageSize=${pageSize}&vehicleType=${vehicleType || ""}`),

  createVehicle: (data: { plateNumber: string; vehicleType: string; ownerName: string; ownerPhone: string; monthlyValidUntil: string }) =>
    api.post<Vehicle>("/api/v1/admin/vehicles", data),

  updateVehicle: (id: string, data: { plateNumber: string; vehicleType: string; ownerName: string; ownerPhone: string; monthlyValidUntil: string }) =>
    api.put<Vehicle>(`/api/v1/admin/vehicles/${id}`, data),

  listParkingRecords: (lotId?: string, startTime?: string, endTime?: string, plateNumber?: string, page = 1, pageSize = 10) =>
    api.get<ListResponse<ParkingRecord>>(`/api/v1/admin/records?lotId=${lotId || ""}&startTime=${startTime || ""}&endTime=${endTime || ""}&plateNumber=${plateNumber || ""}&page=${page}&pageSize=${pageSize}`),

  listOrders: (page = 1, pageSize = 10, status?: string, lotId?: string) =>
    api.get<ListResponse<Order>>(`/api/v1/admin/orders?page=${page}&pageSize=${pageSize}&status=${status || ""}&lotId=${lotId || ""}`),

  getOrder: (id: string) =>
    api.get<Order>(`/api/v1/admin/orders/${id}`),

  getDailyReport: (lotId: string, date: string) =>
    api.get<DailyReport>(`/api/v1/admin/reports/daily?lotId=${lotId}&date=${date}`),

  getMonthlyReport: (lotId: string, year: number, month: number) =>
    api.get<MonthlyReport>(`/api/v1/admin/reports/monthly?lotId=${lotId}&year=${year}&month=${month}`),
};
