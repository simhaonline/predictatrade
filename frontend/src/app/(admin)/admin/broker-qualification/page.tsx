"use client";
import { useQuery } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import { IconBuildingBank } from "@tabler/icons-react";
import { DegradedBanner } from "@/components/ui/degraded-banner";

interface BrokerRow {
  broker: string;
  server: string;
  platform: string;
  broker_symbol: string;
  typical_spread: number | null;
  spread_p95: number | null;
  contract_size: number | null;
  qualification_result: string;
  last_validated_at: string | null;
  last_observed_at: string | null;
}

interface BrokerData {
  items: BrokerRow[];
  note?: string;
}

const SCHEMA_COLUMNS = [
  "Broker",
  "Server",
  "Platform",
  "Symbol",
  "Typical Spread",
  "Spread P95",
  "Contract Size",
  "Qualification",
  "Last Validated",
];

export default function AdminBrokerQualificationPage() {
  const { data, isLoading, isError, error } = useQuery<BrokerData>({
    queryKey: ["admin-broker-qualification"],
    queryFn: async () => (await customInstance.get("/admin/broker-qualification")).data as BrokerData,
  });

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-bold text-pat-text-primary">Broker Execution Qualification</h1>
        <p className="text-sm text-pat-text-secondary mt-1">
          Measured broker/strategy execution economics. Honest read-only view from the qualification registry.
        </p>
      </div>

      {isLoading && <DegradedBanner>Loading broker qualification registry from backend…</DegradedBanner>}
      {isError && (
        <DegradedBanner>
          Broker execution qualification backend degraded: {error instanceof Error ? error.message : "unable to reach endpoint"}.
          No spread, slippage, margin, latency, or reject-rate values are rendered.
        </DegradedBanner>
      )}

      {data && (
        <div className="bg-pat-card-bg border border-pat-card-border rounded-lg p-4 shadow-sm">
          <h2 className="text-sm font-medium text-pat-text-primary mb-3 flex items-center gap-2">
            <IconBuildingBank size={16} /> Qualification Registry
          </h2>
          {data.note && !data.items.length ? (
            <div className="px-3 py-6 text-center text-xs text-pat-text-muted">{data.note}</div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="text-left text-xs text-pat-text-muted border-b border-pat-border">
                    {SCHEMA_COLUMNS.map((c) => (
                      <th key={c} className="px-3 py-2 font-medium">{c}</th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {data.items.map((r, i) => (
                    <tr key={`${r.broker}-${r.server}-${r.platform}-${i}`} className="border-b border-pat-border/60">
                      <td className="px-3 py-2 text-pat-text-primary">{r.broker}</td>
                      <td className="px-3 py-2 text-pat-text-secondary">{r.server}</td>
                      <td className="px-3 py-2">{r.platform}</td>
                      <td className="px-3 py-2">{r.broker_symbol}</td>
                      <td className="px-3 py-2">{r.typical_spread ?? "—"}</td>
                      <td className="px-3 py-2">{r.spread_p95 ?? "—"}</td>
                      <td className="px-3 py-2">{r.contract_size ?? "—"}</td>
                      <td className="px-3 py-2">{r.qualification_result}</td>
                      <td className="px-3 py-2 text-pat-text-muted">
                        {r.last_validated_at ? new Date(r.last_validated_at).toISOString().slice(0, 19).replace("T", " ") : "—"}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      )}

      <div className="bg-pat-card-bg border border-pat-card-border rounded-lg border-pat-card-border p-4 shadow-sm opacity-80">
        <h2 className="text-sm font-medium text-pat-text-primary mb-3">Add Qualification Run</h2>
        <form className="grid grid-cols-1 md:grid-cols-4 gap-3" onSubmit={(e) => e.preventDefault()}>
          <input disabled placeholder="Broker" className="rounded-md border border-pat-input-border bg-pat-input-bg px-3 py-2 text-sm text-pat-input-text opacity-60" />
          <input disabled placeholder="Strategy class" className="rounded-md border border-pat-input-border bg-pat-input-bg px-3 py-2 text-sm text-pat-input-text opacity-60" />
          <input disabled placeholder="Sample window" className="rounded-md border border-pat-input-border bg-pat-input-bg px-3 py-2 text-sm text-pat-input-text opacity-60" />
          <button disabled className="px-3 py-2 text-sm rounded-md bg-pat-primary text-pat-primary-foreground opacity-50 cursor-not-allowed">
            Run (disabled)
          </button>
        </form>
      </div>
    </div>
  );
}
