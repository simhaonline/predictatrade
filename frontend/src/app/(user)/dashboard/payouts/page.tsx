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

type Method = "BANK_TRANSFER" | "USDT";
const METHODS: Method[] = ["BANK_TRANSFER", "USDT"];
const USDT_NETWORKS = ["TRC20", "ERC20", "BEP20"];

export default function UserPayoutsPage() {
  const queryClient = useQueryClient();
  const [amount, setAmount] = useState("");
  const [method, setMethod] = useState<Method>("BANK_TRANSFER");

  // Bank Transfer fields
  const [bankName, setBankName] = useState("");
  const [accountHolder, setAccountHolder] = useState("");
  const [accountNumber, setAccountNumber] = useState("");
  const [swiftBic, setSwiftBic] = useState("");
  const [country, setCountry] = useState("");

  // USDT fields
  const [usdtNetwork, setUsdtNetwork] = useState("TRC20");
  const [walletAddress, setWalletAddress] = useState("");

  const { data, isLoading, error, refetch } = useQuery<Payout[]>({
    queryKey: ["user-payouts"],
    queryFn: fetchPayouts,
  });

  const request = useMutation({
    mutationFn: (p: RequestPayoutPayload) => requestPayout(p),
    onSuccess: () => {
      toast.success("Payout request submitted. Awaiting review.");
      resetForm();
      queryClient.invalidateQueries({ queryKey: ["user-payouts"] });
    },
    onError: () => toast.error("Could not submit payout request."),
  });

  const resetForm = () => {
    setAmount("");
    setBankName("");
    setAccountHolder("");
    setAccountNumber("");
    setSwiftBic("");
    setCountry("");
    setUsdtNetwork("TRC20");
    setWalletAddress("");
  };

  const numeric = parseFloat(amount);
  const destination = method === "BANK_TRANSFER" ? accountNumber.trim() : walletAddress.trim();

  let details: Record<string, string> | undefined;
  if (method === "BANK_TRANSFER") {
    details = {
      bank_name: bankName.trim(),
      account_holder: accountHolder.trim(),
      account_number: accountNumber.trim(),
      swift_bic: swiftBic.trim(),
      country: country.trim(),
    };
  } else {
    details = { network: usdtNetwork, wallet_address: walletAddress.trim() };
  }

  const bankValid =
    bankName.trim().length > 0 &&
    accountHolder.trim().length > 0 &&
    accountNumber.trim().length > 0 &&
    swiftBic.trim().length > 0 &&
    country.trim().length > 0;
  const usdtValid =
    USDT_NETWORKS.includes(usdtNetwork) &&
    (/^0x[a-fA-F0-9]{40}$/.test(walletAddress.trim()) || /^T[1-9A-HJ-NP-Za-km-z]{33}$/.test(walletAddress.trim()));
  const valid = numeric >= 50 && (method === "BANK_TRANSFER" ? bankValid : usdtValid);

  const columns: DataTableColumn<Payout>[] = [
    { key: "amount", header: "Amount", cell: (r) => <span className="font-medium text-pat-text-primary">${Number(r.amount || 0).toFixed(2)}</span> },
    { key: "method", header: "Method", cell: (r) => <span className="text-xs text-pat-text-secondary">{r.method === "BANK_TRANSFER" ? "Bank Transfer" : r.method === "USDT" ? "USDT (Crypto)" : r.method}</span> },
    { key: "destination", header: "Destination", cell: (r) => <span className="text-xs text-pat-text-secondary break-all">{r.destination}</span> },
    { key: "status", header: "Status", cell: (r) => <StatusBadge status={r.status} size="sm" /> },
    { key: "created_at", header: "Requested", cell: (r) => <span className="text-xs text-pat-text-muted">{r.created_at ? format(new Date(r.created_at), "MMM d, yyyy") : "—"}</span> },
  ];

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-bold text-pat-text-primary">Payouts</h1>
        <p className="text-sm text-pat-text-secondary mt-1">Request a payout of your earned commissions via Bank Transfer or USDT.</p>
      </div>

      <div className="rounded-lg border border-pat-border bg-pat-bg-surface p-5 space-y-4">
        <div className="flex items-center gap-2 text-sm font-medium text-pat-text-primary">
          <IconPlus size={18} className="text-pat-success" /> Request a payout
        </div>

        <div className="flex gap-2">
          {METHODS.map((m) => (
            <button
              key={m}
              type="button"
              onClick={() => setMethod(m)}
              className={`rounded-lg px-4 py-2 text-sm border transition-colors ${
                method === m
                  ? "border-primary bg-primary/10 text-pat-text-primary font-medium"
                  : "border-pat-border bg-pat-bg-surface text-pat-text-secondary hover:text-pat-text-primary"
              }`}
            >
              {m === "BANK_TRANSFER" ? "Bank Transfer" : "USDT (Crypto)"}
            </button>
          ))}
        </div>

        <div className="grid grid-cols-1 gap-3 md:grid-cols-3">
          <div>
            <label className="text-xs text-pat-text-secondary">Amount (USD, min 50)</label>
            <input
              type="number"
              value={amount}
              onChange={(e) => setAmount(e.target.value)}
              placeholder="50.00"
              className="mt-1 w-full rounded-md border border-pat-input-border bg-pat-input-bg px-3 py-2 text-sm text-pat-input-text outline-none focus:border-primary"
            />
          </div>
        </div>

        {method === "BANK_TRANSFER" ? (
          <div className="grid grid-cols-1 gap-3 md:grid-cols-3">
            <div>
              <label className="text-xs text-pat-text-secondary">Bank Name</label>
              <input
                value={bankName}
                onChange={(e) => setBankName(e.target.value)}
                placeholder="e.g. HDFC Bank"
                className="mt-1 w-full rounded-md border border-pat-input-border bg-pat-input-bg px-3 py-2 text-sm text-pat-input-text outline-none focus:border-primary"
              />
            </div>
            <div>
              <label className="text-xs text-pat-text-secondary">Account Holder Name</label>
              <input
                value={accountHolder}
                onChange={(e) => setAccountHolder(e.target.value)}
                placeholder="Full name as on the account"
                className="mt-1 w-full rounded-md border border-pat-input-border bg-pat-input-bg px-3 py-2 text-sm text-pat-input-text outline-none focus:border-primary"
              />
            </div>
            <div>
              <label className="text-xs text-pat-text-secondary">Account Number / IBAN</label>
              <input
                value={accountNumber}
                onChange={(e) => setAccountNumber(e.target.value)}
                placeholder="Account number or IBAN"
                className="mt-1 w-full rounded-md border border-pat-input-border bg-pat-input-bg px-3 py-2 text-sm text-pat-input-text outline-none focus:border-primary"
              />
            </div>
            <div>
              <label className="text-xs text-pat-text-secondary">SWIFT / BIC</label>
              <input
                value={swiftBic}
                onChange={(e) => setSwiftBic(e.target.value.toUpperCase())}
                placeholder="e.g. HDFCINBB"
                className="mt-1 w-full rounded-md border border-pat-input-border bg-pat-input-bg px-3 py-2 text-sm text-pat-input-text outline-none focus:border-primary"
              />
            </div>
            <div>
              <label className="text-xs text-pat-text-secondary">Country</label>
              <input
                value={country}
                onChange={(e) => setCountry(e.target.value)}
                placeholder="Bank country"
                className="mt-1 w-full rounded-md border border-pat-input-border bg-pat-input-bg px-3 py-2 text-sm text-pat-input-text outline-none focus:border-primary"
              />
            </div>
          </div>
        ) : (
          <div className="grid grid-cols-1 gap-3 md:grid-cols-3">
            <div>
              <label className="text-xs text-pat-text-secondary">Network</label>
              <select
                value={usdtNetwork}
                onChange={(e) => setUsdtNetwork(e.target.value)}
                className="mt-1 w-full rounded-md border border-pat-input-border bg-pat-input-bg px-3 py-2 text-sm text-pat-input-text outline-none focus:border-primary"
              >
                {USDT_NETWORKS.map((n) => (
                  <option key={n} value={n}>{n}</option>
                ))}
              </select>
            </div>
            <div className="md:col-span-2">
              <label className="text-xs text-pat-text-secondary">USDT Wallet Address ({usdtNetwork})</label>
              <input
                value={walletAddress}
                onChange={(e) => setWalletAddress(e.target.value)}
                placeholder={usdtNetwork === "TRC20" ? "T…" : "0x…"}
                className="mt-1 w-full rounded-md border border-pat-input-border bg-pat-input-bg px-3 py-2 text-sm text-pat-input-text outline-none focus:border-primary break-all"
              />
              <p className="mt-1 text-[11px] text-pat-warning">
                Double-check the network and address — crypto payouts are irreversible once sent.
              </p>
            </div>
          </div>
        )}

        <button
          onClick={() =>
            valid && request.mutate({ amount: numeric, method, destination, details })
          }
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
