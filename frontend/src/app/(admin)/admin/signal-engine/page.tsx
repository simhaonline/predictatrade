"use client";
import { useQuery } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import StatusBadge from "@/components/ui/status-badge";

// Capital-Tiered Signal Engine (v1.23) — one engine serving MICRO <$500 /
// STANDARD $500–5k / PRO ≥$5k customers with tradeable, suitably-sized
// signals. Sourced live from the Go engine's /api/v1/admin/signal-engine.

interface TierRow {
  tier: string;
  devices: number;
  equity_sum: number;
}

interface DeliveryRow {
  tier: string;
  delivered: number;
  acked: number;
  expired: number;
}

interface SignalRow {
  signal_id: string;
  created_at: string;
  strategy_id: string;
  direction: string;
  entry_price: number;
  stop_loss: number;
  suggested_lot: number;
  eligible_tiers: string[];
  delivered: number;
  acked: number;
}

interface SignalEngineResponse {
  status: string;
  generated_at: string;
  devices_by_tier: TierRow[];
  delivery_24h_by_tier: DeliveryRow[];
  recent_signals: SignalRow[];
  stats_24h: {
    enqueued_24h: number;
    acked_24h: number;
    expired_24h: number;
    pending_24h: number;
    tier_restricted_24h: number;
  };
  last_signal_age_seconds: number;
}

/** Tier chip colours — PAT semantic tokens only. */
const TIER_STYLES: Record<string, string> = {
  MICRO: "bg-pat-warning/15 text-pat-warning border-pat-warning/40",
  STANDARD: "bg-pat-info/15 text-pat-info border-pat-info/40",
  PRO: "bg-pat-success/15 text-pat-success border-pat-success/40",
};

const TIER_LABELS: Record<string, string> = {
  "": "Unknown",
  MICRO: "MICRO · <$500",
  STANDARD: "STANDARD · $500–5k",
  PRO: "PRO · $5k+",
};

function TierChip({ tier }: { tier: string }) {
  const cls = TIER_STYLES[tier] || "bg-pat-text-muted/10 text-pat-text-muted border-pat-border";
  return (
    <span className={`rounded-md border px-2 py-0.5 text-xs font-medium ${cls}`}>
      {TIER_LABELS[tier] || tier}
    </span>
  );
}

export default function AdminSignalEnginePage() {
  const { data, isLoading, isError, error, refetch } = useQuery({
    queryKey: ["signal-engine"],
    // Real endpoint served by the Go realtime engine (admin JWT gated).
    queryFn: async () => (await customInstance.get("/admin/signal-engine")).data as SignalEngineResponse,
    refetchInterval: 15000,
  });

  const stats = data?.stats_24h;

  return (
    <div className="space-y-6">
      <div className="flex items-start justify-between gap-4">
        <div>
          <h1 className="text-xl font-bold text-pat-text-primary">Signal Engine</h1>
          <p className="text-sm text-pat-text-secondary mt-1">
            Capital-tiered signal engine — MICRO &lt;$500 · STANDARD $500–5k · PRO ≥$5k. Every
            signal carries its tier eligibility (min-lot risk vs per-tier caps) and is delivered
            only to matching devices.
          </p>
        </div>
        {data?.generated_at && (
          <span className="text-xs text-pat-text-muted whitespace-nowrap">
            Updated {new Date(data.generated_at).toLocaleTimeString()}
          </span>
        )}
      </div>

      {isLoading && (
        <div className="rounded-lg border border-pat-border bg-pat-bg-surface p-8 text-center text-sm text-pat-text-muted">
          Loading signal engine state…
        </div>
      )}

      {isError && (
        <div className="rounded-lg border border-pat-danger/40 bg-pat-danger/10 p-4 text-sm text-pat-danger">
          Failed to load signal engine: {(error as Error)?.message || "unknown error"}
          <button onClick={() => refetch()} className="ml-3 underline">
            retry
          </button>
        </div>
      )}

      {/* ── 24h pipeline stats ── */}
      {stats && (
        <div className="grid grid-cols-2 gap-4 md:grid-cols-5">
          {[
            { label: "Enqueued (24h)", value: stats.enqueued_24h, tone: "text-pat-text-primary" },
            { label: "Acked (24h)", value: stats.acked_24h, tone: "text-pat-success" },
            { label: "Expired (24h)", value: stats.expired_24h, tone: "text-pat-warning" },
            { label: "Pending", value: stats.pending_24h, tone: "text-pat-text-primary" },
            { label: "Tier-restricted (24h)", value: stats.tier_restricted_24h, tone: "text-pat-info" },
          ].map((c) => (
            <div key={c.label} className="rounded-lg border border-pat-border bg-pat-bg-surface p-4">
              <div className="text-xs text-pat-text-muted">{c.label}</div>
              <div className={`mt-1 text-2xl font-bold ${c.tone}`}>{c.value}</div>
            </div>
          ))}
        </div>
      )}

      <div className="grid gap-6 lg:grid-cols-2">
        {/* ── Devices by capital tier ── */}
        <div className="rounded-lg border border-pat-border bg-pat-bg-surface p-5">
          <h2 className="text-base font-semibold text-pat-text-primary">Devices by Capital Tier</h2>
          <p className="mt-1 text-sm text-pat-text-secondary">
            Classified from each device&apos;s live account equity (heartbeat stream). Unknown =
            EA build predates equity reporting (recompile to activate tiers).
          </p>
          <div className="mt-4 space-y-2">
            {(data?.devices_by_tier ?? []).length === 0 && (
              <div className="text-sm text-pat-text-muted">No exec devices connected.</div>
            )}
            {(data?.devices_by_tier ?? []).map((t) => (
              <div
                key={t.tier}
                className="flex items-center justify-between rounded-md border border-pat-border bg-pat-bg-surface-secondary px-3 py-2"
              >
                <div className="flex items-center gap-3">
                  <TierChip tier={t.tier} />
                  <span className="text-sm text-pat-text-secondary">
                    {t.devices} device{t.devices === 1 ? "" : "s"}
                  </span>
                </div>
                {t.equity_sum > 0 && (
                  <span className="text-sm font-semibold text-pat-text-primary">
                    ${t.equity_sum.toLocaleString(undefined, { maximumFractionDigits: 2 })} equity
                  </span>
                )}
              </div>
            ))}
          </div>
        </div>

        {/* ── Delivery outcomes by tier (24h) ── */}
        <div className="rounded-lg border border-pat-border bg-pat-bg-surface p-5">
          <h2 className="text-base font-semibold text-pat-text-primary">Delivery by Tier (24h)</h2>
          <p className="mt-1 text-sm text-pat-text-secondary">
            Edge-poll delivery outcomes per customer capital category.
          </p>
          <div className="mt-4 space-y-2">
            {(data?.delivery_24h_by_tier ?? []).length === 0 && (
              <div className="text-sm text-pat-text-muted">No deliveries in the last 24h.</div>
            )}
            {(data?.delivery_24h_by_tier ?? []).map((d) => (
              <div
                key={d.tier}
                className="flex items-center justify-between rounded-md border border-pat-border bg-pat-bg-surface-secondary px-3 py-2"
              >
                <TierChip tier={d.tier} />
                <div className="flex gap-4 text-sm text-pat-text-secondary">
                  <span>
                    Delivered <span className="font-semibold text-pat-text-primary">{d.delivered}</span>
                  </span>
                  <span>
                    Acked <span className="font-semibold text-pat-success">{d.acked}</span>
                  </span>
                  {d.expired > 0 && (
                    <span>
                      Expired <span className="font-semibold text-pat-warning">{d.expired}</span>
                    </span>
                  )}
                </div>
              </div>
            ))}
          </div>
          {typeof data?.last_signal_age_seconds === "number" && data.last_signal_age_seconds >= 0 && (
            <p className="mt-4 text-xs text-pat-text-muted">
              Last executable signal {Math.round(data.last_signal_age_seconds / 60)} min ago.
            </p>
          )}
        </div>
      </div>

      {/* ── Recent signals with tier eligibility ── */}
      <div className="rounded-lg border border-pat-border bg-pat-bg-surface p-5">
        <div className="flex items-center justify-between">
          <h2 className="text-base font-semibold text-pat-text-primary">Recent Signals (12h)</h2>
          <span className="text-xs text-pat-text-muted">Tier eligibility per signal</span>
        </div>
        <div className="mt-4 overflow-x-auto">
          <table className="w-full text-left text-sm">
            <thead>
              <tr className="border-b border-pat-border text-xs uppercase tracking-wide text-pat-text-muted">
                <th className="py-2 pr-4">Time</th>
                <th className="py-2 pr-4">Strategy</th>
                <th className="py-2 pr-4">Dir</th>
                <th className="py-2 pr-4">Entry</th>
                <th className="py-2 pr-4">SL</th>
                <th className="py-2 pr-4">Lot</th>
                <th className="py-2 pr-4">Eligible Tiers</th>
                <th className="py-2 pr-4">Delivered</th>
                <th className="py-2">Acked</th>
              </tr>
            </thead>
            <tbody>
              {(data?.recent_signals ?? []).length === 0 && (
                <tr>
                  <td colSpan={9} className="py-4 text-center text-pat-text-muted">
                    No executable signals enqueued in the last 12h (quiet market — profitability
                    gate vetoes unprofitable geometry by design).
                  </td>
                </tr>
              )}
              {(data?.recent_signals ?? []).map((s) => (
                <tr key={s.signal_id} className="border-b border-pat-border/50 text-pat-text-secondary">
                  <td className="py-2 pr-4 whitespace-nowrap">
                    {new Date(s.created_at).toLocaleTimeString()}
                  </td>
                  <td className="py-2 pr-4 font-medium text-pat-text-primary">{s.strategy_id}</td>
                  <td className="py-2 pr-4">
                    <StatusBadge status={s.direction === "BUY" ? "buy" : "sell"} size="sm" />
                  </td>
                  <td className="py-2 pr-4 tabular-nums">{s.entry_price.toFixed(2)}</td>
                  <td className="py-2 pr-4 tabular-nums">{s.stop_loss.toFixed(2)}</td>
                  <td className="py-2 pr-4 tabular-nums">
                    {s.suggested_lot > 0 ? s.suggested_lot.toFixed(2) : "—"}
                  </td>
                  <td className="py-2 pr-4">
                    <div className="flex gap-1">
                      {(s.eligible_tiers ?? []).map((t) => (
                        <TierChip key={t} tier={t} />
                      ))}
                      {(s.eligible_tiers ?? []).length === 0 && (
                        <span className="text-xs text-pat-text-muted">—</span>
                      )}
                    </div>
                  </td>
                  <td className="py-2 pr-4 tabular-nums">{s.delivered}</td>
                  <td className="py-2 tabular-nums text-pat-success">{s.acked}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      <p className="text-xs text-pat-text-muted">
        Tier eligibility is computed from real signal geometry at generation time: minimum-lot risk
        vs each tier&apos;s conservative per-trade cap (reference equity × 2%). Delivery fails open
        only when a device&apos;s tier is unknown (no equity heartbeat yet) — every classified
        device is strictly protected. See docs/strategy/CAPITAL_TIERS.md.
      </p>
    </div>
  );
}