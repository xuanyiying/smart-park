import { api } from "@/lib/api";

export interface CalculateFeeRequest {
  recordId: string;
  lotId: string;
  plateNumber: string;
  entryTime: number;
  exitTime: number;
  vehicleType: string;
}

export interface AppliedRule {
  ruleId: string;
  ruleName: string;
  amount: number;
}

export interface BillData {
  recordId: string;
  baseAmount: number;
  discountAmount: number;
  finalAmount: number;
  appliedRules: AppliedRule[];
}

export interface CalculateFeeResponse {
  code: number;
  message: string;
  data: BillData;
}

export interface BillingRule {
  id: string;
  lotId: string;
  ruleName: string;
  ruleType: string;
  conditionsJson: string;
  actionsJson: string;
  priority: number;
  isActive: boolean;
  createdAt: string;
}

export interface CreateBillingRuleRequest {
  lotId: string;
  ruleName: string;
  ruleType: string;
  conditionsJson: string;
  actionsJson: string;
  priority: number;
  isActive: boolean;
}

export interface UpdateBillingRuleRequest {
  id: string;
  ruleName: string;
  ruleType: string;
  conditionsJson: string;
  actionsJson: string;
  priority: number;
  isActive: boolean;
}

export const billingService = {
  calculateFee: (data: CalculateFeeRequest): Promise<BillData> =>
    api.post<BillData>("/api/v1/billing/calculate", data),

  createBillingRule: (data: CreateBillingRuleRequest): Promise<BillingRule> =>
    api.post<BillingRule>("/api/v1/admin/billing/rules", data),

  updateBillingRule: (
    id: string,
    data: Omit<UpdateBillingRuleRequest, "id">,
  ): Promise<void> =>
    api.put<void>(`/api/v1/admin/billing/rules/${id}`, data),

  deleteBillingRule: (id: string): Promise<void> =>
    api.delete<void>(`/api/v1/admin/billing/rules/${id}`),

  getBillingRules: (lotId: string): Promise<BillingRule[]> =>
    api.get<BillingRule[]>(`/api/v1/admin/billing/rules?lotId=${lotId}`),
};
