"use client";
import { useQuery } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import DataTable, { DataTableColumn } from "@/components/ui/data-table";
import { format } from "date-fns";
import { useState } from "react";
import { IconCopy, IconCheck, IconLink } from "@tabler/icons-react";

interface Commission { id: string; commission_amount: string; status: string; created_at: string; }

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
    queryFn: async () => {
      const res = await customInstance.get("/referrals/network");
      return res.data as { referrals: { child_user_id: string; email: string; full_name: string; level: number; created_at: string }[]; count: number };
    },
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
        {/* Referral Code */}
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

        {/* Referral Signup Link */}
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
          <p className="text-xs text-pat-text-muted mt-2">Share this link with friends — when they sign up, their referral code is automatically filled in and they'll be linked to your account.</p>
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

      {/* Commission History */}
      <DataTable data={commissions || []} columns={columns} loading={isLoading} error={error as Error|null} onRetry={refetch} />
    </div>
  );
}
