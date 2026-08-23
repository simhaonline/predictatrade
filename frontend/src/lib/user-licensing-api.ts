import { customInstance } from "@/lib/axios-instance";

export interface LicenseRecord {
  id?: string;
  user_id?: string;
  plan_id?: string;
  license_key?: string;
  status?: string;
  max_devices?: number;
  max_mt_accounts?: number;
  created_at?: string;
  expires_at?: string | null;
  revoked_at?: string | null;
  revocation_reason?: string | null;
  plan_name?: string;
  device_count?: number;
  // Any other column may be present; rendered defensively as N/A when missing.
  [key: string]: unknown;
}

export async function fetchLicenses(): Promise<LicenseRecord[]> {
  const res = await customInstance.get<LicenseRecord[]>("/licensing/licenses");
  return res.data ?? [];
}

export interface MtAccount {
  id?: string;
  client_type?: string;
  broker_name?: string;
  broker_server?: string;
  mt_account_login?: string;
  device_name?: string;
  hostname?: string;
  status?: string;
  license_status?: string;
  floating_pnl?: number;
  account_balance?: number;
  account_equity?: number;
  last_account_update?: string;
}

export async function fetchMtAccounts(): Promise<MtAccount[]> {
  const res = await customInstance.get<MtAccount[]>("/licensing/mt-accounts");
  return res.data ?? [];
}
