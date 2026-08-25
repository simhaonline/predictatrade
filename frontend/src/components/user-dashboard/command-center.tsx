"use client";
import { useState, useEffect } from "react";
import { useQuery } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import { getGlobalWs, type ConnectionState } from "@/lib/websocket";
import { IconChartBar, IconCoins, IconCheck, IconHourglass } from "@tabler/icons-react";
import { format } from "date-fns";

// ─── Types ──────────────────────────────────────────────────────────────────
interface MarketSnapshot {
  source?: string;
  symbol?: string;
  timestamp?: string;
  broker?: string;
  tick?: { bid: number; ask: number; spread: number; spread_points: number; volume: number; time: string };
  indicators?: Record<string, number | string | boolean>;
  session?: { name: string; is_overlap: boolean; is_weekend: boolean; gmt_hour: number };
  vwap?: { session_vwap: number };
  positions?: { total_positions: number; buy_count: number; sell_count: number; total_lots: number; floating_profit: number };
  account_info?: { balance: number; equity: number; margin: number; free_margin: number; profit: number; currency: string; leverage: number; server: string };
  symbol_info?: { digits: number; point: number; spread: number; stops_level: number; freeze_level: number; contract_size: number; min_lot: number; max_lot: number; lot_step: number; swap_long: number; swap_short: number; tick_value: number; tick_size: number };
}

interface SignalRecord {
  ID: string;
  StrategyID: string;
  Direction: string;
  Status: string;
  Grade: string;
  RawScore: string;
  CalibratedProbability: string;
  EntryPrice: string;
  StopLoss: string;
  TP1: string;
  TP2: string;
  TP3: string;
  GrossRRTP1: string;
  NetRRTP1: string;
  Regime: string;
  Session: string;
  CreatedAt: string;
  ExpiresAt: string;
  Executable: boolean;
  SignalClass: string;
  Evidence?: Array<{ pillar: string; feature: string; direction: string; contribution: string }>;
  ReasonCodes?: string[];
}

type Mode = "MARKET" | "TRADING" | "GROWTH" | "COMMAND_CENTER";

// ─── Main Command Center Component ────────────────────────────────────────────
export function CommandCenter({ isAdmin = false }: { isAdmin?: boolean }) {
  void isAdmin;
  const [mode, setMode] = useState<Mode>("MARKET");
  const [wsState, setWsState] = useState<ConnectionState>("CONNECTING");
  const ws = getGlobalWs();

  // Go engine: market snapshot (auto-refresh every 3s)
  const { data: snapshot } = useQuery<MarketSnapshot>({
    queryKey: ["command-center-snapshot"],
    queryFn: async () => (await customInstance.get("/market/snapshot")).data,
    refetchInterval: 3000,
  });

  // Go engine: signals (auto-refresh every 5s)
  const { data: signalsData } = useQuery<{ signals: SignalRecord[] }>({
    queryKey: ["command-center-signals"],
    queryFn: async () => (await customInstance.get("/signals")).data,
    refetchInterval: 5000,
  });

  // Go engine: market state (for regime, MTF, structure)
  const { data: marketState } = useQuery<Record<string, unknown>>({
    queryKey: ["command-center-state"],
    queryFn: async () => (await customInstance.get("/market/state")).data,
    refetchInterval: 5000,
  });

  // NestJS: user subscription / commission / referral data
  const { data: subscriptions } = useQuery({
    queryKey: ["cc-subscriptions"],
    queryFn: async () => (await customInstance.get("/subscriptions")).data,
    refetchInterval: 30000,
  });

  const { data: commissionSummary } = useQuery<{
    total_amount: string; pending_count: string; confirmed_count: string;
    pending_amount: string; confirmed_amount: string;
  }>({
    queryKey: ["cc-commission"],
    queryFn: async () => (await customInstance.get("/commissions/summary")).data,
    refetchInterval: 30000,
  });

  const { data: referralNetwork } = useQuery({
    queryKey: ["cc-referrals"],
    queryFn: async () => (await customInstance.get("/referrals/network")).data,
    refetchInterval: 30000,
  });

  // WebSocket for live tick updates
  useEffect(() => {
    ws.connect();
    const unsubState = ws.subscribeState((s) => setWsState(s));
    return () => { unsubState(); };
  }, [ws]);

  const modes: { id: Mode; label: string }[] = [
    { id: "MARKET", label: "Market" },
    { id: "TRADING", label: "Trading" },
    { id: "GROWTH", label: "Growth" },
    { id: "COMMAND_CENTER", label: "Command Center" },
  ];

  return (
    <div className="space-y-4">
      {/* Mode tabs */}
      <div className="flex gap-1 rounded-lg border border-pat-border bg-pat-bg-surface p-1">
        {modes.map((m) => (
          <button
            key={m.id}
            onClick={() => setMode(m.id)}
            className={`flex-1 px-4 py-2 text-sm font-medium rounded-md transition-all ${
              mode === m.id
                ? "bg-pat-success/15 text-pat-success"
                : "text-pat-text-muted hover:text-pat-text-secondary hover:bg-pat-bg-surface-secondary/50"
            }`}
          >
            {m.label}
          </button>
        ))}
      </div>

      {/* Global Market Header — always visible (SOW 169.1) */}
      <GlobalMarketHeader snapshot={snapshot} wsState={wsState} marketState={marketState} />

      {/* Mode content */}
      {mode === "MARKET" && <MarketMode snapshot={snapshot} marketState={marketState} />}
      {mode === "TRADING" && <TradingMode signals={signalsData?.signals ?? []} snapshot={snapshot} />}
      {mode === "GROWTH" && <GrowthMode subscriptions={subscriptions} commission={commissionSummary} referrals={referralNetwork} />}
      {mode === "COMMAND_CENTER" && (
        <CommandCenterMode
          snapshot={snapshot}
          signals={signalsData?.signals ?? []}
          subscriptions={subscriptions}
          commission={commissionSummary}
        />
      )}
    </div>
  );
}

// ─── Global Market Header (SOW 169.1) ────────────────────────────────────────
function GlobalMarketHeader({ snapshot, wsState, marketState }: {
  snapshot?: MarketSnapshot;
  wsState: ConnectionState;
  marketState?: Record<string, unknown>;
}) {
  const tick = snapshot?.tick;
  const session = snapshot?.session;
  const indicators = snapshot?.indicators || {};
  const regimeRaw = (marketState as Record<string, unknown>)?.Regime as Record<string, unknown> | undefined;
  const regime = String(regimeRaw?.Current || "");
  const bid = tick?.bid ?? 0;
  const ask = tick?.ask ?? 0;
  const spread = tick?.spread ?? 0;
  const atr = Number(indicators.atr || 0);
  const adx = Number(indicators.adx || 0);
  const rsi = Number(indicators.rsi || 0);
  const source = snapshot?.source || "—";
  const tickTime = tick?.time;
  const tickAgeSec = tickTime ? Math.max(0, (Date.now() - new Date(tickTime).getTime()) / 1000) : null;
  const snack = !tick ? "UNKNOWN"
    : tickAgeSec !== null && tickAgeSec < 60 ? "LIVE"
    : tickAgeSec !== null && tickAgeSec < 300 ? "DEGRADED"
    : "STALE";

  return (
    <div className="rounded-lg border border-pat-border bg-pat-bg-surface p-3">
      <div className="flex flex-wrap items-center justify-between gap-3">
        {/* Left: Symbol + price */}
        <div className="flex items-center gap-4">
          <div className="flex items-center gap-1.5">
            <span title={wsState === "CONNECTED" ? "WS Connected" : wsState === "RECONNECTING" ? "WS Retry" : "WS Off"} className={`inline-block h-2 w-2 rounded-full ${wsState === "CONNECTED" ? "bg-pat-success animate-pulse" : wsState === "RECONNECTING" ? "bg-pat-warning" : "bg-pat-danger"}`} />
            <span className="text-sm font-bold text-pat-text-primary">XAUUSD</span>
          </div>
          <div className="flex items-center gap-3">
            <div>
              <span className="text-[10px] text-pat-text-muted uppercase">Bid</span>
              <span className="text-sm font-mono text-pat-success ml-1 tabular-nums">{bid.toFixed(2)}</span>
            </div>
            <div>
              <span className="text-[10px] text-pat-text-muted uppercase">Ask</span>
              <span className="text-sm font-mono text-pat-danger ml-1 tabular-nums">{ask.toFixed(2)}</span>
            </div>
            <div>
              <span className="text-[10px] text-pat-text-muted uppercase">Spread</span>
              <span className={`text-sm font-mono ml-1 tabular-nums ${spread > 0.5 ? "text-pat-warning" : "text-pat-text-primary"}`}>{spread.toFixed(2)}</span>
            </div>
          </div>
        </div>

        {/* Right: Market state bar */}
        <div className="flex items-center gap-3 text-xs">
          {regime && (
            <div className="flex items-center gap-1">
              <span className="text-pat-text-muted">Regime:</span>
              <span className="text-pat-text-secondary font-medium">{regime}</span>
            </div>
          )}
          {session?.name && (
            <div className="flex items-center gap-1">
              <span className="text-pat-text-muted">Session:</span>
              <span className="text-pat-text-secondary font-medium">{session.name}</span>
              {session.is_overlap && <span className="text-[10px] text-pat-success">OVERLAP</span>}
            </div>
          )}
          {atr > 0 && (
            <div className="flex items-center gap-1">
              <span className="text-pat-text-muted">ATR:</span>
              <span className="text-pat-text-secondary font-mono">{atr.toFixed(2)}</span>
            </div>
          )}
          {adx > 0 && (
            <div className="flex items-center gap-1">
              <span className="text-pat-text-muted">ADX:</span>
              <span className={`font-mono ${adx > 25 ? "text-pat-success" : "text-pat-text-secondary"}`}>{adx.toFixed(1)}</span>
            </div>
          )}
          {rsi > 0 && (
            <div className="flex items-center gap-1">
              <span className="text-pat-text-muted">RSI:</span>
              <span className={`font-mono ${rsi > 70 ? "text-pat-danger" : rsi < 30 ? "text-pat-success" : "text-pat-text-secondary"}`}>{rsi.toFixed(1)}</span>
            </div>
          )}
          <div className="flex items-center gap-1">
            <span className="text-pat-text-muted">Feed:</span>
            <span className={`text-[10px] px-1.5 py-0.5 rounded-full border font-medium ${
              snack === "LIVE" ? "bg-pat-badge-success-bg text-pat-badge-success-text border-pat-badge-success-bg" :
              snack === "DEGRADED" ? "bg-pat-badge-warning-bg text-pat-badge-warning-text border-pat-badge-warning-bg" :
              snack === "STALE" ? "bg-pat-badge-neutral-bg text-pat-badge-neutral-text border-pat-badge-neutral-bg" :
              "bg-pat-badge-info-bg text-pat-badge-info-text border-pat-badge-info-bg"
            }`}>{snack}</span>
          </div>
          <span className="text-[10px] text-pat-text-muted">{source.replace("+LOCAL_COMPUTE", "")}</span>
        </div>
      </div>
    </div>
  );
}

// ─── MARKET Mode (SOW 170) ──────────────────────────────────────────────────
function MarketMode({ snapshot, marketState }: { snapshot?: MarketSnapshot; marketState?: Record<string, unknown> }) {
  const indicators = snapshot?.indicators || {};
  const mtf = ((marketState as Record<string, unknown>)?.MTF as { Score?: number; States?: Record<string, number> } | undefined);
  const structure = ((marketState as Record<string, unknown>)?.Structure as { CurrentTrend?: string; LastBOS?: { Direction?: string; Price?: string }; LastCHoCH?: { Direction?: string } } | undefined);
  const candles = ((marketState as Record<string, unknown>)?.Candles as Record<string, { Close?: string }>) || {} || {};

  // Multi-timeframe data
  const timeframes = ["M1", "M5", "M15", "H1", "H4", "D1"];
  const mtfStates = mtf?.States || {};

  return (
    <div className="space-y-4">
      {/* Multi-timeframe pulse (SOW 170.1) */}
      <div className="rounded-xl border border-pat-border bg-pat-bg-surface p-4">
        <h3 className="text-sm font-semibold text-pat-text-primary mb-3">Multi-Timeframe Pulse</h3>
        <div className="grid grid-cols-3 md:grid-cols-6 gap-2">
          {timeframes.map((tf) => {
            const state = mtfStates[tf] as number | undefined;
            const candle = candles[tf];
            const close = candle ? Number(candle.Close) : 0;
            const dir = state === 1 ? "BULL" : state === -1 ? "BEAR" : "FLAT";
            const color = state === 1 ? "text-pat-success" : state === -1 ? "text-pat-danger" : "text-pat-text-muted";
            return (
              <div key={tf} className="rounded-lg bg-pat-bg-surface-secondary/30 border border-pat-border/50 p-2 text-center">
                <div className="text-xs font-semibold text-pat-text-primary">{tf}</div>
                <div className={`text-xs font-bold ${color}`}>{dir}</div>
                {close > 0 && <div className="text-[10px] text-pat-text-muted font-mono mt-0.5">{close.toFixed(1)}</div>}
              </div>
            );
          })}
        </div>
        {mtf?.Score !== undefined && (
          <div className="mt-2 text-xs text-pat-text-muted text-center">
            MTF Alignment Score: <span className={mtf.Score > 20 ? "text-pat-success" : mtf.Score < -20 ? "text-pat-danger" : "text-pat-text-secondary"}>{mtf.Score.toFixed(1)}</span>
          </div>
        )}
      </div>

      {/* Indicator Intelligence Cards (SOW 170.2) */}
      <div className="rounded-xl border border-pat-border bg-pat-bg-surface p-4">
        <h3 className="text-sm font-semibold text-pat-text-primary mb-3">Indicator Intelligence</h3>
        <div className="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-6 gap-2">
          <IndicatorCard label="EMA 9" value={indicators.ema9} hint="Trend" />
          <IndicatorCard label="EMA 21" value={indicators.ema21} hint="Trend" />
          <IndicatorCard label="EMA 50" value={indicators.ema50} hint="Trend" />
          <IndicatorCard label="EMA 200" value={indicators.ema200} hint="Trend" />
          <IndicatorCard label="RSI 14" value={indicators.rsi} hint={Number(indicators.rsi) > 70 ? "Overbought" : Number(indicators.rsi) < 30 ? "Oversold" : "Neutral"} highlight={Number(indicators.rsi) > 70 || Number(indicators.rsi) < 30} />
          <IndicatorCard label="MACD" value={indicators.macd_main} hint="Momentum" />
          <IndicatorCard label="ADX 14" value={indicators.adx} hint={Number(indicators.adx) > 25 ? "Trending" : "Weak"} highlight={Number(indicators.adx) > 25} />
          <IndicatorCard label="+DI" value={indicators.adx_plus_di} hint="Bull" />
          <IndicatorCard label="-DI" value={indicators.adx_minus_di} hint="Bear" />
          <IndicatorCard label="ATR 14" value={indicators.atr} hint="Volatility" />
          <IndicatorCard label="Boll Upper" value={indicators.boll_upper} hint="Volatility" />
          <IndicatorCard label="Boll Lower" value={indicators.boll_lower} hint="Volatility" />
          <IndicatorCard label="Stochastic" value={indicators.stoch_main} hint="Momentum" />
          <IndicatorCard label="CCI 20" value={indicators.cci} hint="Momentum" />
          <IndicatorCard label="OBV" value={indicators.obv} hint="Volume" />
          <IndicatorCard label="VWAP" value={indicators.vwap} hint="Volume" />
          <IndicatorCard label="PSAR" value={indicators.psar} hint="Trend" />
          <IndicatorCard label="Session" value={snapshot?.session?.name} hint={snapshot?.session?.is_overlap ? "Overlap" : ""} />
        </div>
      </div>

      {/* Structure summary */}
      {structure && (
        <div className="rounded-xl border border-pat-border bg-pat-bg-surface p-4">
          <h3 className="text-sm font-semibold text-pat-text-primary mb-3">Market Structure</h3>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-3 text-xs">
            <StructureItem label="Current Trend" value={structure.CurrentTrend || "—"} />
            <StructureItem label="Last BOS" value={structure.LastBOS?.Direction || "—"} />
            <StructureItem label="BOS Price" value={structure.LastBOS?.Price || "—"} />
            <StructureItem label="Last CHoCH" value={structure.LastCHoCH?.Direction || "—"} />
          </div>
        </div>
      )}

      {/* Symbol Info */}
      {snapshot?.symbol_info && (
        <div className="rounded-xl border border-pat-border bg-pat-bg-surface p-4">
          <h3 className="text-sm font-semibold text-pat-text-primary mb-3">Symbol Information</h3>
          <div className="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-6 gap-3 text-xs">
            <InfoItem label="Digits" value={snapshot.symbol_info.digits} />
            <InfoItem label="Spread" value={snapshot.symbol_info.spread} />
            <InfoItem label="Min Lot" value={snapshot.symbol_info.min_lot} />
            <InfoItem label="Max Lot" value={snapshot.symbol_info.max_lot} />
            <InfoItem label="Lot Step" value={snapshot.symbol_info.lot_step} />
            <InfoItem label="Swap Long" value={snapshot.symbol_info.swap_long} />
            <InfoItem label="Swap Short" value={snapshot.symbol_info.swap_short} />
            <InfoItem label="Tick Value" value={snapshot.symbol_info.tick_value} />
            <InfoItem label="Tick Size" value={snapshot.symbol_info.tick_size} />
          </div>
        </div>
      )}
    </div>
  );
}

// ─── TRADING Mode (SOW 174) ──────────────────────────────────────────────────
function TradingMode({ signals, snapshot }: { signals: SignalRecord[]; snapshot?: MarketSnapshot }) {
  const directional = signals.filter((s) => s.Direction !== "NO-TRADE");
  const candidates = directional.filter((s) => s.SignalClass === "ADVISORY" || s.Direction.includes("CANDIDATE"));
  const qualified = directional.filter((s) => !s.Direction.includes("CANDIDATE"));

  return (
    <div className="space-y-4">
      {/* Signal Pipeline (SOW 174.1) */}
      <div className="rounded-xl border border-pat-border bg-pat-bg-surface p-4">
        <div className="flex items-center justify-between mb-3">
          <h3 className="text-sm font-semibold text-pat-text-primary">Live Signal Pipeline</h3>
          <div className="flex items-center gap-2 text-xs">
            <span className="text-pat-text-muted">{directional.length} active</span>
            <span className="inline-block h-1.5 w-1.5 rounded-full bg-pat-success animate-pulse" />
            <span className="text-[10px] text-pat-text-muted">Auto-refresh 5s</span>
          </div>
        </div>

        {directional.length === 0 ? (
          <div className="text-sm text-pat-text-muted py-6 text-center">No directional signals. Market is quiet — this is correct behavior.</div>
        ) : (
          <div className="space-y-2">
            {/* Qualified signals first */}
            {qualified.length > 0 && (
              <>
                <div className="text-[10px] text-pat-success font-semibold uppercase">Qualified Signals ({qualified.length})</div>
                {qualified.slice(0, 5).map((s) => <SignalCard key={s.ID} signal={s} />)}
              </>
            )}
            {/* Candidate signals */}
            {candidates.length > 0 && (
              <>
                <div className="text-[10px] text-pat-warning font-semibold uppercase mt-3">Candidate Signals ({candidates.length})</div>
                {candidates.slice(0, 8).map((s) => <SignalCard key={s.ID} signal={s} />)}
              </>
            )}
          </div>
        )}
      </div>

      {/* Account & Positions (SOW 176) */}
      {snapshot?.account_info && (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div className="rounded-xl border border-pat-border bg-pat-bg-surface p-4">
            <h3 className="text-sm font-semibold text-pat-text-primary mb-3">Account</h3>
            <div className="space-y-2 text-xs">
              <AccountRow label="Balance" value={snapshot.account_info.balance} currency={snapshot.account_info.currency} />
              <AccountRow label="Equity" value={snapshot.account_info.equity} currency={snapshot.account_info.currency} />
              <AccountRow label="Margin" value={snapshot.account_info.margin} currency={snapshot.account_info.currency} />
              <AccountRow label="Free Margin" value={snapshot.account_info.free_margin} currency={snapshot.account_info.currency} />
              <AccountRow label="Floating P&L" value={snapshot.account_info.profit} currency={snapshot.account_info.currency} highlight />
              <AccountRow label="Leverage" value={`1:${snapshot.account_info.leverage}`} />
              <AccountRow label="Server" value={snapshot.account_info.server} />
            </div>
          </div>

          <div className="rounded-xl border border-pat-border bg-pat-bg-surface p-4">
            <h3 className="text-sm font-semibold text-pat-text-primary mb-3">Open Positions</h3>
            {snapshot.positions && snapshot.positions.total_positions > 0 ? (
              <div className="space-y-2 text-xs">
                <AccountRow label="Total Positions" value={snapshot.positions.total_positions} />
                <AccountRow label="Buy Count" value={snapshot.positions.buy_count} />
                <AccountRow label="Sell Count" value={snapshot.positions.sell_count} />
                <AccountRow label="Total Lots" value={snapshot.positions.total_lots} />
                <AccountRow label="Floating Profit" value={snapshot.positions.floating_profit} currency="USD" highlight />
              </div>
            ) : (
              <div className="text-sm text-pat-text-muted py-4 text-center">No open positions</div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}

// ─── GROWTH Mode (SOW 178) ──────────────────────────────────────────────────
function GrowthMode({ subscriptions, commission, referrals }: {
  subscriptions?: unknown;
  commission?: { total_amount: string; pending_count: string; confirmed_count: string; pending_amount: string; confirmed_amount: string };
  referrals?: unknown;
}) {
  const subCount = Array.isArray(subscriptions) ? subscriptions.length : 0;
  const totalEarnings = parseFloat(commission?.total_amount ?? "0");
  const confirmedAmt = parseFloat(commission?.confirmed_amount ?? "0");
  const pendingAmt = parseFloat(commission?.pending_amount ?? "0");
  const referralCount = Array.isArray(referrals) ? referrals.length : (referrals && typeof referrals === "object" ? Object.keys(referrals).length : 0);

  return (
    <div className="space-y-4">
      <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
        <GrowthCard label="Active Subscriptions" value={subCount} icon={IconChartBar} />
        <GrowthCard label="Total Earnings" value={`$${totalEarnings.toFixed(2)}`} icon={IconCoins} />
        <GrowthCard label="Confirmed" value={`$${confirmedAmt.toFixed(2)}`} icon={IconCheck} color="text-pat-success" />
        <GrowthCard label="Pending" value={`$${pendingAmt.toFixed(2)}`} icon={IconHourglass} color="text-pat-warning" />
      </div>

      <div className="rounded-xl border border-pat-border bg-pat-bg-surface p-4">
        <h3 className="text-sm font-semibold text-pat-text-primary mb-3">Commission Breakdown</h3>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-3 text-xs">
          <InfoItem label="Confirmed Count" value={commission?.confirmed_count ?? "0"} />
          <InfoItem label="Pending Count" value={commission?.pending_count ?? "0"} />
          <InfoItem label="Confirmed Amount" value={`$${confirmedAmt.toFixed(2)}`} />
          <InfoItem label="Pending Amount" value={`$${pendingAmt.toFixed(2)}`} />
        </div>
      </div>

      <div className="rounded-xl border border-pat-border bg-pat-bg-surface p-4">
        <h3 className="text-sm font-semibold text-pat-text-primary mb-3">Referral Network</h3>
        {referralCount > 0 ? (
          <div className="text-xs text-pat-text-secondary">{referralCount} referrals in network</div>
        ) : (
          <div className="text-sm text-pat-text-muted py-4 text-center">No referrals yet. Share your link to start earning.</div>
        )}
      </div>
    </div>
  );
}

// ─── COMMAND_CENTER Mode (SOW 167.2) ────────────────────────────────────────
function CommandCenterMode({ snapshot, signals, subscriptions, commission }: {
  snapshot?: MarketSnapshot;
  signals: SignalRecord[];
  subscriptions?: unknown;
  commission?: { total_amount: string; pending_count: string; confirmed_count: string; pending_amount: string; confirmed_amount: string };
}) {
  return (
    <div className="space-y-4">
      {/* Combined view: top row = Market + Trading, bottom = Growth */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <div className="space-y-3">
          <div className="rounded-xl border border-pat-border bg-pat-bg-surface p-4">
            <h3 className="text-sm font-semibold text-pat-text-primary mb-2">Market Overview</h3>
            <CompactMarketView snapshot={snapshot} />
          </div>
        </div>
        <div className="space-y-3">
          <div className="rounded-xl border border-pat-border bg-pat-bg-surface p-4">
            <h3 className="text-sm font-semibold text-pat-text-primary mb-2">Active Signals</h3>
            <CompactSignalsView signals={signals} />
          </div>
        </div>
      </div>

      <div className="rounded-xl border border-pat-border bg-pat-bg-surface p-4">
        <h3 className="text-sm font-semibold text-pat-text-primary mb-2">Growth Summary</h3>
        <div className="grid grid-cols-3 gap-3 text-xs">
          <InfoItem label="Subscriptions" value={Array.isArray(subscriptions) ? subscriptions.length : 0} />
          <InfoItem label="Total Earnings" value={`$${parseFloat(commission?.total_amount ?? "0").toFixed(2)}`} />
          <InfoItem label="Pending" value={`$${parseFloat(commission?.pending_amount ?? "0").toFixed(2)}`} />
        </div>
      </div>
    </div>
  );
}

// ─── Helper Components ─────────────────────────────────────────────────────
function IndicatorCard({ label, value, hint, highlight }: { label: string; value?: number | string | boolean; hint?: string; highlight?: boolean }) {
  const display = value === undefined || value === null ? "—" : typeof value === "number" ? value.toFixed(2) : typeof value === "boolean" ? (value ? "Yes" : "No") : String(value);
  return (
    <div className={`rounded-lg border p-2.5 ${highlight ? "border-pat-warning/30 bg-pat-warning/5" : "border-pat-border/50 bg-pat-bg-surface-secondary/30"}`}>
      <div className="text-[10px] text-pat-text-muted uppercase">{label}</div>
      <div className="text-sm font-mono text-pat-text-primary tabular-nums mt-0.5">{display}</div>
      {hint && <div className={`text-[10px] mt-0.5 ${highlight ? "text-pat-warning" : "text-pat-text-muted"}`}>{hint}</div>}
    </div>
  );
}

function SignalCard({ signal }: { signal: SignalRecord }) {
  const entry = parseFloat(signal.EntryPrice || "0");
  const sl = parseFloat(signal.StopLoss || "0");
  const tp1 = parseFloat(signal.TP1 || "0");
  const tp2 = parseFloat(signal.TP2 || "0");
  const tp3 = parseFloat(signal.TP3 || "0");
  const slDist = Math.abs(entry - sl);
  const tp1Dist = Math.abs(tp1 - entry);
  const rr1 = slDist > 0 ? tp1Dist / slDist : 0;
  const score = parseFloat(signal.RawScore || "0");
  const prob = parseFloat(signal.CalibratedProbability || "0");
  const isCandidate = signal.Direction.includes("CANDIDATE");

  return (
    <div className={`rounded-lg border p-3 ${isCandidate ? "border-pat-warning/30 bg-pat-warning/5" : "border-pat-success/30 bg-pat-success/5"}`}>
      <div className="flex items-center justify-between mb-2">
        <div className="flex items-center gap-2">
          <span className={`text-sm font-bold ${
            signal.Direction === "BUY" ? "text-pat-success" :
            signal.Direction === "SELL" ? "text-pat-danger" :
            signal.Direction === "BUY_CANDIDATE" ? "text-pat-warning" :
            signal.Direction === "SELL_CANDIDATE" ? "text-pat-candidate-sell" :
            signal.Direction === "NO-TRADE" ? "text-pat-text-muted" :
            "text-pat-text-muted"
          }`}>{signal.Direction}</span>
          <span className="text-xs text-pat-text-muted">{signal.StrategyID?.replace(/_/g, " ") || "Unknown"}</span>
        </div>
        <div className="flex items-center gap-3 text-xs">
          <span className="text-pat-text-muted">Score: <span className="text-pat-text-primary font-mono">{score.toFixed(1)}</span></span>
          {prob > 0 && <span className="text-pat-text-muted">Prob: <span className="text-pat-text-primary font-mono">{(prob * 100).toFixed(1)}%</span></span>}
          {signal.Executable && <span className="text-[10px] px-1.5 py-0.5 rounded-full bg-pat-success/15 text-pat-success border border-pat-success/30">EXEC</span>}
        </div>
      </div>
      <div className="grid grid-cols-4 gap-2 text-xs">
        <div><span className="text-pat-text-muted">Entry:</span> <span className="font-mono text-pat-text-primary">{entry > 0 ? entry.toFixed(2) : "—"}</span></div>
        <div><span className="text-pat-text-muted">SL:</span> <span className="font-mono text-pat-danger">{sl > 0 ? sl.toFixed(2) : "—"}</span></div>
        <div><span className="text-pat-text-muted">TP1:</span> <span className="font-mono text-pat-success">{tp1 > 0 ? tp1.toFixed(2) : "—"}</span></div>
        <div><span className="text-pat-text-muted">R:R:</span> <span className="font-mono text-pat-text-secondary">{rr1 > 0 ? `1:${rr1.toFixed(2)}` : "—"}</span></div>
      </div>
      {(tp2 > 0 || tp3 > 0) && (
        <div className="grid grid-cols-4 gap-2 text-xs mt-1">
          <div><span className="text-pat-text-muted">TP2:</span> <span className="font-mono text-pat-success">{tp2 > 0 ? tp2.toFixed(2) : "—"}</span></div>
          <div><span className="text-pat-text-muted">TP3:</span> <span className="font-mono text-pat-success">{tp3 > 0 ? tp3.toFixed(2) : "—"}</span></div>
          <div><span className="text-pat-text-muted">Regime:</span> <span className="text-pat-text-secondary">{signal.Regime || "—"}</span></div>
          <div><span className="text-pat-text-muted">Session:</span> <span className="text-pat-text-secondary">{signal.Session || "—"}</span></div>
        </div>
      )}
      {signal.ReasonCodes && signal.ReasonCodes.length > 0 && signal.Direction.includes("CANDIDATE") && (
        <div className="mt-2 text-[10px] text-pat-text-muted">Note: {signal.ReasonCodes.join(", ")}</div>
      )}
    </div>
  );
}

function StructureItem({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between rounded-md bg-pat-bg-surface-secondary/30 px-3 py-1.5">
      <span className="text-pat-text-muted">{label}</span>
      <span className="text-pat-text-secondary capitalize">{value}</span>
    </div>
  );
}

function InfoItem({ label, value }: { label: string; value: number | string }) {
  return (
    <div className="rounded-md bg-pat-bg-surface-secondary/30 px-3 py-1.5">
      <div className="text-[10px] text-pat-text-muted uppercase">{label}</div>
      <div className="text-xs text-pat-text-primary font-mono mt-0.5">{value}</div>
    </div>
  );
}

function AccountRow({ label, value, currency, highlight }: { label: string; value: number | string; currency?: string; highlight?: boolean }) {
  const formatted = typeof value === "number" ? `${currency ? "$" : ""}${value.toFixed(2)}${currency ? ` ${currency}` : ""}` : String(value);
  return (
    <div className="flex items-center justify-between">
      <span className="text-pat-text-muted">{label}</span>
      <span className={`font-mono ${highlight ? (typeof value === "number" && value >= 0 ? "text-pat-success" : "text-pat-danger") : "text-pat-text-primary"}`}>{formatted}</span>
    </div>
  );
}

function GrowthCard({ label, value, icon: Icon, color }: { label: string; value: string | number; icon: React.ComponentType<{ size?: number; className?: string }>; color?: string }) {
  return (
    <div className="rounded-xl border border-pat-border bg-pat-bg-surface p-4">
      <div className="flex items-center justify-between mb-1">
        <span className="text-xs text-pat-text-muted uppercase">{label}</span>
        <Icon size={18} className={color || "text-pat-text-secondary"} />
      </div>
      <div className={`text-lg font-bold ${color || "text-pat-text-primary"}`}>{value}</div>
    </div>
  );
}

function CompactMarketView({ snapshot }: { snapshot?: MarketSnapshot }) {
  const tick = snapshot?.tick;
  const ind = snapshot?.indicators || {};
  return (
    <div className="space-y-2 text-xs">
      <div className="flex items-center justify-between">
        <div><span className="text-pat-text-muted">Bid:</span> <span className="font-mono text-pat-success">{tick?.bid?.toFixed(2) ?? "—"}</span></div>
        <div><span className="text-pat-text-muted">Ask:</span> <span className="font-mono text-pat-danger">{tick?.ask?.toFixed(2) ?? "—"}</span></div>
        <div><span className="text-pat-text-muted">Spread:</span> <span className="font-mono text-pat-text-primary">{tick?.spread?.toFixed(2) ?? "—"}</span></div>
      </div>
      <div className="flex items-center justify-between">
        <div><span className="text-pat-text-muted">RSI:</span> <span className="font-mono text-pat-text-secondary">{Number(ind.rsi || 0).toFixed(1)}</span></div>
        <div><span className="text-pat-text-muted">ADX:</span> <span className="font-mono text-pat-text-secondary">{Number(ind.adx || 0).toFixed(1)}</span></div>
        <div><span className="text-pat-text-muted">ATR:</span> <span className="font-mono text-pat-text-secondary">{Number(ind.atr || 0).toFixed(2)}</span></div>
      </div>
    </div>
  );
}

function CompactSignalsView({ signals }: { signals: SignalRecord[] }) {
  const directional = signals.filter((s) => s.Direction !== "NO-TRADE").slice(0, 5);
  if (directional.length === 0) return <div className="text-xs text-pat-text-muted py-2 text-center">No active signals</div>;
  return (
    <div className="space-y-1.5">
      {directional.map((s) => (
        <div key={s.ID} className="flex items-center justify-between text-xs rounded-md bg-pat-bg-surface-secondary/30 px-2 py-1.5">
          <div className="flex items-center gap-2">
            <span className={`font-bold ${s.Direction.includes("BUY") ? "text-pat-success" : "text-pat-danger"}`}>{s.Direction.replace("_", " ")}</span>
            <span className="text-pat-text-muted">{s.StrategyID?.replace(/_/g, " ") || "Unknown"}</span>
          </div>
          <div className="flex items-center gap-2">
            <span className="text-pat-text-muted">Score: <span className="font-mono text-pat-text-primary">{parseFloat(s.RawScore || "0").toFixed(1)}</span></span>
            <span className="text-pat-text-muted">{s.CreatedAt ? format(new Date(s.CreatedAt), "HH:mm") : "—"}</span>
          </div>
        </div>
      ))}
    </div>
  );
}
