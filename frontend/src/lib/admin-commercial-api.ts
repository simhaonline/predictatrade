import { customInstance } from "@/lib/axios-instance";

// === Plans & Entitlements ===
export async function fetchPlansList() {
  const res = await customInstance.get("/plans");
  return res.data;
}

export async function fetchPlanById(id: string) {
  const res = await customInstance.get(`/plans/${id}`);
  return res.data;
}

// Plan update — backend exposes PATCH /api/v1/plans/:id (POST returns 405).
export async function updatePlan(id: string, payload: Record<string, unknown>) {
  const res = await customInstance.patch(`/plans/${id}`, payload);
  return res.data;
}

export async function fetchEntitlements() {
  const res = await customInstance.get("/subscriptions/entitlements");
  return res.data;
}

// === Commissions (admin read) ===
export async function fetchCommissionsAdminAll(page: number, limit: number) {
  const res = await customInstance.get(`/commissions/admin/all?page=${page}&limit=${limit}`);
  return res.data;
}

export async function fetchCommissionAdminSummary() {
  const res = await customInstance.get("/commissions/admin/summary");
  return res.data;
}

export async function fetchCommissionRules() {
  const res = await customInstance.get("/commissions/admin/rules");
  return res.data;
}

// === Commissions (admin lifecycle mutations) ===
export async function holdCommission(id: string, reason: string) {
  const res = await customInstance.post(`/commissions/admin/${id}/hold`, { reason });
  return res.data;
}

export async function releaseCommission(id: string, reason: string) {
  const res = await customInstance.post(`/commissions/admin/${id}/release`, { reason });
  return res.data;
}

export async function reverseCommission(id: string, reason: string, amount?: number) {
  const res = await customInstance.post(`/commissions/admin/${id}/reverse`, { reason, amount });
  return res.data;
}

export async function adjustCommission(id: string, amount: number, reason: string) {
  const res = await customInstance.post(`/commissions/admin/${id}/adjust`, { amount, reason });
  return res.data;
}

export async function clearEligibleCommissions() {
  const res = await customInstance.post(`/commissions/admin/clear-eligible`);
  return res.data;
}

export async function saveCommissionRule(id: string, payload: { base_rate?: number; active?: boolean; effective_until?: string }) {
  const res = await customInstance.put(`/commissions/admin/rules/${id}`, payload);
  return res.data;
}

// === Payouts (admin read) ===
export async function fetchPayoutsAdminAll(page: number, limit: number) {
  const res = await customInstance.get(`/admin/payouts?page=${page}&limit=${limit}`);
  return res.data;
}

export async function fetchPayoutStatsAdmin() {
  const res = await customInstance.get("/admin/payouts/stats");
  return res.data;
}

// === Referrals ===
export async function fetchReferralNetwork() {
  const res = await customInstance.get("/referrals/network");
  return res.data;
}

// === Licensing management (degrade gracefully — dedicated admin endpoints pending) ===
export async function fetchMtAccounts() {
  const res = await customInstance.get("/licensing/mt-accounts");
  return res.data;
}

export async function createMtAccount(payload: Record<string, unknown>) {
  const res = await customInstance.post("/licensing/mt-accounts", payload);
  return res.data;
}

export async function revokeLicenseDevice(deviceId: string, reason: string) {
  const res = await customInstance.post(`/licensing/devices/${deviceId}/revoke`, { reason });
  return res.data;
}

export async function createLicense(payload: Record<string, unknown>) {
  const res = await customInstance.post("/licensing/licenses", payload);
  return res.data;
}

export async function suspendLicense(id: string, reason: string) {
  const res = await customInstance.post(`/licensing/licenses/${id}/suspend`, { reason });
  return res.data;
}

export async function revokeLicense(id: string, reason: string) {
  const res = await customInstance.post(`/licensing/licenses/${id}/revoke`, { reason });
  return res.data;
}

export async function renewLicense(id: string) {
  const res = await customInstance.post(`/licensing/licenses/${id}/renew`, {});
  return res.data;
}

export async function resetLicense(id: string) {
  const res = await customInstance.post(`/licensing/licenses/${id}/reset`, {});
  return res.data;
}

export async function forceLogoutLicense(id: string) {
  const res = await customInstance.post(`/licensing/licenses/${id}/force-logout`, {});
  return res.data;
}

export async function fetchLicenseActivations(id: string) {
  const res = await customInstance.get(`/licensing/licenses/${id}/activations`);
  return res.data;
}

// === Device auth extra (degrade gracefully) ===
export async function resetDevice(id: string) {
  const res = await customInstance.post(`/licensing/devices/${id}/reset`, {});
  return res.data;
}

export async function forceUpgradeDevice(id: string) {
  const res = await customInstance.post(`/licensing/devices/${id}/force-upgrade`, {});
  return res.data;
}

export async function disableDeviceSignal(id: string) {
  const res = await customInstance.post(`/licensing/devices/${id}/disable-signal`, {});
  return res.data;
}

// === Local-only CSV export of already-fetched rows (legitimate) ===
export function exportRowsToCsv(rows: Record<string, unknown>[], filename: string) {
  if (!rows.length) {
    return false;
  }
  const headers = Array.from(new Set(rows.flatMap((r) => Object.keys(r))));
  const escape = (v: unknown) => {
    const s = v == null ? "" : String(v);
    return /[",\n]/.test(s) ? `"${s.replace(/"/g, '""')}"` : s;
  };
  const csv = [
    headers.join(","),
    ...rows.map((r) => headers.map((h) => escape(r[h])).join(",")),
  ].join("\n");
  const blob = new Blob([csv], { type: "text/csv;charset=utf-8;" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = filename;
  a.click();
  URL.revokeObjectURL(url);
  return true;
}
