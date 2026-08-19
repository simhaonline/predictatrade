"use client";
import { useState, useEffect } from "react";
import { useTheme } from "next-themes";

function Toggle({ label, value, onChange }: { label: string; value: boolean; onChange: () => void }) {
  return (
    <div className="flex items-center justify-between">
      <span className="text-sm text-pat-text-primary">{label}</span>
      <button onClick={onChange} className={"px-3 py-1 rounded-md text-xs font-medium transition-colors " + (value ? "bg-pat-primary text-pat-primary-foreground" : "bg-pat-bg-surface-secondary text-pat-text-secondary border border-pat-border")}>{value ? "On" : "Off"}</button>
    </div>
  );
}

export default function AccessibilitySettings() {
  const { theme, setTheme } = useTheme();
  const [fontScale, setFontScale] = useState(() => {
    if (typeof window === 'undefined') return 100;
    try { return JSON.parse(localStorage.getItem('pat_accessibility') || '{}').fontScale ?? 100; } catch { return 100; }
  });
  const [highContrast, setHighContrast] = useState(() => {
    if (typeof window === 'undefined') return false;
    try { return JSON.parse(localStorage.getItem('pat_accessibility') || '{}').highContrast ?? false; } catch { return false; }
  });
  const [reduceMotion, setReduceMotion] = useState(() => {
    if (typeof window === 'undefined') return false;
    try { return JSON.parse(localStorage.getItem('pat_accessibility') || '{}').reduceMotion ?? false; } catch { return false; }
  });
  const [keyboardNav, setKeyboardNav] = useState(() => {
    if (typeof window === 'undefined') return false;
    try { return JSON.parse(localStorage.getItem('pat_accessibility') || '{}').keyboardNav ?? false; } catch { return false; }
  });

  useEffect(() => {
    if (typeof window !== "undefined") localStorage.setItem("pat_accessibility", JSON.stringify({ fontScale, highContrast, reduceMotion, keyboardNav }));
  }, [fontScale, highContrast, reduceMotion, keyboardNav]);

  useEffect(() => {
    if (typeof document !== "undefined") {
      document.documentElement.style.fontSize = fontScale + "%";
      document.documentElement.classList.toggle("high-contrast", highContrast);
      document.documentElement.classList.toggle("reduce-motion", reduceMotion);
      document.documentElement.classList.toggle("keyboard-nav", keyboardNav);
    }
  }, [fontScale, highContrast, reduceMotion, keyboardNav]);

  return (
    <div className="space-y-6">
      <div className="rounded-lg border border-pat-card-border bg-pat-card-bg p-4">
        <h2 className="text-sm font-semibold text-pat-text-primary mb-4">Display Theme</h2>
        <div className="space-y-2">
          {[
            { value: 'system', label: 'System Mode' },
            { value: 'light', label: 'Light Mode' },
            { value: 'dark', label: 'Dark Mode' },
          ].map((opt) => (
            <div key={opt.value} className="flex items-center justify-between">
              <span className="text-sm text-pat-text-primary">{opt.label}</span>
              <button
                onClick={() => setTheme(opt.value)}
                className={"px-3 py-1 rounded-md text-xs font-medium transition-colors " + (theme === opt.value ? "bg-pat-primary text-pat-primary-foreground" : "bg-pat-bg-surface-secondary text-pat-text-secondary border border-pat-border")}
              >
                {theme === opt.value ? "✓ " : ""}{opt.label}
              </button>
            </div>
          ))}
        </div>
      </div>

      <div className="rounded-lg border border-pat-card-border bg-pat-card-bg p-4">
        <h2 className="text-sm font-semibold text-pat-text-primary mb-4">Accessibility</h2>
        <div className="space-y-4">
          <div className="flex items-center justify-between">
            <span className="text-sm text-pat-text-primary">Font Scale {fontScale}%</span>
            <input type="range" min="80" max="150" value={fontScale} onChange={(e)=>setFontScale(Number(e.target.value))} className="w-32 accent-pat-primary" />
          </div>
          <Toggle label="High Contrast" value={highContrast} onChange={()=>setHighContrast((v: boolean)=>!v)} />
          <Toggle label="Reduce Motion" value={reduceMotion} onChange={()=>setReduceMotion((v: boolean)=>!v)} />
          <Toggle label="Keyboard Navigation" value={keyboardNav} onChange={()=>setKeyboardNav((v: boolean)=>!v)} />
        </div>
      </div>
    </div>
  );
}
