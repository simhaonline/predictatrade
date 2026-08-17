interface SignalCardProps {
  direction: 'BUY' | 'SELL' | 'NO-TRADE' | 'WAIT';
  strategy: string;
  grade: string;
  ttl: string;
  entry: number | null;
  stopLoss: number | null;
  tp1: number | null;
  tp2?: number | null;
  tp3?: number | null;
  grossRR?: string;
  netRR?: string;
  calibratedProbability?: string;
  reason?: string[];
}

export function SignalCard(props: SignalCardProps) {
  const { direction, strategy, grade, ttl, entry, stopLoss, tp1, reason } = props;
  const dirClass = direction === 'BUY' ? 'buy' : direction === 'SELL' ? 'sell' : 'no-trade';
  const dirIcon = direction === 'BUY' ? '▲' : direction === 'SELL' ? '▼' : '●';

  return (
    <div className="signal-card">
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 'var(--spacing-2)' }}>
        <span style={{ fontSize: 'var(--font-size-12)', color: 'var(--text-muted)' }}>{strategy}</span>
        <span className={`chip ${grade === 'UNRATED' ? 'chip-neutral' : 'chip-warning'}`}>{grade}</span>
      </div>
      <div className={`signal-card-direction ${dirClass}`}>
        {dirIcon} {direction}
      </div>
      {props.calibratedProbability && (
        <div style={{ fontSize: 'var(--font-size-12)', color: 'var(--text-secondary)', marginTop: 'var(--spacing-1)' }}>
          p(TP1&lt;SL) = {props.calibratedProbability}
        </div>
      )}
      <div style={{ marginTop: 'var(--spacing-2)', fontSize: 'var(--font-size-12)', color: 'var(--text-muted)' }}>
        TTL: {ttl}
      </div>
      {entry !== null && (
        <div style={{ marginTop: 'var(--spacing-3)', fontSize: 'var(--font-size-13)' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between' }}>
            <span style={{ color: 'var(--text-muted)' }}>Entry</span>
            <span className="tabular">{entry.toFixed(2)}</span>
          </div>
          {stopLoss !== null && (
            <div style={{ display: 'flex', justifyContent: 'space-between' }}>
              <span style={{ color: 'var(--status-down)' }}>SL</span>
              <span className="tabular">{stopLoss.toFixed(2)}</span>
            </div>
          )}
          {tp1 !== null && (
            <div style={{ display: 'flex', justifyContent: 'space-between' }}>
              <span style={{ color: 'var(--status-up)' }}>TP1</span>
              <span className="tabular">{tp1.toFixed(2)}</span>
            </div>
          )}
        </div>
      )}
      {props.grossRR && (
        <div style={{ marginTop: 'var(--spacing-2)', fontSize: 'var(--font-size-12)' }}>
          <span style={{ color: 'var(--text-muted)' }}>Gross R:R </span>
          <span className="tabular">{props.grossRR}</span>
          {props.netRR && <span style={{ color: 'var(--text-muted)' }}> | Net: </span>}
          {props.netRR && <span className="tabular">{props.netRR}</span>}
        </div>
      )}
      {reason && reason.length > 0 && (
        <div style={{ marginTop: 'var(--spacing-2)', display: 'flex', flexWrap: 'wrap', gap: 'var(--spacing-1)' }}>
          {reason.map((r) => (
            <span key={r} className="chip chip-warning" style={{ fontSize: '10px' }}>{r}</span>
          ))}
        </div>
      )}
    </div>
  );
}
