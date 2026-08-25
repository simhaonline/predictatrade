"use client";
import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import { MarketHeader } from "@/components/user-command-center/market-header";
import { MtfPulse } from "@/components/user-command-center/mtf-pulse";
import { IndicatorCards } from "@/components/user-command-center/indicator-cards";
import { SignalPipeline } from "@/components/user-command-center/signal-pipeline";
import { GrowthPanel } from "@/components/user-command-center/growth-panel";
import { MarketContextPanel } from "@/components/market-context/market-context-panel";
import { IconChartLine, IconActivity, IconTrendingUp, IconLayoutGrid } from "@tabler/icons-react";

type Mode = "MARKET" | "TRADING" | "GROWTH" | "COMMAND_CENTER";

interface SnapshotData {
  indicators?: Record<string, number | string | boolean>;
  bars?: Record<string, { close: number; high: number; low: number; open: number; time: string }>;
  source?: string;
  tick?: { bid: number; ask: number; spread: number; time: string };
  session?: { name: string; is_overlap: boolean; is_weekend: boolean };
}

export default function UserLiveDashboardPage() {
  const [mode, setMode] = useState<Mode>("MARKET");

  const { data: snapshot } = useQuery<SnapshotData>({
    queryKey: ["user-command-center-snapshot"],
    queryFn: async () => (await customInstance.get("/market/snapshot")).data,
    refetchInterval: 5000,
  });

  const modes: { id: Mode; label: string; icon: React.ComponentType<{ size?: number; className?: string }> }[] = [
    { id: "MARKET", label: "Market", icon: IconChartLine },
    { id: "TRADING", label: "Trading", icon: IconActivity },
    { id: "GROWTH", label: "Growth", icon: IconTrendingUp },
    { id: "COMMAND_CENTER", label: "Command Center", icon: IconLayoutGrid },
  ];

  return (
    <div className="space-y-4">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-bold text-pat-text-primary">Market Pulse</h1>
          <p className="text-sm text-pat-text-secondary mt-0.5">Real-time market intelligence, signals, and growth overview.</p>
        </div>
      </div>

      {/* Mode tabs */}
      <div className="flex gap-1 rounded-xl border border-pat-border bg-pat-bg-surface p-1">
        {modes.map((m) => (
          <button
            key={m.id}
            onClick={() => setMode(m.id)}
            className={`flex items-center gap-1.5 px-4 py-2 text-xs font-medium rounded-lg transition-all flex-1 justify-center ${
              mode === m.id
                ? "bg-pat-success/15 text-pat-success"
                : "text-pat-text-muted hover:text-pat-text-secondary hover:bg-pat-bg-surface-secondary/50"
            }`}
          >
            <m.icon size={15} />
            {m.label}
          </button>
        ))}
      </div>

      {/* Market Header — always visible in all modes */}
      <MarketHeader />

      {/* MARKET mode */}
      {mode === "MARKET" && (
        <div className="space-y-4">
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
            <MtfPulse snapshot={snapshot} />
            <IndicatorCards snapshot={snapshot} />
          </div>
        </div>
      )}

      {/* TRADING mode */}
      {mode === "TRADING" && (
        <div className="space-y-4">
          <SignalPipeline />
        </div>
      )}

      {/* GROWTH mode */}
      {mode === "GROWTH" && (
        <GrowthPanel />
      )}

      {/* COMMAND_CENTER mode — combined view */}
      {mode === "COMMAND_CENTER" && (
        <div className="space-y-4">
          <div className="grid grid-cols-1 xl:grid-cols-3 gap-4">
            {/* Left: MTF Pulse */}
            <MtfPulse snapshot={snapshot} />
            {/* Center: Signal Pipeline */}
            <div className="xl:col-span-1">
              <SignalPipeline />
            </div>
            {/* Right: Indicators */}
            <IndicatorCards snapshot={snapshot} />
          </div>
          {/* Bottom: Growth summary */}
          <GrowthPanel />
        </div>
      )}

      {/* Data source notice */}
      <div className="text-[10px] text-pat-text-muted text-center">
        Server-authoritative data from Go engine · No browser-side indicator computation
        {snapshot?.source && <span> · Source: {snapshot.source}</span>}
      </div>
      <MarketContextPanel />
    </div>
  );
}
