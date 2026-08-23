"use client";
import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { IconPlus } from "@tabler/icons-react";
import DataTable, { DataTableColumn } from "@/components/ui/data-table";
import StatusBadge from "@/components/ui/status-badge";
import { format } from "date-fns";
import { fetchPayouts, requestPayout, type Payout, type RequestPayoutPayload } from "@/lib/user-payouts-api";
import { DegradedNote } from "@/components/ui/tabs";

const METHODS = ["BANK_TRANSFER", "PAYPAL", "WISE", "USDT"];

export default function UserPayoutsPage() {
  const queryClient = useQueryClient();
  const [amount, setAmount] = useState("");
  const [method, setMethod] = useState(METHODS[0]);
  const [destination, setDestination] = useState("");

  const { data, isLoading, error, refetch } = useQuery<Payout[]>({
    queryKey: ["user-payouts"],
    queryFn: fetchPayouts,
  });

  const request = useMutation({
    mutationFn: (p: RequestPayoutPayload) => requestPayout(p),
    onSuccess: () => {
      toast.success("Payout request submitted. Awaiting review.");
      setAmount("");
      setDestination("");
      queryClient.invalidateQueries({ queryKey: ["user-payouts"] });
    },
    onError: () => toast.error("Could not submit payout request."),
  });

  const numeric = parseFloat(amount);
  const valid = numeric >= 10 && destination.trim().length > 0;

  const columns: DataTableColumn<Payout>[] = [
    { key: "amount", header: "Amount", cell: (r) => <span className="font-medium text-pat-text-primary">${Number(r.amount || 0).toFixed(2)}</span> },
    { key: "method", header: "Method", cell: (r) => <span className="text-xs text-pat-text-secondary">{r.method}</span> },
    { key: "destination", header: "Destination", cell: (r) => <span className="text-xs text-pat-text-secondary break-all">{r.destination}</span> },
    { key: "status", header: "Status", cell: (r) => <StatusBadge status={r.status} size="sm" /> },
    { key: "created_at", header: "Requested", cell: (r) => <span className="text-xs text-pat-text-muted">{r.created_at ? format(new Date(r.created_at), "MMM d, yyyy") : "—"}</span> },
  ];

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-bold text-pat-text-primary">Payouts</h1>
        <p className="text-sm text-pat-text-secondary mt-1">Request a payout of your earned commissions and review request history.</p>
      </div>

      <div className="rounded-lg border border-pat-border bg-pat-bg-surface p-5 space-y-4">
        <div className="flex items-center gap-2 text-sm font-medium text-pat-text-primary">
          <IconPlus size={18} className="text-pat-success" /> Request a payout
        </div>
        <div className="grid grid-cols-1 gap-3 md:grid-cols-3">
          <div>
            <label className="text-xs text-pat-text-secondary">Amount (USD, min 10)</label>
            <input
              type="number"
              value={amount}
              onChange={(e) => setAmount(e.target.value)}
              placeholder="10.00"
              className="mt-1 w-full rounded-md border border-pat-input-border bg-pat-input-bg px-3 py-2 text-sm text-pat-input-text outline-none focus:border-primary"
            />
          </div>
          <div>
            <label className="text-xs text-pat-text-secondary">Method</label>
            <select
              value={method}
              onChange={(e) => setMethod(e.target.value)}
              className="mt-1 w-full rounded-md border border-pat-input-border bg-pat-input-bg px-3 py-2 text-sm text-pat-input-text outline-none focus:border-primary"
            >
              {METHODS.map((m) => (
                <option key={m} value={m}>{m}</option>
              ))}
            </select>
          </div>
          <div>
            <label className="text-xs text-pat-text-secondary">Destination</label>
            <input
              value={destination}
              onChange={(e) => setDestination(e.target.value)}
              placeholder="Email / account / wallet"
              className="mt-1 w-full rounded-md border border-pat-input-border bg-pat-input-bg px-3 py-2 text-sm text-pat-input-text outline-none focus:border-primary"
            />
          </div>
        </div>
        <button
          onClick={() => valid && request.mutate({ amount: numeric, method, destination: destination.trim() })}
          disabled={!valid || request.isPending}
          className="rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground disabled:opacity-50"
        >
          {request.isPending ? "Submitting…" : "Submit request"}
        </button>
      </div>

      <div>
        <h2 className="mb-3 text-sm font-semibold text-pat-text-primary">Request history</h2>
        <DataTable
          data={data ?? []}
          columns={columns}
          loading={isLoading}
          error={error as Error | null}
          onRetry={() => refetch()}
        />
      </div>

      <DegradedNote>
        Payout requests are reviewed by administrators before approval. This page only reflects requests you have
        submitted; admin approval actions happen outside this view.
      </DegradedNote>
    </div>
  );
}
