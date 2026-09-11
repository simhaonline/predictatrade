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
// empty for admin sessions; this is the admin-wide listing. Enriched from
// device_activations so it carries live balance/connection/device data.
export interface AdminMtAccount {
  id: string;
  mt_account_login?: string;
  broker_name?: string;
  broker_server?: string;
  client_type?: string;
  account_balance?: number;
  account_equity?: number;
  currency?: string;
  connection_status?: string;
  device_name?: string;
  license_key?: string;
  license_status?: string;
  user_email?: string;
  activated_at?: string;
}

export async function fetchAllMtAccountsAdmin(): Promise<AdminMtAccount[]> {
  const res = await customInstance.get("/licensing/admin-mt-accounts");
  return Array.isArray(res.data) ? (res.data as AdminMtAccount[]) : [];
}

// Admin-wide device listing (powers the MT-account registration dropdown).
export interface AdminDevice {
  id: string;
  device_name?: string;
  hostname?: string;
  connection_status?: string;
  license_key?: string;
  license_status?: string;
  user_email?: string;
}

export async function fetchAllDevicesAdmin(): Promise<AdminDevice[]> {
  const res = await customInstance.get("/licensing/admin-devices");
  return Array.isArray(res.data) ? (res.data as AdminDevice[]) : [];
}

