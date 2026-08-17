import { SignalCard } from '@/components/trading/signal-card';
import { KPIStat } from '@/components/ui/kpi-stat';

export default function DashboardOverview() {
  return (
    <div className="page-content">
      <h1 style={{ fontSize: 'var(--font-size-24)', marginBottom: 'var(--spacing-4)' }}>Trading Overview</h1>
      <div className="grid grid-12" style={{ marginBottom: 'var(--spacing-4)' }}>
        <div className="col-span-3"><div className="card"><KPIStat label="XAUUSD" value="$2,431.18" delta="+0.42%" deltaPositive /></div></div>
        <div className="col-span-3"><div className="card"><KPIStat label="Session" value="LONDON" /></div></div>
        <div className="col-span-3"><div className="card"><KPIStat label="Regime" value="RANGE" /></div></div>
        <div className="col-span-3"><div className="card"><KPIStat label="License" value="ACTIVE" /></div></div>
      </div>
      <div className="grid grid-12">
        <div className="col-span-7">
          <div className="card" style={{ minHeight: 400 }}>
            <h3 style={{ marginBottom: 'var(--spacing-4)' }}>Live XAUUSD Chart</h3>
            <div className="empty-state">Chart loading — connecting to real-time gateway...</div>
          </div>
        </div>
        <div className="col-span-3">
          <SignalCard direction="NO-TRADE" strategy="STANDARD_SCALPING" grade="UNRATED" ttl="—" entry={null} stopLoss={null} tp1={null} reason={['INSUFFICIENT_SCORE']} />
        </div>
        <div className="col-span-2">
          <div className="card">
            <h4 style={{ marginBottom: 'var(--spacing-2)' }}>Market Pulse</h4>
            <div style={{ fontSize: 'var(--font-size-12)', color: 'var(--text-secondary)' }}>
              <div>M1: NEUTRAL</div><div>M5: BULLISH</div><div>M15: RANGE</div>
              <div>H1: BULLISH</div><div>H4: NEUTRAL</div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
