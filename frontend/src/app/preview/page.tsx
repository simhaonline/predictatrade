"use client";
import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import { MarketHeader } from "@/components/user-command-center/market-header";
import { MtfPulse } from "@/components/user-command-center/mtf-pulse";
import { IndicatorCards } from "@/components/user-command-center/indicator-cards";
import { SignalPipeline } from "@/components/user-command-center/signal-pipeline";
import { GrowthPanel } from "@/components/user-command-center/growth-panel";
import { GuestPreviewGate } from "@/components/guest-preview/guest-preview-gate";
import { IconChartLine, IconActivity, IconTrendingUp, IconLayoutGrid } from "@tabler/icons-react";

type Mode = "MARKET" | "TRADING" | "GROWTH" | "COMMAND_CENTER";

interface SnapshotData {
  indicators?: Record<string, number | string | boolean>;
  bars?: Record<string, { close: number; high: number; low: number; open: number; time: string }>;
  source?: string;
  tick?: { bid: number; ask: number; spread: number; time: string };
  session?: { name: string; is_overlap: boolean; is_weekend: boolean };
}

/**
 * Guest preview page — renders the FULL live dashboard for unauthenticated
 * visitors, wrapped in the server-enforced GuestPreviewGate. Authenticated
 * users never reach this page (the proxy redirects them to /dashboard/live).
 *
 * This is additive: the existing /dashboard/live (auth-gated) is untouched, and
 * the live MT5/WebSocket pipeline keeps working.
 */
export default function PreviewPage() {
  const [mode, setMode] = useState<Mode>("MARKET");

  const { data: snapshot } = useQuery<SnapshotData>({
    queryKey: ["preview-command-center-snapshot"],
    queryFn: async () => (await customInstance.get("/market/snapshot")).data,
    refetchInterval: 5000,
  });

  const { data: agentsStatus } = useQuery<{
    agents_connected: number;
    agents_online: boolean;
    snapshot_count: number;
    mt4_connected: number;
    mt5_connected: number;
  }>({
    queryKey: ["preview-live-agents"],
    queryFn: async () => (await customInstance.get("/agents/status")).data,
    refetchInterval: 5000,
  });

  const modes: { id: Mode; label: string; icon: React.ComponentType<{ size?: number; className?: string }> }[] = [
    { id: "MARKET", label: "Market", icon: IconChartLine },
    { id: "TRADING", label: "Trading", icon: IconActivity },
    { id: "GROWTH", label: "Growth", icon: IconTrendingUp },
    { id: "COMMAND_CENTER", label: "Command Center", icon: IconLayoutGrid },
  ];

  const dashboard = (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-bold text-pat-text-primary">Market Pulse</h1>
          <p className="text-sm text-pat-text-secondary mt-0.5">Real-time market intelligence, signals, and growth overview.</p>
        </div>
      </div>

      <div className="flex gap-1 rounded-xl border border-pat-border bg-pat-bg-surface p-1">
        {modes.map((m) => (
          <button key={m.id} onClick={() => setMode(m.id)}
            className={`flex items-center gap-1.5 px-4 py-2 text-xs font-medium rounded-lg transition-all flex-1 justify-center ${
              mode === m.id ? "bg-pat-success/15 text-pat-success" : "text-pat-text-muted hover:text-pat-text-secondary hover:bg-pat-bg-surface-secondary/50"
            }`}>
            <m.icon size={15} />{m.label}
          </button>
        ))}
      </div>

      <div className="rounded-xl border border-pat-border bg-pat-bg-surface p-3">
        <div className="flex items-center justify-between flex-wrap gap-2">
          <div className="flex items-center gap-2">
            <span className="text-[11px] uppercase tracking-wide text-pat-text-muted">Your Terminals</span>
            <span className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-lg text-xs font-medium ${
              (agentsStatus?.mt4_connected ?? 0) > 0 ? "bg-pat-success/10 text-pat-success border border-pat-success/20" : "bg-pat-danger/10 text-pat-danger border border-pat-danger/20"
            }`}>
              <span className={`inline-block h-2 w-2 rounded-full bg-pat-danger`} />
              MT4 Offline
            </span>
            <span className={`inline-flex items-center gap-1.5 px-2.5 py-1 rounded-lg text-xs font-medium ${
              (agentsStatus?.mt5_connected ?? 0) > 0 ? "bg-pat-success/10 text-pat-success border border-pat-success/20" : "bg-pat-danger/10 text-pat-danger border border-pat-danger/20"
            }`}>
              <span className={`inline-block h-2 w-2 rounded-full bg-pat-danger`} />
              MT5 Offline
            </span>
          </div>
          <span className="text-[10px] text-pat-text-muted">Terminal link status updates live</span>
        </div>
      </div>

      <MarketHeader />

      {mode === "MARKET" && (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
          <MtfPulse snapshot={snapshot} />
          <IndicatorCards snapshot={snapshot} />
        </div>
      )}
      {mode === "TRADING" && <SignalPipeline />}
      {mode === "GROWTH" && <GrowthPanel />}
      {mode === "COMMAND_CENTER" && (
        <div className="space-y-4">
          <div className="grid grid-cols-1 xl:grid-cols-3 gap-4">
            <MtfPulse snapshot={snapshot} />
            <div className="xl:col-span-1"><SignalPipeline /></div>
            <IndicatorCards snapshot={snapshot} />
          </div>
          <GrowthPanel />
        </div>
      )}

      <div className="text-[10px] text-pat-text-muted text-center">
        Server-authoritative data from Go engine · No browser-side indicator computation
        {snapshot?.source && <span> · Source: {snapshot.source}</span>}
      </div>
    </div>
  );

  return <GuestPreviewGate>{dashboard}</GuestPreviewGate>;
}
