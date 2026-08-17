interface KPIStatProps {
  label: string;
  value: string;
  delta?: string;
  deltaPositive?: boolean;
}

export function KPIStat({ label, value, delta, deltaPositive }: KPIStatProps) {
  return (
    <div className="kpi-stat">
      <span className="kpi-stat-label">{label}</span>
      <span className="kpi-stat-value tabular">{value}</span>
      {delta && (
        <span className={`kpi-stat-delta ${deltaPositive ? 'chip-up' : 'chip-down'}`} style={{ display: 'inline-flex', alignItems: 'center', gap: '2px' }}>
          {deltaPositive ? '▲' : '▼'} {delta}
        </span>
      )}
    </div>
  );
}
