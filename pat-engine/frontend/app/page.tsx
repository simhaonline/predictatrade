"use client";

import { useEffect, useState } from "react";
import { api, BrokerProfile, RiskConfig, SessionInfo, SignalRecord, BarRecord } from "@/lib/api";

export default function Home() {
  const [broker, setBroker] = useState<BrokerProfile | null>(null);
  const [risk, setRisk] = useState<RiskConfig | null>(null);
  const [session, setSession] = useState<SessionInfo | null>(null);
  const [strategies, setStrategies] = useState<{ strategies: string[]; license_plan: string } | null>(null);
  const [signals, setSignals] = useState<SignalRecord[]>([]);
  const [bars, setBars] = useState<BarRecord[]>([]);
  const [updated, setUpdated] = useState<string>("");

  useEffect(() => {
    let alive = true;
    const load = async () => {
      const [b, r, s, st, sg, br] = await Promise.all([
        api.broker(), api.risk(), api.session(), api.strategies(), api.signals(25), api.bars(50),
      ]);
      if (!alive) return;
      setBroker(b); setRisk(r); setSession(s); setStrategies(st);
      setSignals(sg ?? []); setBars(br ?? []);
      setUpdated(new Date().toLocaleTimeString());
    };
    load();
    const id = setInterval(load, 5000);
    return () => { alive = false; clearInterval(id); };
  }, []);

  return (
    <div className="wrap">
      <header>
        <h1>Predict-A-Trade · XAUUSD Live Command Center</h1>
        <span className="badge">updated {updated || "—"}</span>
      </header>

      <div className="grid">
        <Panel title="Broker Execution">
          {!broker && <Empty />}
          {broker && (<>
            <Row k="Symbol" v={broker.symbol} />
            <Row k="Broker time offset (UTC)" v={`+${broker.timezone_offset}`} />
            <Row k="Contract size" v={`${broker.contract_size} oz`} />
            <Row k="Leverage" v={`1:${broker.leverage}`} />
            <Row k="Commission / lot" v={`$${broker.commission_per_lot}`} />
            <Row k="Typical spread" v={`${broker.typical_spread} pts`} />
            <Row k="Swap long / short" v={`${broker.swap_long} / ${broker.swap_short} pts`} />
            <Row k="Digits" v={`${broker.digits}`} />
          </>)}
        </Panel>

        <Panel title="Capital Risk Mandate">
          {!risk && <Empty />}
          {risk && (<>
            <Row k="Equity" v={fmt(risk.Equity)} />
            <Row k="Risk / trade" v={`${risk.RiskPerTradePct}%`} />
            <Row k="Max daily loss" v={`${risk.MaxDailyLossPct}%`} />
            <Row k="Max positions" v={`${risk.MaxPositions}`} />
            <Row k="Max leverage" v={risk.MaxLeverage < 0 ? "plan-limited" : `1:${risk.MaxLeverage}`} />
            <Row k="Min net R:R" v={`${risk.MinRR}`} />
          </>)}
        </Panel>

        <Panel title="Session (broker time)">
          {!session && <Empty />}
          {session && (<>
            <Row k="Current" v={session.session} />
            <Row k="London/NY overlap" v={session.overlap ? "YES" : "no"} />
            <Row k="TZ offset" v={`+${session.tz_offset}`} />
            <div style={{ marginTop: 10 }}>
              {strategies?.strategies.map((s) => (
                <span className="badge" key={s} style={{ marginRight: 6 }}>{s}</span>
              ))}
            </div>
            <div className="muted" style={{ marginTop: 8 }}>plan: {strategies?.license_plan}</div>
          </>)}
        </Panel>
      </div>

      <div className="grid" style={{ marginTop: 16 }}>
        <Panel title="Recent Signals">
          {signals.length === 0 && <Empty />}
          <table>
            <thead><tr><th>Time</th><th>Strategy</th><th>Dir</th><th>Entry</th><th>SL</th><th>TP1</th><th>Score</th><th>Status</th></tr></thead>
            <tbody>
              {signals.map((s) => (
                <tr key={s.id}>
                  <td>{s.ts?.slice(0, 19)?.replace("T", " ")}</td>
                  <td>{s.strategy_id}</td>
                  <td className={s.direction === "BUY" ? "buy" : "sell"}>{s.direction}</td>
                  <td className="v">{s.entry}</td>
                  <td className="v">{s.sl}</td>
                  <td className="v">{s.tp1}</td>
                  <td className="v">{s.raw_score?.toFixed(2)}</td>
                  <td><span className={`pill ${s.status === "ACTIVE" ? "ok" : "bad"}`}>{s.status}</span></td>
                </tr>
              ))}
            </tbody>
          </table>
        </Panel>

        <Panel title="Recent Bars">
          {bars.length === 0 && <Empty />}
          <table>
            <thead><tr><th>Time</th><th>Open</th><th>High</th><th>Low</th><th>Close</th><th>Spread</th></tr></thead>
            <tbody>
              {bars.slice().reverse().slice(0, 20).map((b, i) => (
                <tr key={i}>
                  <td>{b.time?.slice(0, 19)?.replace("T", " ")}</td>
                  <td className="v">{b.open}</td>
                  <td className="v">{b.high}</td>
                  <td className="v">{b.low}</td>
                  <td className="v">{b.close}</td>
                  <td className="v">{b.spread}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </Panel>
      </div>

      <p className="muted" style={{ marginTop: 16 }}>
        All values are server-authoritative from pat-engine. Demo/replay data is labeled and never mutates live trading.
      </p>
    </div>
  );
}

function Panel({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="card">
      <h2>{title}</h2>
      {children}
    </div>
  );
}
function Row({ k, v }: { k: string; v: string | number }) {
  return (<div className="row"><span className="k">{k}</span><span className="v">{v}</span></div>);
}
function Empty() { return <div className="muted">connecting to /api/v1…</div>; }
function fmt(n: number) { return n.toLocaleString(undefined, { maximumFractionDigits: 2 }); }
