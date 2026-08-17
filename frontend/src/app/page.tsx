export default function Home() {
  return (
    <div style={{ minHeight: '100vh', display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center' }}>
      <h1 style={{ fontSize: 'var(--font-size-32)', color: 'var(--color-brand-gold)' }}>Predict-A-Trade</h1>
      <p style={{ color: 'var(--text-secondary)', marginTop: 'var(--spacing-4)' }}>XAUUSD Intelligence Platform</p>
      <div style={{ marginTop: 'var(--spacing-8)', display: 'flex', gap: 'var(--spacing-4)' }}>
        <a href="/dashboard" className="btn btn-primary">Enter Dashboard</a>
        <a href="/admin" className="btn">Admin</a>
      </div>
    </div>
  );
}
