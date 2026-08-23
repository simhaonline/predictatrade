import { customInstance } from "@/lib/axios-instance";

export type PayoutStatus = "PENDING" | "APPROVED" | "PAID" | "REJECTED" | string;

export interface Payout {
  id: string;
  amount: number;
  method: string;
  destination: string;
  status: PayoutStatus;
  created_at?: string;
  updated_at?: string;
  notes?: string;
}

export interface RequestPayoutPayload {
  amount: number;
  method: string;
  destination: string;
}

export async function fetchPayouts(): Promise<Payout[]> {
  const res = await customInstance.get<Payout[]>("/payouts");
  return res.data ?? [];
}

export async function requestPayout(payload: RequestPayoutPayload): Promise<Payout> {
  const res = await customInstance.post<Payout>("/payouts/request", payload);
  return res.data;
}
