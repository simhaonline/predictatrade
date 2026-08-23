"use client";
import { useQuery } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import DataTable, { DataTableColumn } from "@/components/ui/data-table";
import { format } from "date-fns";
import { useState } from "react";
import { IconCopy, IconCheck, IconLink, IconNetwork, IconWallet } from "@tabler/icons-react";
import Link from "next/link";
import { fetchReferralNetwork, type NetworkReferral } from "@/lib/user-referrals-api";

interface Commission { id: string; commission_amount: string; status: string; created_at: string; level?: number; }

const LEVEL_LABELS = ["L1", "L2", "L3", "L4", "L5"];

export default function UserReferralsPage() {
  const [copiedCode, setCopiedCode] = useState(false);
  const [copiedLink, setCopiedLink] = useState(false);

  const { data: commissions, isLoading, error, refetch } = useQuery({
    queryKey: ["user-commissions"],
    queryFn: async () => {
      const res = await customInstance.get("/commissions");
      return (res.data as Commission[]) || [];
    },
  });

  const { data: summary } = useQuery({
    queryKey: ["user-commission-summary"],
    queryFn: async () => {
      const res = await customInstance.get("/commissions/summary");
      return res.data as { total_amount: string; pending_count: number; pending_amount: string; available_amount: string; paid_amount: string; };
    },
  });

  const { data: referralData } = useQuery({
    queryKey: ["user-referral-code"],
    queryFn: async () => {
      const res = await customInstance.get("/referrals/code");
      return res.data as { code: string };
    },
  });

  const { data: network } = useQuery({
    queryKey: ["user-referral-network"],
    queryFn: fetchReferralNetwork,
  });

  const referralCode = referralData?.code || "";
  const signupUrl = referralCode ? `https://predictatrade.com/register?ref=${referralCode}` : "";

  const copyCode = () => {
    if (!referralCode) return;
    navigator.clipboard.writeText(referralCode);
    setCopiedCode(true);
    setTimeout(() => setCopiedCode(false), 2000);
  };

  const copyLink = () => {
    if (!signupUrl) return;
    navigator.clipboard.writeText(signupUrl);
    setCopiedLink(true);
    setTimeout(() => setCopiedLink(false), 2000);
  };

  const byLevel = (level: number) => (network?.referrals ?? []).filter((r) => r.level === level);

  const columns: DataTableColumn<Commission>[] = [
    { key: "commission_amount", header: "Amount", cell: (row) => <span className="text-pat-text-primary font-medium">${parseFloat(row.commission_amount || "0").toFixed(2)}</span> },
    { key: "status", header: "Status", cell: (row) => <span className={`text-xs px-2 py-1 rounded-full border ${row.status === 'CONFIRMED' ? 'bg-pat-success/10 text-pat-success border-pat-success/20' : 'bg-pat-warning/10 text-pat-session border-pat-warning/20'}`}>{row.status}</span> },
    { key: "created_at", header: "Date", cell: (row) => <span className="text-xs text-pat-text-muted">{row.created_at ? format(new Date(row.created_at), "MMM d, yyyy") : "—"}</span> },
  ];

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-bold text-pat-text-primary">Referral & Earnings</h1>
        <p className="text-sm text-pat-text-secondary mt-1">Your referral stats and commission history.</p>
      </div>

      {/* Referral Code + Share Link Card */}
      <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-5 space-y-4">
        <div>
          <div className="text-sm font-medium text-pat-text-secondary mb-2">Your Referral Code</div>
          <div className="flex items-center gap-3">
            <code className="text-lg font-mono font-bold text-pat-primary bg-pat-info/5 px-3 py-2 rounded-md border border-pat-info/20 flex-1">
              {referralCode || "Loading..."}
            </code>
            <button onClick={copyCode} disabled={!referralCode}
              className="flex items-center gap-1.5 px-3 py-2 rounded-md bg-pat-primary text-pat-primary-foreground hover:bg-pat-primary-hover disabled:opacity-50 transition-colors text-sm font-medium whitespace-nowrap">
              {copiedCode ? <><IconCheck size={16} /> Copied!</> : <><IconCopy size={16} /> Copy Code</>}
            </button>
          </div>
        </div>
        <div>
          <div className="text-sm font-medium text-pat-text-secondary mb-2 flex items-center gap-1.5">
            <IconLink size={14} /> Your Referral Signup Link
          </div>
          <div className="flex items-center gap-3">
            <code className="text-sm font-mono text-pat-text-primary bg-pat-bg-surface-secondary px-3 py-2 rounded-md border border-pat-border flex-1 truncate" style={{ overflow: "hidden", textOverflow: "ellipsis" }}>
              {signupUrl || "Loading..."}
            </code>
            <button onClick={copyLink} disabled={!signupUrl}
              className="flex items-center gap-1.5 px-3 py-2 rounded-md bg-pat-success text-white hover:opacity-90 disabled:opacity-50 transition-colors text-sm font-medium whitespace-nowrap">
              {copiedLink ? <><IconCheck size={16} /> Copied!</> : <><IconLink size={16} /> Copy Link</>}
            </button>
          </div>
          <p className="text-xs text-pat-text-muted mt-2">Share this link with friends — when they sign up, their referral code is automatically filled in and they&apos;ll be linked to your account.</p>
        </div>
      </div>

      {/* Stats Grid */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-5">
          <div className="text-sm text-pat-text-secondary">Total Referrals</div>
          <div className="text-2xl font-bold text-pat-text-primary mt-1">{network?.count || 0}</div>
        </div>
        <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-5">
          <div className="text-sm text-pat-text-secondary">Paid</div>
          <div className="text-2xl font-bold text-pat-text-primary mt-1">${parseFloat(summary?.paid_amount || "0").toFixed(2)}</div>
        </div>
        <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-5">
          <div className="text-sm text-pat-text-secondary">Available</div>
          <div className="text-2xl font-bold text-pat-success mt-1">${parseFloat(summary?.available_amount || "0").toFixed(2)}</div>
        </div>
        <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-5">
          <div className="text-sm text-pat-text-secondary">Pending</div>
          <div className="text-2xl font-bold text-pat-warning mt-1">${parseFloat(summary?.pending_amount || "0").toFixed(2)}</div>
        </div>
        <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-5">
          <div className="text-sm text-pat-text-secondary">Total Earned</div>
          <div className="text-2xl font-bold text-pat-text-primary mt-1">${parseFloat(summary?.total_amount || "0").toFixed(2)}</div>
        </div>
        <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-5">
          <div className="text-sm text-pat-text-secondary">Entries</div>
          <div className="text-2xl font-bold text-pat-text-primary mt-1">{commissions?.length || 0}</div>
        </div>
      </div>

      {/* Downline network tree (L1-L5) */}
      <div>
        <h2 className="mb-3 text-sm font-semibold text-pat-text-primary flex items-center gap-1.5"><IconNetwork size={16} /> Downline network</h2>
        {!network || network.referrals.length === 0 ? (
          <div className="rounded-lg border border-pat-border bg-pat-bg-surface p-6 text-center text-sm text-pat-text-muted">
            No referrals in your downline yet. Share your link to start building your network.
          </div>
        ) : (
          <div className="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-3">
            {LEVEL_LABELS.map((label, idx) => {
              const refs = byLevel(idx + 1);
              return (
                <div key={label} className="rounded-lg border border-pat-border bg-pat-bg-surface p-4">
                  <div className="mb-2 flex items-center justify-between">
                    <span className="text-xs font-semibold text-pat-text-primary">{label}</span>
                    <span className="text-[10px] text-pat-text-muted">{refs.length} member{refs.length === 1 ? "" : "s"}</span>
                  </div>
                  {refs.length === 0 ? (
                    <div className="text-[11px] text-pat-text-muted">No referrals at this level.</div>
                  ) : (
                    <div className="space-y-2">
                      {refs.map((r: NetworkReferral) => (
                        <div key={r.child_user_id} className="rounded-md border border-pat-border/60 bg-pat-bg-surface-secondary/20 px-3 py-2">
                          <div className="text-xs font-medium text-pat-text-primary">{r.full_name || r.email}</div>
                          <div className="text-[10px] text-pat-text-muted">
                            {r.email}{r.created_at ? ` · joined ${format(new Date(r.created_at), "MMM d, yyyy")}` : ""}
                          </div>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        )}
        <p className="text-[10px] text-pat-text-muted mt-2">
          The referral network endpoint returns a flat list with a <code>level</code> attribute; it is grouped here into L1–L5 tiers. Multi-tier commission mapping is rendered from available data only.
        </p>
      </div>

      {/* Payout link */}
      <Link href="/dashboard/payouts" className="flex items-center justify-between rounded-lg border border-pat-border bg-pat-bg-surface p-4 hover:border-pat-border/80 transition-colors">
        <div className="flex items-center gap-3">
          <div className="flex items-center justify-center w-9 h-9 rounded-lg bg-pat-success/10"><IconWallet size={18} className="text-pat-success" /></div>
          <div>
            <div className="text-sm font-medium text-pat-text-primary">Request a payout</div>
            <div className="text-xs text-pat-text-muted">Withdraw your available commission earnings.</div>
          </div>
        </div>
        <span className="text-xs text-pat-info">Open →</span>
      </Link>

      {/* Commission History */}
      <DataTable data={commissions || []} columns={columns} loading={isLoading} error={error as Error|null} onRetry={refetch} />
    </div>
  );
}
