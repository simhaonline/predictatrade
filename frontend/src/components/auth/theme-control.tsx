"use client";
import { useTheme } from "next-themes";
import { IconSun, IconMoon, IconDeviceDesktop, IconChevronDown } from "@tabler/icons-react";
import { useState, useRef, useEffect } from "react";

const themeOptions = [
  { value: "system" as const, label: "System Mode", icon: IconDeviceDesktop },
  { value: "light" as const, label: "Light Mode", icon: IconSun },
  { value: "dark" as const, label: "Dark Mode", icon: IconMoon },
];

export default function ThemeControl() {
  const { theme, setTheme, resolvedTheme } = useTheme();
  const [open, setOpen] = useState(false);
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handler = (e: MouseEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) setOpen(false);
    };
    document.addEventListener("mousedown", handler);
    return () => document.removeEventListener("mousedown", handler);
  }, []);

  const CurrentIcon =
    theme === "system" ? IconDeviceDesktop : resolvedTheme === "dark" ? IconMoon : IconSun;

  return (
    <div ref={ref} className="relative">
      <button
        onClick={() => setOpen(!open)}
        className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-md border border-pat-border bg-pat-bg-surface text-pat-text-secondary hover:bg-pat-bg-surface-secondary hover:text-pat-text-primary transition-colors"
        aria-label="Display preferences"
        aria-haspopup="menu"
        aria-expanded={open}
      >
        <CurrentIcon size={18} />
        <IconChevronDown size={14} className={`transition-transform ${open ? "rotate-180" : ""}`} />
      </button>
      {open && (
        <div
          className="absolute right-0 top-full mt-2 w-44 bg-pat-bg-surface border border-pat-border rounded-lg shadow-lg py-1 z-50"
          role="menu"
          aria-label="Display preferences"
        >
          <div className="px-3 py-1 text-xs text-pat-text-muted border-b border-pat-border mb-1">
            Display Preferences
          </div>
          {themeOptions.map((opt) => (
            <button
              key={opt.value}
              onClick={() => { setTheme(opt.value); setOpen(false); }}
              className={`w-full flex items-center gap-2 px-3 py-2 text-sm transition-colors ${
                theme === opt.value
                  ? "bg-pat-bg-surface-secondary text-pat-text-primary font-medium"
                  : "text-pat-text-secondary hover:bg-pat-bg-surface-secondary hover:text-pat-text-primary"
              }`}
              role="menuitem"
            >
              <opt.icon size={16} />
              <span className="flex-1 text-left">{opt.label}</span>
              {theme === opt.value && <span className="text-pat-primary text-xs">✓</span>}
              {opt.value === "system" && theme === "system" && resolvedTheme && (
                <span className="text-xs text-pat-text-muted">({resolvedTheme})</span>
              )}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}
