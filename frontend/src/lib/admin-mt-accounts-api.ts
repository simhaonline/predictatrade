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
