"use client";
import { useQuery } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import DataTable, { DataTableColumn } from "@/components/ui/data-table";
import { format } from "date-fns";

interface Commission { id: string; commission_amount: string; status: string; created_at: string; }

export default function UserReferralsPage() {
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

      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
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
          <div className="text-2xl font-bold text-pat-session mt-1">${parseFloat(summary?.pending_amount || "0").toFixed(2)}</div>
        </div>
        <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-5">
          <div className="text-sm text-pat-text-secondary">Entries</div>
          <div className="text-2xl font-bold text-pat-text-primary mt-1">{commissions?.length || 0}</div>
        </div>
      </div>

      <DataTable data={commissions || []} columns={columns} loading={isLoading} error={error as Error|null} onRetry={refetch} />
    </div>
  );
}
