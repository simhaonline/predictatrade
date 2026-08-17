import { KPIStat } from '@/components/ui/kpi-stat';

export default function AdminDashboard() {
  return (
    <div className="page-content">
      <h1 style={{ fontSize: 'var(--font-size-24)', marginBottom: 'var(--spacing-4)' }}>Operations Dashboard</h1>
      <div className="grid grid-12" style={{ marginBottom: 'var(--spacing-4)' }}>
        <div className="col-span-3"><div className="card"><KPIStat label="MRR" value="$12,450" delta="+8.2%" deltaPositive /></div></div>
        <div className="col-span-3"><div className="card"><KPIStat label="Active Subs" value="34" delta="+3" deltaPositive /></div></div>
        <div className="col-span-3"><div className="card"><KPIStat label="Pending Comm" value="$1,240" /></div></div>
        <div className="col-span-3"><div className="card"><KPIStat label="Payouts Pending" value="2" /></div></div>
      </div>
      <div className="card">
        <h3 style={{ marginBottom: 'var(--spacing-4)' }}>Platform Health</h3>
        <div style={{ display: 'flex', gap: 'var(--spacing-4)', flexWrap: 'wrap' }}>
          <span className="chip chip-up">● Operational</span>
          <span className="chip chip-info">Feed: Primary OK</span>
          <span className="chip chip-neutral">DB: Connected</span>
          <span className="chip chip-neutral">Valkey: Connected</span>
          <span className="chip chip-neutral">Agents: 12 online</span>
        </div>
      </div>
    </div>
  );
}
