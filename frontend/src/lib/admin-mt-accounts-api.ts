import { customInstance } from "@/lib/axios-instance";

export interface MtAccountActivation {
  id: string;
  client_type?: string;
  terminal_build?: string;
  ea_version?: string;
  broker_name?: string;
  broker_server?: string;
  mt_account_login?: string;
  installation_id?: string;
  activated_at?: string;
  balance?: number;
  equity?: number;
  profit?: number;
  currency?: string;
  open_positions?: number;
  floating_pnl?: number;
  last_account_update?: string;
}

export interface MtAccountDevice {
  id: string;
  device_name?: string;
  hostname?: string;
  connection_status?: string;
  license_key?: string;
  license_status?: string;
  activations?: MtAccountActivation[];
}

export interface CreateMtAccountBody {
  deviceId: string;
  brokerName?: string;
  brokerServer?: string;
  mtAccountLogin: string;
  clientType?: string;
}

export async function fetchMtAccounts(): Promise<MtAccountDevice[]> {
  const res = await customInstance.get("/licensing/mt-accounts");
  return Array.isArray(res.data) ? (res.data as MtAccountDevice[]) : [];
}

export async function createMtAccount(body: CreateMtAccountBody) {
  const res = await customInstance.post("/licensing/mt-accounts", body);
  return res.data;
}

// Admin fleet-wide MT accounts (check.md #4): user-scoped endpoint returned
// empty for admin sessions; this is the admin-wide listing.
export interface AdminMtAccount {
  id: string;
  account_number?: string;
  broker?: string;
  client_type?: string;
  license_key?: string;
  license_status?: string;
  user_email?: string;
  hardware_id?: string;
  broker_server?: string;
  created_at?: string;
}

export async function fetchAllMtAccountsAdmin(): Promise<AdminMtAccount[]> {
  const res = await customInstance.get("/licensing/admin-mt-accounts");
  return Array.isArray(res.data) ? (res.data as AdminMtAccount[]) : [];
}
