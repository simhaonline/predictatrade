"use client";
import { useQuery } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import { getGlobalWs, type WsMessage, type ConnectionState } from "@/lib/websocket";
import { fetchEnginesStatus } from "@/lib/engines-api";
import AdminEngineCards from "@/components/admin/engine-cards";
import { IconUsers, IconReceipt, IconCoin, IconChartBar, IconShield, IconDeviceDesktop, IconActivity, IconBolt, IconServer, IconDatabase, IconBroadcast, IconKey } from "@tabler/icons-react";
import StatusBadge from "@/components/ui/status-badge";
import { format } from "date-fns";
import { useState, useEffect, useRef } from "react";
import { MarketStatusBanner } from "@/components/market-status-banner";
import { strategyLabel } from "@/lib/strategy-labels";

export default function AdminDashboardPage() {
  // --- Data queries ---
  const { data: overview, isLoading } = useQuery({
    queryKey: ["admin-overview"],
    queryFn: async () => (await customInstance.get("/admin/overview")).data,
  });

  const { data: opsState } = useQuery({
    queryKey: ["ops-state"],
    queryFn: async () => (await customInstance.get("/operations/state")).data,
    refetchInterval: 15000,
  });

  // Go engine health — check if signals endpoint returns data
  const { data: engineSignals } = useQuery<{ signals: unknown[] }>({
    queryKey: ["engine-signals"],
    queryFn: async () => (await customInstance.get("/signals")).data,
    refetchInterval: 10000,
  });

  // Go engine market state
  const { data: marketState } = useQuery<Record<string, unknown>>({
    queryKey: ["market-state"],
    queryFn: async () => (await customInstance.get("/market/state")).data,
    refetchInterval: 5000,
  });

  // Go engine agents status
  const { data: agentsStatus } = useQuery<Record<string, unknown>>({
    queryKey: ["agents-status"],
    queryFn: async () => (await customInstance.get("/agents/status")).data,
    refetchInterval: 10000,
  });

  // NestJS health
  const { data: nestHealth } = useQuery<Record<string, unknown>>({
    queryKey: ["nestjs-health"],
    queryFn: async () => (await customInstance.get("/health")).data,
    refetchInterval: 30000,
  });

  // Go engine system-health (public, no auth needed)
  const { data: goSystemHealth } = useQuery<Record<string, unknown>>({
    queryKey: ["go-system-health"],
    queryFn: async () => (await customInstance.get("/system-health")).data,
    refetchInterval: 30000,
  });

  // Backend-authoritative per-engine liveness (Go realtime plane)
  const { data: enginesStatus } = useQuery({
    queryKey: ["engines-status"],
    queryFn: fetchEnginesStatus,
    refetchInterval: 10000,
  });

  // --- WebSocket for live signals ---
  const [liveSignals, setLiveSignals] = useState<{ id: string; direction: string; strategy: string; probability: number; timestamp: string; qualityGrade?: string }[]>([]);
  const [wsState, setWsState] = useState<ConnectionState>('CONNECTING');
  const sigBuffer = useRef<{ id: string; direction: string; strategy: string; probability: number; timestamp: string }[]>([]);
  const ws = getGlobalWs();

  useEffect(() => {
    ws.connect();
    const unsubState = ws.subscribeState((state) => setWsState(state));
    const unsubMsg = ws.subscribe((msg: WsMessage) => {
      if (msg.type === "signal") {
        const s = msg.payload;
        const entry = { id: s.id, direction: s.direction, strategy: s.strategy, probability: s.probability, timestamp: s.timestamp };
        sigBuffer.current = [entry, ...sigBuffer.current].slice(0, 8);
        setLiveSignals(sigBuffer.current);
      }
    });
    return () => { unsubState(); unsubMsg(); };
  }, [ws]);

  // Auto-refresh the Live Signal Pipeline from the REST API.
  // WebSocket signals take priority when they arrive, but REST polling (every 10s)
  // ensures the pipeline stays fresh even when WebSocket is disconnected.
  // Strategy-diverse ordering: one latest signal per strategy first, then fill
  // remaining slots chronologically so every active strategy is visible.
  useEffect(() => {
    const signals = (engineSignals as { signals?: Record<string, unknown>[] })?.signals;
    if (!signals || !Array.isArray(signals) || signals.length === 0) return;
    const directional = signals.filter((s: Record<string, unknown>) => String(s.Direction || s.direction || "") !== "NO-TRADE");
    if (directional.length === 0) {
      // If no directional signals, show the latest signals including NO-TRADE
      const allLatest = signals.slice(0, 8).map((s: Record<string, unknown>) => ({
        id: String(s.ID || s.id || ""),
        direction: String(s.Direction || s.direction || "NO_TRADE"),
        strategy: String(s.StrategyID || s.strategy || s.Strategy || ""),
        probability: Number(s.CalibratedProbability || s.calibratedProbability || s.Probability || 0),
        timestamp: String(s.CreatedAt || s.created_at || s.Timestamp || ""),
        qualityGrade: s.QualityGrade !== undefined ? String(s.QualityGrade) : s.qualityGrade !== undefined ? String(s.qualityGrade) : undefined,
      }));
      sigBuffer.current = allLatest;
      queueMicrotask(() => setLiveSignals(allLatest));
      return;
    }
    // Build strategy-diverse list: one latest per strategy first, then remaining chronologically
    const seen = new Set<string>();
    const perStrategy: Record<string, unknown>[] = [];
    const rest: Record<string, unknown>[] = [];
    for (const s of directional) {
      const sid = String(s.StrategyID || s.strategy || "");
      if (!seen.has(sid)) {
        seen.add(sid);
        perStrategy.push(s);
      } else {
        rest.push(s);
      }
    }
    const combined = [...perStrategy, ...rest].slice(0, 8);
    const refreshed = combined.map((s: Record<string, unknown>) => ({
      id: String(s.ID || s.id || ""),
      direction: String(s.Direction || s.direction || "NO_TRADE"),
      strategy: String(s.StrategyID || s.strategy || s.Strategy || ""),
      probability: Number(s.CalibratedProbability || s.calibratedProbability || s.Probability || 0),
      timestamp: String(s.CreatedAt || s.created_at || s.Timestamp || ""),
    }));
    sigBuffer.current = refreshed;
    queueMicrotask(() => setLiveSignals(refreshed));
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [engineSignals]);

  // --- Derived states (evidence-based, never fake green — prompt.md #111) ---
  const tradingHalted = opsState?.trading_halted ?? false;
  const signalsPaused = opsState?.signals_paused ?? false;
  // Engine is "operational" only if the engines/status endpoint reports at
  // least one running engine with a recent evaluation. A successful empty
  // HTTP response alone proves nothing.
  const engineEvaluating = (() => {
    const engines = enginesStatus?.engines ?? [];
    return engines.some((e) => e.running && e.evaluation_count > 0);
  })();
  const engineAlive = engineEvaluating;
  const agentCount = Number(agentsStatus?.agents_connected ?? 0) || (Array.isArray(agentsStatus?.agents) ? agentsStatus.agents.length : 0);
  const hasAgents = agentCount > 0;
  const wsConnected = wsState === 'CONNECTED';
  const agentsOnline = agentsStatus?.agents_online ?? false;
  // Market-closed awareness: the engine may report market_closed, and even when
  // it reports open the next-open window tells us we are pre-open (no live ticks
  // expected). Treat both as a genuine closure so the UI shows CLOSED/STANDBY
  // instead of falsely alarming STALE/OFFLINE (prompt.md market_closed handling).
  const nextOpenStr = agentsStatus?.next_market_open_utc as string | undefined;
  const nextOpen = nextOpenStr ? new Date(nextOpenStr).getTime() : null;
  const nowMs = Date.now();
  const hoursToOpen = nextOpen !== null ? (nextOpen - nowMs) / 3600000 : null;
  const preOpenClosed = hoursToOpen !== null && hoursToOpen > 0 && hoursToOpen < 48;
  const marketClosed = agentsStatus?.market_closed === true || preOpenClosed;

  // Extract market data from the Go engine response
  const marketData = marketState as Record<string, unknown> | undefined;
  const lastTick = marketData?.LastTick as Record<string, unknown> | undefined;
  const bid = Number(lastTick?.Bid ?? marketData?.Bid ?? 0);
  const ask = Number(lastTick?.Ask ?? marketData?.Ask ?? 0);
  const spread = Number(lastTick?.Spread ?? marketData?.Spread ?? 0);
  const tickSource = (lastTick?.Source as string) ?? (marketData?.Source as string) ?? "";
  // Feed liveness derives from backend tick age (server clock), not from the
  // browser's WebSocket connection state (prompt.md Sections 43, 111).
  const tickTimeStr = (lastTick?.GatewayTimestamp ?? lastTick?.Timestamp) as string | undefined;
  const serverNowStr = enginesStatus?.server_time;
  const serverNow = serverNowStr ? new Date(serverNowStr).getTime() : null;
  const tickAgeSec = tickTimeStr && serverNow !== null ? Math.max(0, (serverNow - new Date(tickTimeStr).getTime()) / 1000) : null;
  const feedState =
    !marketData ? "OFFLINE"
    : marketClosed && (tickAgeSec === null || tickAgeSec >= 300) ? "CLOSED"
    : tickAgeSec !== null && tickAgeSec < 60 ? "LIVE"
    : tickAgeSec !== null && tickAgeSec < 300 ? "DEGRADED"
    : marketClosed ? "CLOSED"
    : "STALE";
  const regime = (marketData?.Regime as Record<string, unknown>) ?? {};
  const currentRegime = (regime.Current as string) ?? "";
  const session = (marketData?.Session as Record<string, unknown>) ?? {};
  const currentSession = (session.CurrentSession as string) ?? "";

  const platformStatusItems = [
    { label: "Trading", status: tradingHalted ? "halted" : "active" },
    { label: "Signals", status: signalsPaused ? "paused" : "active" },
    { label: "Agents", status: agentsOnline ? "online" : marketClosed ? "standby" : "offline" },
    { label: "Market Feed", status: feedState.toLowerCase() },
    { label: "RT Engine", status: engineAlive ? "operational" : marketClosed ? "standby" : "unknown" },
    { label: "Control Plane", status: (nestHealth?.status as string) === "ok" ? "operational" : "unknown" },
    { label: "Database", status: (nestHealth?.database as string) === "healthy" ? "healthy" : "unknown" },
    { label: "WebSocket", status: wsState.toLowerCase() },
  ];

  const metricCards = [
    { label: "Total Users", value: Number(overview?.users?.total || 0), sub: `${Number(overview?.users?.active || 0)} active`, icon: IconUsers, color: "text-pat-info" },
    { label: "Subscriptions", value: Number(overview?.subscriptions?.total || 0), sub: `${Number(overview?.subscriptions?.active || 0)} active · MRR $${Number(overview?.subscriptions?.mrr || 0).toFixed(2)}`, icon: IconReceipt, color: "text-pat-success" },
    { label: "Commissions", value: Number(overview?.commissions?.total_entries || 0), sub: `$${Number(overview?.commissions?.confirmed_amount || 0).toFixed(2)} confirmed`, icon: IconCoin, color: "text-pat-warning" },
    { label: "Payouts", value: Number(overview?.payouts?.total || 0), sub: `${Number(overview?.payouts?.pending || 0)} pending`, icon: IconChartBar, color: "text-pat-info" },
    { label: "Plans", value: Number(overview?.plans?.total || 0), sub: `${Number(overview?.plans?.active || 0)} active`, icon: IconShield, color: "text-pat-success" },
    { label: "Agents", value: agentCount, sub: hasAgents ? "Connected" : "No agents", icon: IconDeviceDesktop, color: hasAgents ? "text-pat-success" : "text-pat-danger" },
  ];

  if (isLoading) {
    return (
      <div className="space-y-4">
        <MarketStatusBanner />
        <h1 className="text-xl font-bold text-pat-text-primary">Real-Time Console</h1>
        <div className="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-6 gap-3">
          {Array.from({ length: 6 }).map((_, i) => (
            <div key={i} className="bg-pat-card-bg border border-pat-card-border rounded-lg p-4 animate-pulse">
              <div className="h-4 bg-pat-bg-surface-secondary rounded w-20 mb-2" />
              <div className="h-8 bg-pat-bg-surface-secondary rounded w-12" />
            </div>
          ))}
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <MarketStatusBanner />
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-bold text-pat-text-primary">Real-Time Console</h1>
        <div className="flex items-center gap-2 text-xs">
          <span className={`inline-block h-2 w-2 rounded-full ${wsConnected ? "bg-pat-success" : wsState === 'RECONNECTING' ? "bg-pat-warning" : "bg-pat-text-muted"}`} />
          <span className="text-pat-text-muted">{wsState}</span>
        </div>
      </div>

      {/* Platform Status Strip */}
      <div className="bg-pat-card-bg border border-pat-card-border rounded-lg p-3 shadow-sm">
        <div className="flex flex-wrap items-center gap-2">
          {platformStatusItems.map((item) => (
            <div key={item.label} className="flex items-center gap-1.5 px-2 py-1 rounded-md bg-pat-bg-surface-secondary">
              <span className="text-xs text-pat-text-muted">{item.label}</span>
              <StatusBadge status={item.status} size="sm" />
            </div>
          ))}
        </div>
      </div>

      {/* Market Data + Agents */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        {/* Live Price */}
        <div className="bg-pat-card-bg border border-pat-card-border rounded-lg p-4 shadow-sm">
          <div className="flex items-center justify-between mb-3">
            <span className="text-sm font-medium text-pat-text-primary">XAUUSD Market</span>
            <span className={`text-[10px] px-2 py-0.5 rounded-full border font-medium ${feedState === 'LIVE' ? "bg-pat-badge-success-bg text-pat-badge-success-text border-pat-badge-success-bg" : feedState === 'DEGRADED' ? "bg-pat-badge-warning-bg text-pat-badge-warning-text border-pat-badge-warning-bg" : "bg-pat-badge-neutral-bg text-pat-badge-neutral-text border-pat-badge-neutral-bg"}`}>
              {feedState}
            </span>
          </div>
          {bid > 0 || ask > 0 ? (
            <div className="grid grid-cols-3 gap-3">
              <div>
                <div className="text-xs text-pat-text-muted">BID</div>
                <div className="text-lg font-mono text-pat-success">{bid.toFixed(2)}</div>
              </div>
              <div>
                <div className="text-xs text-pat-text-muted">ASK</div>
                <div className="text-lg font-mono text-pat-danger">{ask.toFixed(2)}</div>
              </div>
              <div>
                <div className="text-xs text-pat-text-muted">SPREAD</div>
                <div className={`text-lg font-mono ${spread > 0.5 ? "text-pat-warning" : "text-pat-text-primary"}`}>{spread.toFixed(2)}</div>
              </div>
            </div>
          ) : (
            <div className="text-sm text-pat-text-muted py-4 text-center">No live market data</div>
          )}
          <div className="text-xs text-pat-text-muted mt-2">
            {tickTimeStr && <div>Last Tick: {new Date(tickTimeStr).toLocaleTimeString()} UTC ({tickAgeSec !== null ? `${Math.round(tickAgeSec)}s ago` : "—"})</div>}
            {tickSource && <div>Source: {tickSource}</div>}
            {currentSession && <div>Session: {currentSession}</div>}
            {currentRegime && <div>Regime: {currentRegime}</div>}
            {marketClosed && nextOpenStr && <div className="text-pat-warning">Market closed — live feed resumes {new Date(nextOpenStr).toUTCString()}</div>}
          </div>
        </div>

        {/* Agents / Windows Agent */}
        <div className="bg-pat-card-bg border border-pat-card-border rounded-lg p-4 shadow-sm">
          <div className="flex items-center justify-between mb-3">
            <span className="text-sm font-medium text-pat-text-primary">Agents</span>
            <span className={`text-[10px] px-2 py-0.5 rounded-full border font-medium ${agentsOnline ? "bg-pat-badge-success-bg text-pat-badge-success-text border-pat-badge-success-bg" : "bg-pat-badge-danger-bg text-pat-badge-danger-text border-pat-badge-danger-bg"}`}>
              {agentsOnline ? "ONLINE" : "OFFLINE"}
            </span>
          </div>
          {hasAgents ? (
            <div className="space-y-2 text-xs">
              <div className="flex justify-between"><span className="text-pat-text-muted">Connected Agents</span><span className="text-pat-text-primary">{agentCount}</span></div>
              <div className="flex justify-between"><span className="text-pat-text-muted">Agents</span><span className={agentsOnline ? "text-pat-success" : "text-pat-danger"}>{agentsOnline ? "Connected" : "Disconnected"}</span></div>
              <div className="flex justify-between"><span className="text-pat-text-muted">Snapshots</span><span className="text-pat-text-primary">{(agentsStatus?.snapshot_count as number) ?? 0}</span></div>
            </div>
          ) : (
            <div className="text-sm text-pat-text-muted py-4 text-center">
              <IconBroadcast size={24} className="mx-auto mb-2 text-pat-text-muted" />
              No Windows Agent connected
            </div>
          )}
        </div>

        {/* Service Health */}
        <div className="bg-pat-card-bg border border-pat-card-border rounded-lg p-4 shadow-sm">
          <div className="flex items-center justify-between mb-3">
            <span className="text-sm font-medium text-pat-text-primary">Service Health</span>
            <IconServer size={16} className="text-pat-text-secondary" />
          </div>
          <div className="space-y-2 text-xs">
            <div className="flex justify-between items-center">
              <span className="text-pat-text-muted">RT Engine</span>
              <StatusBadge status={engineAlive ? "operational" : "unknown"} size="sm" />
            </div>
            <div className="flex justify-between items-center">
              <span className="text-pat-text-muted">Control Plane</span>
              <StatusBadge status={(nestHealth?.status as string) === "ok" ? "operational" : "unknown"} size="sm" />
            </div>
            <div className="flex justify-between items-center">
              <span className="text-pat-text-muted">Database</span>
              <StatusBadge status={(nestHealth?.database as string) === "healthy" ? "healthy" : "unknown"} size="sm" />
            </div>
            <div className="flex justify-between items-center">
              <span className="text-pat-text-muted">WebSocket</span>
              <StatusBadge status={wsState.toLowerCase()} size="sm" />
            </div>
          </div>
        </div>
      </div>

      {/* Four Strategy Engine Cards (prompt.md Section 44) */}
      <AdminEngineCards />

      {/* Platform Metrics */}
      <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-3">
        {metricCards.map((card) => (
          <div key={card.label} className="bg-pat-card-bg border border-pat-card-border rounded-lg p-4 shadow-sm">
            <div className="flex items-center justify-between mb-2">
              <span className="text-xs text-pat-text-muted">{card.label}</span>
              <card.icon size={18} className={card.color} />
            </div>
            <div className="text-xl font-bold text-pat-text-primary">{typeof card.value === "number" ? card.value.toLocaleString() : card.value}</div>
            {card.sub && <div className="text-xs text-pat-text-muted mt-1">{card.sub}</div>}
          </div>
        ))}
      </div>

      {/* Signal Pipeline + Active Strategies */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <div className="bg-pat-card-bg border border-pat-card-border rounded-lg p-4 shadow-sm">
          <div className="flex items-center justify-between mb-3">
            <h2 className="text-sm font-medium text-pat-text-primary flex items-center gap-2">
              <IconActivity size={16} /> Live Signal Pipeline
            </h2>
            <div className="flex items-center gap-1.5">
              <span className={`inline-block h-1.5 w-1.5 rounded-full ${wsConnected ? "bg-pat-success animate-pulse" : "bg-pat-warning"}`} />
              <span className="text-[10px] text-pat-text-muted">{wsConnected ? "WS Live" : "REST 10s"}</span>
            </div>
          </div>
          {liveSignals.length === 0 ? (
            <div className="text-sm text-pat-text-muted py-4 text-center">
              No signals detected yet. Loading from engine...
            </div>
          ) : (
            <div className="space-y-2">
              {liveSignals.map((s) => (
                <div key={s.id} className="flex items-center justify-between rounded-md bg-pat-bg-surface-secondary px-3 py-2">
                  <div className="flex items-center gap-2">
                    <span className={`text-xs font-bold ${s.direction === "BUY" ? "text-pat-success" : s.direction === "SELL" ? "text-pat-danger" : "text-pat-text-muted"}`}>{s.direction}</span>
                    <span className="text-xs text-pat-text-muted">{s.strategy.replace(/_/g, ' ')}</span>
                  </div>
                  <div className="flex items-center gap-3">
                    {s.qualityGrade && <span className="text-[9px] px-1 py-0.5 rounded-full border bg-pat-info/15 text-pat-info">{s.qualityGrade}</span>}
                    <span className="text-xs text-pat-text-secondary">{(Number(s.probability) * 100).toFixed(1)}%</span>
                    <span className="text-xs text-pat-text-muted">{s.timestamp ? format(new Date(s.timestamp), "HH:mm:ss") : "—"}</span>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Active Strategies */}
        <div className="bg-pat-card-bg border border-pat-card-border rounded-lg p-4 shadow-sm">
          <h2 className="text-sm font-medium text-pat-text-primary mb-3 flex items-center gap-2">
            <IconBolt size={16} /> Active Strategies
          </h2>
          <div className="space-y-2">
            {["STANDARD_SCALPING", "ULTRA_SCALPING", "STANDARD_SWING", "TREND_SWING", "MARNIE_FIB", "ATEN"].map((name) => {
              const isActive = opsState?.active_strategies?.includes(name);
              const stratSignals = ((engineSignals?.signals ?? []) as Record<string, unknown>[]).filter(
                (s: Record<string, unknown>) => String(s.StrategyID || s.strategy || "") === name
              );
              const directionalCount = stratSignals.filter(
                (s: Record<string, unknown>) => String(s.Direction || s.direction || "") !== "NO-TRADE"
              ).length;
              return (
                <div key={strategyLabel(name)} className="flex items-center justify-between rounded-md bg-pat-bg-surface-secondary px-3 py-2">
                  <div className="flex items-center gap-2">
                    <span className="text-xs text-pat-text-primary">{strategyLabel(name)}</span>
                    <span className="text-[10px] text-pat-text-muted">({stratSignals.length} signals, {directionalCount} directional)</span>
                  </div>
                  <StatusBadge status={isActive ? "active" : "inactive"} size="sm" />
                </div>
              );
            })}
          </div>
        </div>
      </div>

      {/* Trading Mode + Operations State */}
      <div className="bg-pat-card-bg border border-pat-card-border rounded-lg p-4 shadow-sm">
        <h2 className="text-sm font-medium text-pat-text-primary mb-3">Platform Operations State</h2>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
          <div className="flex items-center justify-between rounded-md bg-pat-bg-surface-secondary px-3 py-2">
            <span className="text-xs text-pat-text-muted">Trading Mode</span>
            <StatusBadge status={tradingHalted ? "halted" : "active"} size="sm" />
          </div>
          <div className="flex items-center justify-between rounded-md bg-pat-bg-surface-secondary px-3 py-2">
            <span className="text-xs text-pat-text-muted">Signal Generation</span>
            <StatusBadge status={signalsPaused ? "paused" : "active"} size="sm" />
          </div>
          {opsState?.last_updated && (
            <div className="flex items-center justify-between rounded-md bg-pat-bg-surface-secondary px-3 py-2">
              <span className="text-xs text-pat-text-muted">Last Updated</span>
              <span className="text-xs text-pat-text-secondary">{new Date(opsState.last_updated).toLocaleString()}</span>
            </div>
          )}
          <div className="flex items-center justify-between rounded-md bg-pat-bg-surface-secondary px-3 py-2">
            <span className="text-xs text-pat-text-muted">Signals Today</span>
            <span className="text-xs text-pat-text-secondary">{engineSignals?.signals?.length ?? 0}</span>
          </div>
        </div>
      </div>
    </div>
  );
}
