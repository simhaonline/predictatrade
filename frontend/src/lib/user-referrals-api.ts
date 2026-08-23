import { customInstance } from "@/lib/axios-instance";

export interface NetworkReferral {
  child_user_id: string;
  email: string;
  full_name: string;
  level: number;
  created_at: string;
}

export interface ReferralNetwork {
  referrals: NetworkReferral[];
  count: number;
}

export async function fetchReferralNetwork(): Promise<ReferralNetwork> {
  const res = await customInstance.get<ReferralNetwork>("/referrals/network");
  return res.data ?? { referrals: [], count: 0 };
}

export interface ReferralCommission {
  id: string;
  commission_amount: string;
  status: string;
  created_at: string;
  source_email?: string;
  level?: number;
}

export async function fetchReferralCommissions(): Promise<ReferralCommission[]> {
  const res = await customInstance.get<ReferralCommission[]>("/referrals/commissions");
  return res.data ?? [];
}

export async function fetchReferralCode(): Promise<{ code: string }> {
  const res = await customInstance.get<{ code: string }>("/referrals/code");
  return res.data ?? { code: "" };
}
