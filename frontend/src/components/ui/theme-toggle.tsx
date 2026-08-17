'use client';
export function ThemeToggle() {
  const toggle = () => {
    const current = document.documentElement.getAttribute('data-theme') || 'dark';
    const next = current === 'dark' ? 'light' : 'dark';
    document.documentElement.setAttribute('data-theme', next);
    try { localStorage.setItem('pat-theme', next); } catch {}
  };
  return <button onClick={toggle} className="btn" aria-label="Toggle theme">🌙 / ☀️</button>;
}
