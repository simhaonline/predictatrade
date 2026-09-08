"use client";
import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import DataTable, { DataTableColumn } from "@/components/ui/data-table";
import StatusBadge from "@/components/ui/status-badge";
import ConfirmDialog from "@/components/admin/confirm-dialog";
import { format } from "date-fns";
import { toast } from "sonner";
import { approveEntitlement } from "@/lib/admin-commercial-api";
import { IconEye, IconKey, IconDownload } from "@tabler/icons-react";

interface User {
  id: string;
  email: string;
  full_name: string;
  status: string;
  role: string;
  created_at: string;
  last_login_at: string | null;
  subscription_plan?: string | null;
  license_status?: string | null;
  has_license?: boolean;
}

interface UsersResponse {
  items: User[];
  total: number;
  page: number;
  limit: number;
}

export default function AdminUsersPage() {
  const queryClient = useQueryClient();
  const [page, setPage] = useState(1);
  const [selectedUser, setSelectedUser] = useState<User | null>(null);
  const [confirm, setConfirm] = useState<{ action: string; userId: string; status: string } | null>(null);
  const limit = 20;

  const { data, isLoading, error, refetch } = useQuery<UsersResponse>({
    queryKey: ["admin-users", page],
    queryFn: async () => (await customInstance.get(`/admin/users?page=${page}&limit=${limit}`)).data,
  });

  const statusMutation = useMutation({
    mutationFn: async ({ userId, status }: { userId: string; status: string }) => {
      await customInstance.patch(`/admin/users/${userId}/status?status=${status}`);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-users"] });
      toast.success("User status updated");
      setConfirm(null);
      setSelectedUser(null);
    },
    onError: () => toast.error("Failed to update user status"),
  });

  // User detail with subscription/license/device info
  const { data: userDetail } = useQuery({
    queryKey: ["admin-user-detail", selectedUser?.id],
    queryFn: async () => {
      const res = await customInstance.get(`/admin/users/${selectedUser?.id}/detail`);
      return res.data;
    },
    enabled: !!selectedUser,
  });

  // Available plans for license assignment
  const { data: plans } = useQuery<{ id: string; name: string }[]>({
    queryKey: ["admin-plans-list"],
    queryFn: async () => {
      const res = await customInstance.get("/plans");
      return res.data;
    },
    enabled: !!selectedUser,
  });

  const assignLicenseMutation = useMutation({
    mutationFn: async ({ userId, planId }: { userId: string; planId: string }) => {
      await customInstance.post(`/admin/users/${userId}/assign-license`, { planId });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["admin-user-detail"] });
      queryClient.invalidateQueries({ queryKey: ["admin-licenses"] });
      toast.success("License assigned successfully");
    },
    onError: () => toast.error("Failed to assign license"),
  });

  // v1.31: admin-initiated password reset — generates a one-time reset link,
  // emails it when delivery works, and ALWAYS shows the link for out-of-band
  // delivery (email delivery may be down, e.g. Gmail IP blocks).
  const resetLinkMutation = useMutation({
    mutationFn: async (userId: string) => {
      const res = await customInstance.post(`/admin/users/${userId}/send-reset-link`, {});
      return res.data as { resetUrl: string; emailSent: boolean; expiresAt: string; message: string };
    },
    onSuccess: (data) => {
      void navigator.clipboard?.writeText(data.resetUrl).catch(() => undefined);
      if (data.emailSent) {
        toast.success("Reset link emailed to the user — link also copied to your clipboard", { duration: 8000 });
      } else {
        toast.error("Email delivery failed — link COPIED to clipboard, send it to the user via chat/phone", { duration: 12000 });
      }
      // Always surface the link itself so the admin can copy it again.
      setTimeout(() => {
        if (typeof window !== "undefined") {
          window.prompt("Password reset link (also copied to clipboard). Valid 2 hours, single use:", data.resetUrl);
        }
      }, 300);
      queryClient.invalidateQueries({ queryKey: ["admin-user-detail"] });
    },
    onError: () => toast.error("Failed to generate reset link"),
  });

  const columns: DataTableColumn<User>[] = [
    {
      key: "name", header: "Name", sortable: true,
      cell: (row) => (
        <div>
          <div className="font-medium text-pat-text-primary">{row.full_name}</div>
          <div className="text-xs text-pat-text-muted">{row.email}</div>
        </div>
      ),
    },
    { key: "role", header: "Role", sortable: true, cell: (row) => <span className="text-xs font-medium text-pat-text-secondary">{row.role}</span> },
    { key: "subscription_plan", header: "Plan", sortable: true, cell: (row) => row.subscription_plan
        ? <span className="text-xs font-medium text-pat-text-primary">{row.subscription_plan}</span>
        : <span className="text-xs text-pat-text-muted">no subscription</span> },
    { key: "license_status", header: "License", sortable: true, cell: (row) => row.license_status
        ? <StatusBadge status={row.license_status} />
        : <span className="text-xs text-pat-warning">none</span> },
    { key: "status", header: "Status", sortable: true, cell: (row) => <StatusBadge status={row.status} /> },
    { key: "created_at", header: "Registered", sortable: true, cell: (row) => <span className="text-xs text-pat-text-muted">{row.created_at ? format(new Date(row.created_at), "MMM d, yyyy") : "—"}</span> },
    { key: "last_login_at", header: "Last Login", cell: (row) => <span className="text-xs text-pat-text-muted">{row.last_login_at ? format(new Date(row.last_login_at), "MMM d, yyyy HH:mm") : "Never"}</span> },
    {
      key: "actions", header: "Actions",
      cell: (row) => (
        <div className="flex items-center gap-1">
          <button onClick={() => setSelectedUser(row)} className="text-xs bg-pat-bg-surface-secondary text-pat-text-secondary hover:text-pat-text-primary px-2 py-1 rounded transition-colors" title="View details">
            <IconEye size={14} />
          </button>
          {row.status === "ACTIVE" ? (
            <button onClick={() => setConfirm({ action: "Suspend User", userId: row.id, status: "SUSPENDED" })}
              className="text-xs bg-pat-badge-danger-bg/20 text-pat-badge-danger-text hover:opacity-80 px-2 py-1 rounded transition-colors">
              Suspend
            </button>
          ) : row.status === "PENDING" ? (
            <button onClick={() => setConfirm({ action: "Approve User", userId: row.id, status: "ACTIVE" })}
              className="text-xs bg-pat-badge-success-bg/20 text-pat-badge-success-text hover:opacity-80 px-2 py-1 rounded transition-colors">
              Approve
            </button>
          ) : (
            <button onClick={() => setConfirm({ action: "Activate User", userId: row.id, status: "ACTIVE" })}
              className="text-xs bg-pat-badge-success-bg/20 text-pat-badge-success-text hover:opacity-80 px-2 py-1 rounded transition-colors">
              Activate
            </button>
          )}
        </div>
      ),
    },
  ];

  const users = data?.items ?? [];
  const total = data?.total ?? 0;
  const totalPages = Math.ceil(total / limit);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-bold text-pat-text-primary">User Onboarding</h1>
          <p className="text-sm text-pat-text-secondary mt-1">Manage user accounts, approvals, and status.</p>
        </div>
        <div className="text-sm text-pat-text-muted">Total: <span className="font-semibold text-pat-text-primary">{total}</span> users</div>
      </div>

      {/* Sync-coverage banner: explains why totals differ across the three admin pages */}
      <div className="rounded-lg border border-pat-border bg-pat-bg-surface p-4 text-xs text-pat-text-muted leading-relaxed">
        These three pages count different things, so the numbers are <span className="font-semibold text-pat-text-primary">not supposed to match</span>:
        <span className="text-pat-text-primary"> Users</span> = every registered account;
        <span className="text-pat-text-primary"> Subscriptions</span> = only accounts that started a plan;
        <span className="text-pat-text-primary"> Licenses</span> = issued keys (incl. revoked).
        {(() => {
          const noLicense = (users || []).filter((u) => !u.has_license).length;
          const totalUsers = users?.length ?? 0;
          if (noLicense === 0) return <span> On this page: all {totalUsers} listed users have a license.</span>;
          return <span> <span className="text-pat-warning font-medium">{noLicense} of {totalUsers} users on this page have no active license</span> — issue one from a user row (View → Assign License) or check License Management.</span>;
        })()}
      </div>

      <DataTable data={users} columns={columns} loading={isLoading} error={error as Error | null} onRetry={refetch} pageSize={limit} hidePager />

      {totalPages > 1 && (
        <div className="flex items-center justify-center gap-2 pt-2">
          <button onClick={() => setPage((p) => Math.max(1, p - 1))} disabled={page <= 1} className="px-3 py-1.5 text-xs bg-pat-bg-surface-secondary hover:bg-pat-bg-surface-secondary rounded border border-pat-border disabled:opacity-30">Previous</button>
          <span className="text-xs text-pat-text-muted">Page {page} of {totalPages}</span>
          <button onClick={() => setPage((p) => Math.min(totalPages, p + 1))} disabled={page >= totalPages} className="px-3 py-1.5 text-xs bg-pat-bg-surface-secondary hover:bg-pat-bg-surface-secondary rounded border border-pat-border disabled:opacity-30">Next</button>
        </div>
      )}

      {/* User Detail Drawer */}
      {selectedUser && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={() => setSelectedUser(null)}>
          <div className="bg-pat-bg-surface border border-pat-border rounded-lg shadow-xl max-w-lg w-full mx-4 p-5 max-h-[90vh] overflow-y-auto" onClick={(e) => e.stopPropagation()}>
            <h2 className="text-sm font-semibold text-pat-text-primary mb-4">User Details</h2>
            <div className="space-y-2 text-xs">
              <div className="flex justify-between"><span className="text-pat-text-muted">Name</span><span className="text-pat-text-primary">{selectedUser.full_name}</span></div>
              <div className="flex justify-between"><span className="text-pat-text-muted">Email</span><span className="text-pat-text-primary">{selectedUser.email}</span></div>
              <div className="flex justify-between"><span className="text-pat-text-muted">Role</span><span className="text-pat-text-primary">{selectedUser.role}</span></div>
              <div className="flex justify-between"><span className="text-pat-text-muted">Status</span><StatusBadge status={selectedUser.status} /></div>
              <div className="flex justify-between"><span className="text-pat-text-muted">User ID</span><span className="text-pat-text-muted font-mono">{selectedUser.id.slice(0, 12)}...</span></div>
              <div className="flex justify-between"><span className="text-pat-text-muted">Registered</span><span className="text-pat-text-primary">{selectedUser.created_at ? format(new Date(selectedUser.created_at), "MMM d, yyyy") : "—"}</span></div>
              <div className="flex justify-between"><span className="text-pat-text-muted">Last Login</span><span className="text-pat-text-primary">{selectedUser.last_login_at ? format(new Date(selectedUser.last_login_at), "MMM d, yyyy HH:mm") : "Never"}</span></div>
            </div>

            {/* Trading report downloads (reports module: /reports/admin/reports/trading/:id) */}
            <div className="mt-4 pt-4 border-t border-pat-border">
              <h3 className="text-xs font-semibold text-pat-text-primary mb-2 flex items-center gap-1"><IconDownload size={12} /> Trading Report</h3>
              <div className="flex gap-2">
                {(["pdf", "xlsx", "csv"] as const).map((fmt) => (
                  <a
                    key={fmt}
                    href={`${process.env.NEXT_PUBLIC_API_BASE_URL || "/api/v1"}/reports/admin/reports/trading/${selectedUser.id}?format=${fmt}`}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-xs px-2 py-1 rounded bg-pat-bg-surface-secondary text-pat-text-primary hover:bg-pat-bg-surface-secondary transition-colors uppercase"
                    title={`Download ${fmt.toUpperCase()} trading report`}
                  >
                    {fmt}
                  </a>
                ))}
              </div>
              <p className="text-[10px] text-pat-text-muted mt-1">Requires a linked MT5 agent binding; 404 if the subscriber has no recorded trades.</p>
            </div>

            {/* Subscription */}
            {userDetail && (
              <div className="mt-4 pt-4 border-t border-pat-border">
                <h3 className="text-xs font-semibold text-pat-text-primary mb-2">Subscription</h3>
                {userDetail.subscription ? (
                  <div className="space-y-1 text-xs">
                    <div className="flex justify-between"><span className="text-pat-text-muted">Plan</span><span className="text-pat-text-primary">{userDetail.subscription.plan_name || "—"}</span></div>
                    <div className="flex justify-between"><span className="text-pat-text-muted">Status</span><StatusBadge status={userDetail.subscription.status} /></div>
                    <div className="flex justify-between"><span className="text-pat-text-muted">Auto-Renew</span><span className="text-pat-text-primary">{userDetail.subscription.auto_renew ? "Yes" : "No"}</span></div>
                  </div>
                ) : (
                  <div className="text-xs text-pat-text-muted">No subscription found</div>
                )}
              </div>
            )}

            {/* Account Access — admin password reset (v1.31) */}
            {userDetail && (
              <div className="mt-4 pt-4 border-t border-pat-border">
                <h3 className="text-xs font-semibold text-pat-text-primary mb-2">Account Access</h3>
                <button
                  onClick={() => {
                    if (typeof window !== "undefined" && !window.confirm(`Generate a password reset link for ${selectedUser.email}? The account will also be unlocked. The link is emailed when delivery works — otherwise copy it from the prompt and send it to the user directly.`)) return;
                    resetLinkMutation.mutate(selectedUser.id);
                  }}
                  disabled={resetLinkMutation.isPending}
                  className="w-full px-3 py-1.5 text-xs font-medium bg-pat-bg-surface-secondary text-pat-text-primary border border-pat-border rounded-md hover:bg-pat-bg-surface hover:border-pat-primary disabled:opacity-50">
                  {resetLinkMutation.isPending ? "Generating..." : "Send Reset Link (email + copy)"}
                </button>
                <p className="text-[10px] text-pat-text-muted mt-1">Single-use link, valid 2 hours. Also unlocks a locked account.</p>

                {/* v1.31 unified manual approval — account activation */}
                <div className="mt-3 flex gap-2">
                  <button
                    onClick={async () => {
                      const reason = typeof window !== "undefined" ? window.prompt(`APPROVE account ${selectedUser.email}? It becomes ACTIVE (login allowed, unlock included).\n\nReason (audit log):`, "Admin account activation") : null;
                      if (reason === null) return;
                      try {
                        await approveEntitlement("user", selectedUser.id, "approve", reason);
                        toast.success("Account approved — user is ACTIVE");
                        queryClient.invalidateQueries({ queryKey: ["admin-user-detail"] });
                        queryClient.invalidateQueries({ queryKey: ["admin-users"] });
                      } catch (e) {
                        toast.error("Approval failed: " + (e instanceof Error ? e.message : String(e)));
                      }
                    }}
                    disabled={selectedUser.status === "ACTIVE"}
                    className="flex-1 px-2 py-1.5 text-xs font-medium bg-pat-success/20 text-pat-success rounded-md hover:bg-pat-success/30 disabled:opacity-40">
                    Approve Account
                  </button>
                  <button
                    onClick={async () => {
                      const reason = typeof window !== "undefined" ? window.prompt(`REJECT account ${selectedUser.email}? It becomes SUSPENDED (login blocked, data kept).\n\nReason (audit log):`, "Registration not approved") : null;
                      if (reason === null) return;
                      try {
                        await approveEntitlement("user", selectedUser.id, "reject", reason);
                        toast.success("Account rejected — user is SUSPENDED");
                        queryClient.invalidateQueries({ queryKey: ["admin-user-detail"] });
                        queryClient.invalidateQueries({ queryKey: ["admin-users"] });
                      } catch (e) {
                        toast.error("Rejection failed: " + (e instanceof Error ? e.message : String(e)));
                      }
                    }}
                    disabled={selectedUser.status !== "ACTIVE"}
                    className="flex-1 px-2 py-1.5 text-xs font-medium bg-pat-danger/10 text-pat-danger rounded-md hover:bg-pat-danger/20 disabled:opacity-40">
                    Suspend Account
                  </button>
                </div>
              </div>
            )}

            {/* Licenses */}
            {userDetail && (
              <div className="mt-4 pt-4 border-t border-pat-border">
                <h3 className="text-xs font-semibold text-pat-text-primary mb-2 flex items-center gap-1"><IconKey size={12} /> Licenses</h3>
                {userDetail.licenses && userDetail.licenses.length > 0 ? (
                  <div className="space-y-2">
                    {userDetail.licenses.map((lic: Record<string, unknown>) => (
                      <div key={String(lic.id)} className="flex items-center justify-between rounded-md bg-pat-bg-surface-secondary px-3 py-2">
                        <div>
                          <div className="text-xs text-pat-text-primary font-mono">{String(lic.license_key || "—")}</div>
                          <div className="text-[10px] text-pat-text-muted">{String(lic.plan_name || "—")} · {String(lic.status || "—")}</div>
                        </div>
                        <StatusBadge status={String(lic.status || "unknown")} />
                      </div>
                    ))}
                  </div>
                ) : (
                  <div className="text-xs text-pat-text-muted">No licenses found</div>
                )}

                {/* License Assignment */}
                {(!userDetail.licenses || userDetail.licenses.length === 0) && plans && plans.length > 0 && (
                  <div className="mt-3">
                    <div className="flex gap-2">
                      <select id="plan-select" className="flex-1 rounded-md border border-pat-input-border bg-pat-input-bg px-2 py-1.5 text-xs text-pat-input-text">
                        {plans.map((p) => <option key={p.id} value={p.id}>{p.name}</option>)}
                      </select>
                      <button
                        onClick={() => {
                          const select = document.getElementById("plan-select") as HTMLSelectElement;
                          if (select) assignLicenseMutation.mutate({ userId: selectedUser.id, planId: select.value });
                        }}
                        disabled={assignLicenseMutation.isPending}
                        className="px-3 py-1.5 text-xs font-medium bg-pat-primary text-pat-primary-foreground rounded-md hover:bg-pat-primary-hover disabled:opacity-50">
                        {assignLicenseMutation.isPending ? "Assigning..." : "Assign License"}
                      </button>
                    </div>
                  </div>
                )}
              </div>
            )}

            {/* Devices */}
            {userDetail && userDetail.devices && userDetail.devices.length > 0 && (
              <div className="mt-4 pt-4 border-t border-pat-border">
                <h3 className="text-xs font-semibold text-pat-text-primary mb-2">Devices</h3>
                <div className="space-y-1">
                  {userDetail.devices.map((dev: Record<string, unknown>) => (
                    <div key={String(dev.id)} className="flex items-center justify-between text-xs">
                      <span className="text-pat-text-primary">{String(dev.device_name || "—")}</span>
                      <StatusBadge status={String(dev.connection_status || "unknown")} />
                    </div>
                  ))}
                </div>
              </div>
            )}

            {/* Activations */}
            {userDetail && userDetail.activations && userDetail.activations.length > 0 && (
              <div className="mt-4 pt-4 border-t border-pat-border">
                <h3 className="text-xs font-semibold text-pat-text-primary mb-2">Terminal Activations</h3>
                <div className="space-y-1">
                  {userDetail.activations.map((act: Record<string, unknown>) => (
                    <div key={String(act.id)} className="flex items-center justify-between text-xs">
                      <span className="text-pat-text-primary">{String(act.client_type)}</span>
                      <span className="text-pat-text-muted">{String(act.broker_name || "—")} · {String(act.mt_account_login || "—")}</span>
                    </div>
                  ))}
                </div>
              </div>
            )}

            <div className="flex justify-end gap-2 mt-4">
              {selectedUser.status === "ACTIVE" && (
                <button onClick={() => setConfirm({ action: "Suspend User", userId: selectedUser.id, status: "SUSPENDED" })}
                  className="px-3 py-1.5 text-xs font-medium bg-pat-danger text-white rounded-md hover:opacity-90">Suspend</button>
              )}
              {selectedUser.status !== "ACTIVE" && (
                <button onClick={() => setConfirm({ action: selectedUser.status === "PENDING" ? "Approve User" : "Activate User", userId: selectedUser.id, status: "ACTIVE" })}
                  className="px-3 py-1.5 text-xs font-medium bg-pat-success text-white rounded-md hover:opacity-90">
                  {selectedUser.status === "PENDING" ? "Approve" : "Activate"}
                </button>
              )}
              <button onClick={() => setSelectedUser(null)} className="px-3 py-1.5 text-xs font-medium border border-pat-border-strong rounded-md text-pat-text-secondary">Close</button>
            </div>
          </div>
        </div>
      )}

      <ConfirmDialog
        open={!!confirm}
        title={confirm?.action || ""}
        message={`Are you sure you want to ${confirm?.action.toLowerCase()}? This will ${confirm?.status === "SUSPENDED" ? "revoke all active sessions" : "restore access"}.`}
        confirmLabel={confirm?.action || "Confirm"}
        onConfirm={() => { if (confirm) statusMutation.mutate(confirm); }}
        onCancel={() => setConfirm(null)}
        loading={statusMutation.isPending}
      />
    </div>
  );
}
