"use client";
import { useAuth } from "@/providers/auth-provider";
import { IconLogout, IconSun, IconMoon, IconDeviceDesktop, IconChevronDown } from "@tabler/icons-react";
import { useTheme } from "next-themes";
import { useState, useRef, useEffect } from "react";

export default function Topbar() {
  const { user, logout } = useAuth();
  const { theme, setTheme, resolvedTheme } = useTheme();
  const [menuOpen, setMenuOpen] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setMenuOpen(false);
      }
    };
    document.addEventListener('mousedown', handler);
    return () => document.removeEventListener('mousedown', handler);
  }, []);

  const themeOptions = [
    { value: 'system', label: 'System Mode', icon: IconDeviceDesktop },
    { value: 'light', label: 'Light Mode', icon: IconSun },
    { value: 'dark', label: 'Dark Mode', icon: IconMoon },
  ];

  const CurrentIcon = theme === 'system' ? IconDeviceDesktop : resolvedTheme === 'dark' ? IconMoon : IconSun;

  return (
    <header className="sticky top-0 z-20 flex items-center justify-between px-4 py-3 bg-pat-bg-header/80 backdrop-blur border-b border-pat-border">
      <div className="text-sm font-medium text-pat-text-primary" data-testid="topbar-user-name">
        {user?.name || user?.email || "Guest"}
      </div>
      <div className="flex items-center gap-3">
        {/* Theme preference dropdown */}
        <div ref={menuRef} className="relative">
          <button
            onClick={() => setMenuOpen(!menuOpen)}
            className="p-2 rounded-md hover:bg-pat-bg-surface-secondary text-pat-text-secondary transition-colors flex items-center gap-1"
            aria-label="Display preferences"
            aria-haspopup="menu"
            aria-expanded={menuOpen}
          >
            <CurrentIcon size={18} />
            <IconChevronDown size={14} className={`transition-transform ${menuOpen ? 'rotate-180' : ''}`} />
          </button>
          {menuOpen && (
            <div
              className="absolute right-0 top-full mt-2 w-48 bg-pat-bg-surface border border-pat-border rounded-lg shadow-lg py-1 z-50"
              role="menu"
              aria-label="Display preferences"
            >
              <div className="px-3 py-1 text-xs text-pat-text-muted border-b border-pat-border mb-1">
                Display Preferences
              </div>
              {themeOptions.map((opt) => (
                <button
                  key={opt.value}
                  onClick={() => { setTheme(opt.value); setMenuOpen(false); }}
                  className={`w-full flex items-center gap-2 px-3 py-2 text-sm transition-colors ${
                    theme === opt.value
                      ? 'bg-pat-bg-surface-secondary text-pat-text-primary font-medium'
                      : 'text-pat-text-secondary hover:bg-pat-bg-surface-secondary hover:text-pat-text-primary'
                  }`}
                  role="menuitem"
                  
                >
                  <opt.icon size={16} />
                  <span className="flex-1 text-left">{opt.label}</span>
                  {theme === opt.value && (
                    <span className="text-pat-primary text-xs">✓</span>
                  )}
                  {opt.value === 'system' && theme === 'system' && resolvedTheme && (
                    <span className="text-xs text-pat-text-muted">({resolvedTheme})</span>
                  )}
                </button>
              ))}
            </div>
          )}
        </div>

        <button
          onClick={logout}
          className="flex items-center gap-1 text-sm text-pat-text-secondary hover:text-pat-text-primary transition-colors"
          aria-label="Logout"
        >
          <IconLogout size={18} />
          <span className="hidden sm:inline">Logout</span>
        </button>
      </div>
    </header>
  );
}
