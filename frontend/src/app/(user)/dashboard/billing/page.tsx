"use client";
import { useQuery } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import DataTable, { DataTableColumn } from "@/components/ui/data-table";
import StatusBadge from "@/components/ui/status-badge";
import { format } from "date-fns";

interface Invoice { id: string; amount: number; status: string; description: string; created_at: string; due_date: string; }

export default function UserBillingPage() {
  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ["user-invoices"],
    queryFn: async () => {
      const res = await customInstance.get("/billing/invoices");
      return (res.data as Invoice[]) || [];
    },
  });

  const columns: DataTableColumn<Invoice>[] = [
    { key: "description", header: "Description", cell: (row) => <span className="text-sm text-pat-text-primary">{row.description || "Invoice"}</span> },
    { key: "amount", header: "Amount", cell: (row) => <span className="text-pat-text-primary font-medium">${parseFloat(String(row.amount || 0)).toFixed(2)}</span> },
    { key: "status", header: "Status", cell: (row) => <StatusBadge status={row.status} /> },
    { key: "created_at", header: "Created", cell: (row) => <span className="text-xs text-pat-text-muted">{row.created_at ? format(new Date(row.created_at), "MMM d, yyyy") : "—"}</span> },
    { key: "due_date", header: "Due", cell: (row) => <span className="text-xs text-pat-text-muted">{row.due_date ? format(new Date(row.due_date), "MMM d, yyyy") : "—"}</span> },
  ];

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-bold text-pat-text-primary">Billing & Subscription</h1>
        <p className="text-sm text-pat-text-secondary mt-1">Your invoices and payment history.</p>
      </div>
      <DataTable data={data || []} columns={columns} loading={isLoading} error={error as Error|null} onRetry={refetch} />
    </div>
  );
}
