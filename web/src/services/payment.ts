import { api } from "@/lib/api";

export interface ApiResponse<T> {
  code: number;
  message: string;
  data: T;
}

// ==================== Payment ====================

export interface CreatePaymentRequest {
  recordId: string;
  amount: number;
  payMethod: string;
  openId?: string;
  notifyUrl?: string;
  orderType?: string;
  sessionId?: string;
}

export interface PaymentData {
  orderId: string;
  amount: number;
  payUrl: string;
  qrCode: string;
  expireTime: string;
}

export interface PaymentStatusData {
  orderId: string;
  status: string;
  payTime: string;
  payMethod: string;
}

export interface PaymentOrder {
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

export interface RefundData {
  refundId: string;
  status: string;
}

// ==================== Reconciliation ====================

export interface ReconciliationRecord {
  id: string;
  orderId: string;
  paymentMethod: string;
  orderAmount: number;
  paidAmount: number;
  transactionId: string;
  reconciliationTime: string;
  status: "matched" | "mismatched" | "pending";
  notes: string;
}

export interface PaymentMethodStat {
  count: number;
  amount: number;
  paid: number;
}

export interface ReconciliationReport {
  startDate: string;
  endDate: string;
  totalOrders: number;
  totalAmount: number;
  totalPaid: number;
  matchedOrders: number;
  mismatchedOrders: number;
  paymentMethods: Record<string, PaymentMethodStat>;
}

// ==================== Service ====================

export const paymentService = {
  createPayment: (data: CreatePaymentRequest): Promise<PaymentData> =>
    api.post<PaymentData>("/api/v1/pay/create", data),

  getPaymentStatus: (orderId: string): Promise<PaymentStatusData> =>
    api.get<PaymentStatusData>(`/api/v1/pay/${orderId}/status`),

  refund: (orderId: string, reason: string): Promise<RefundData> =>
    api.post<RefundData>(`/api/v1/pay/${orderId}/refund`, { reason }),

  reconcileDaily: (date: string): Promise<ReconciliationRecord[]> =>
    api.post<ReconciliationRecord[]>("/api/v1/pay/reconcile/daily", { date }),

  getReconciliationReport: (
    startDate: string,
    endDate: string,
  ): Promise<ReconciliationReport> =>
    api.get<ReconciliationReport>("/api/v1/pay/reconcile/report", {
      params: { startDate, endDate },
    }),

  fixMismatchedOrders: async (orderIds: string[]): Promise<void> => {
    await api.post("/api/v1/pay/reconcile/fix", { orderIds });
  },

  listOrders: async (
    page = 1,
    pageSize = 20,
    status?: string,
  ): Promise<{ list: PaymentOrder[]; total: number }> =>
    api.get<{ list: PaymentOrder[]; total: number }>("/api/v1/admin/orders", {
      params: { page, pageSize, status: status || "" },
    }),
};
