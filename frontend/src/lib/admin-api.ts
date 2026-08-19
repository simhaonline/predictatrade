import { customInstance } from "@/lib/axios-instance";

// === Admin Overview ===
export async function fetchAdminOverview() {
  const res = await customInstance.get("/admin/overview");
  return res.data;
}

export async function fetchAdminHealth() {
  const res = await customInstance.get("/admin/health");
  return res.data;
}

// === Go Engine API (proxied through Nginx to port 13081) ===
export async function fetchEngineSignals() {
  const res = await customInstance.get("/signals");
  return res.data;
}

export async function fetchMarketState() {
  const res = await customInstance.get("/market/state");
  return res.data;
}

export async function fetchMarketSnapshot() {
  const res = await customInstance.get("/market/snapshot");
  return res.data;
}

export async function fetchAgentsStatus() {
  const res = await customInstance.get("/agents/status");
  return res.data;
}

export async function fetchEngineHealth() {
  const res = await customInstance.get("/health");
  return res.data;
}

// === Operations ===
export async function fetchOperationsState() {
  const res = await customInstance.get("/operations/state");
  return res.data;
}

export async function fetchActiveOperations() {
  const res = await customInstance.get("/operations/active");
  return res.data;
}

export async function haltTrading(reason: string) {
  const res = await customInstance.post("/operations/halt-trading", { reason });
  return res.data;
}

export async function resumeTrading(reason: string) {
  const res = await customInstance.post("/operations/resume-trading", { reason });
  return res.data;
}

export async function pauseSignals(reason: string) {
  const res = await customInstance.post("/operations/pause-signals", { reason });
  return res.data;
}

export async function resumeSignals(reason: string) {
  const res = await customInstance.post("/operations/resume-signals", { reason });
  return res.data;
}

export async function enableStrategy(id: string, reason: string) {
  const res = await customInstance.post(`/operations/strategy/${id}/enable`, { reason });
  return res.data;
}

export async function disableStrategy(id: string, reason: string) {
  const res = await customInstance.post(`/operations/strategy/${id}/disable`, { reason });
  return res.data;
}

// === AI Models ===
export async function fetchAIModels() {
  const res = await customInstance.get("/operations/ai/models");
  return res.data;
}

export async function fetchTrainingJobs() {
  const res = await customInstance.get("/operations/ai/training-jobs");
  return res.data;
}

export async function fetchInferenceHistory(limit?: number) {
  const res = await customInstance.get(`/operations/ai/inference${limit ? `?limit=${limit}` : ""}`);
  return res.data;
}

// === Users ===
export async function fetchAdminUsers(page: number, limit: number) {
  const res = await customInstance.get(`/admin/users?page=${page}&limit=${limit}`);
  return res.data;
}

export async function updateUserStatus(userId: string, status: string) {
  const res = await customInstance.patch(`/admin/users/${userId}/status?status=${status}`);
  return res.data;
}

// === Subscriptions ===
export async function fetchAdminSubscriptions(page: number, limit: number) {
  const res = await customInstance.get(`/admin/subscriptions?page=${page}&limit=${limit}`);
  return res.data;
}

// === Commissions ===
export async function fetchAdminCommissions(page: number, limit: number) {
  const res = await customInstance.get(`/admin/commissions?page=${page}&limit=${limit}`);
  return res.data;
}

export async function fetchCommissionSummary() {
  const res = await customInstance.get("/admin/commissions/summary");
  return res.data;
}

// === Payouts ===
export async function fetchAdminPayouts(page: number, limit: number) {
  const res = await customInstance.get(`/admin/payouts?page=${page}&limit=${limit}`);
  return res.data;
}

export async function fetchPayoutStats() {
  const res = await customInstance.get("/admin/payouts/stats");
  return res.data;
}

export async function approvePayout(payoutId: string) {
  const res = await customInstance.post(`/payouts/${payoutId}/approve`);
  return res.data;
}

// === Licenses ===
export async function fetchAdminLicenses(page: number, limit: number) {
  const res = await customInstance.get(`/admin/licenses?page=${page}&limit=${limit}`);
  return res.data;
}

// === Devices ===
export async function fetchAdminDevices(page: number, limit: number) {
  const res = await customInstance.get(`/admin/devices?page=${page}&limit=${limit}`);
  return res.data;
}

export async function fetchDeviceSessions() {
  const res = await customInstance.get("/devices/sessions");
  return res.data;
}

export async function revokeDevice(deviceId: string, reason: string) {
  const res = await customInstance.post(`/devices/devices/${deviceId}/revoke`, { reason });
  return res.data;
}

// === Audit ===
export async function fetchAuditLogs(page: number, limit: number) {
  const res = await customInstance.get(`/audit?page=${page}&limit=${limit}`);
  return res.data;
}

// === Plans ===
export async function fetchPlans() {
  const res = await customInstance.get("/plans");
  return res.data;
}

// === Billing ===
export async function fetchInvoices() {
  const res = await customInstance.get("/billing/invoices");
  return res.data;
}

// === User Profile ===
export async function fetchMyProfile() {
  const res = await customInstance.get("/users/me");
  return res.data;
}

export async function updateMyProfile(data: { displayName?: string; timezone?: string }) {
  const res = await customInstance.patch("/users/me", data);
  return res.data;
}

// === Phase 2: Regime Diagnostics (SOW Phase 2 Section 7) ===
export async function fetchRegimeDiagnostics() {
  const res = await customInstance.get("/admin/regime-diagnostics");
  return res.data;
}
