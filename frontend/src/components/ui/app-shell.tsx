'use client';
import { useState } from 'react';
import Link from 'next/link';
import { usePathname } from 'next/navigation';

interface NavItem {
  id: string;
  label: string;
  icon: string;
  route: string;
}

interface NavSection {
  section: string;
  items: NavItem[];
}

interface AppShellProps {
  navItems: NavSection[];
  mode: 'user' | 'admin';
  children: React.ReactNode;
}

export function AppShell({ navItems, mode, children }: AppShellProps) {
  const [collapsed, setCollapsed] = useState(false);
  const [theme, setTheme] = useState<'dark' | 'light'>('dark');
  const pathname = usePathname();

  const toggleTheme = () => {
    const newTheme = theme === 'dark' ? 'light' : 'dark';
    setTheme(newTheme);
    document.documentElement.setAttribute('data-theme', newTheme);
    try { localStorage.setItem('pat-theme', newTheme); } catch {}
  };

  return (
    <div className="app-shell">
      <aside className={`sidebar ${collapsed ? 'collapsed' : ''}`}>
        {navItems.map((sec) => (
          <div key={sec.section} className="sidebar-section">
            {!collapsed && <div className="sidebar-section-header">{sec.section}</div>}
            {sec.items.map((item) => (
              <Link
                key={item.id}
                href={item.route}
                className={`sidebar-item ${pathname === item.route ? 'active' : ''}`}
                title={collapsed ? item.label : undefined}
              >
                <span style={{ fontSize: 'var(--font-size-16)' }}>{item.icon}</span>
                {!collapsed && <span>{item.label}</span>}
              </Link>
            ))}
          </div>
        ))}
      </aside>
      <div className="content">
        <header className="topbar">
          <button onClick={() => setCollapsed(!collapsed)} className="btn" style={{ padding: '4px 8px' }}>
            {collapsed ? '→' : '←'}
          </button>
          <div style={{ fontWeight: 600, color: 'var(--color-brand-gold)' }}>
            Predict-A-Trade {mode === 'admin' && <span style={{ color: 'var(--text-muted)', fontSize: 'var(--font-size-12)' }}>| ADMIN</span>}
          </div>
          <div style={{ flex: 1 }} />
          <button onClick={toggleTheme} className="btn" style={{ padding: '4px 8px' }} aria-label="Toggle theme">
            {theme === 'dark' ? '☀️' : '🌙'}
          </button>
          <Link href="/dashboard/security" className="btn" style={{ padding: '4px 8px' }}>👤</Link>
        </header>
        {mode === 'user' && (
          <div className="ticker-strip">
            <span className="tabular" style={{ fontWeight: 600 }}>XAUUSD $2,431.18</span>
            <span className="chip chip-up tabular">▲ +0.42%</span>
            <span style={{ color: 'var(--text-muted)' }}>Spread $0.21</span>
            <span style={{ color: 'var(--text-muted)' }}>Regime: RANGE</span>
            <span style={{ color: 'var(--text-muted)' }}>Session: LONDON</span>
            <span className="chip chip-up">● LIVE</span>
            <span style={{ color: 'var(--text-muted)', marginLeft: 'auto' }}>Broker 13:45 +03:00</span>
          </div>
        )}
        {children}
      </div>
    </div>
  );
}
