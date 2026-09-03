import { customInstance } from "@/lib/axios-instance";

// === MFA ===
export interface MfaSetupResult {
  secret: string;
  otpauth: string;
}

export async function mfaSetup(): Promise<MfaSetupResult> {
  const res = await customInstance.post<MfaSetupResult>("/auth/mfa/setup");
  return res.data;
}

export async function mfaVerify(code: string): Promise<{ mfaEnabled: boolean; recoveryCodes: string[] }> {
  const res = await customInstance.post<{ mfaEnabled: boolean; recoveryCodes: string[] }>("/auth/mfa/verify", { code });
  return res.data;
}

// === Trusted Devices (user-scoped via licensing) ===
export interface DeviceActivation {
  id?: string;
  client_type?: string;
  broker_name?: string;
  mt_account_login?: string;
  activated_at?: string;
}

export interface TrustedDevice {
  id: string;
  device_name: string;
  hostname?: string;
  os?: string;
  agent_version?: string;
  status?: string;
  security_state?: string;
  registered_at?: string;
  last_seen_at?: string | null;
  installation_id?: string;
  fingerprint_hash?: string | null;
  license_key?: string | null;
  license_status?: string | null;
  revoked_at?: string | null;
  revocation_reason?: string | null;
  activations?: DeviceActivation[] | null;
}

export async function fetchTrustedDevices(): Promise<TrustedDevice[]> {
  const res = await customInstance.get<TrustedDevice[]>("/licensing/devices");
  return res.data ?? [];
}

export async function revokeTrustedDevice(id: string, reason = "user_revoke"): Promise<unknown> {
  const res = await customInstance.post(`/licensing/devices/${id}/revoke`, { reason });
  return res.data;
}

// === Sessions (admin-only endpoint; degrades in user context) ===
export async function fetchSessions(): Promise<unknown> {
  const res = await customInstance.get("/devices/sessions");
  return res.data;
}

// === Login History (admin-only audit endpoint; degrades in user context) ===
export async function fetchLoginHistory(): Promise<unknown> {
  const res = await customInstance.get("/audit");
  return res.data;
}
