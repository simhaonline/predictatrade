"use client";
import { useState } from "react";
import { IconBell, IconCheck } from "@tabler/icons-react";
import { DegradedNote } from "@/components/ui/tabs";

interface Prefs {
  signals: boolean;
  trades: boolean;
  billing: boolean;
  referrals: boolean;
  system: boolean;
}

const DEFAULTS: Prefs = {
  signals: true,
  trades: true,
  billing: true,
  referrals: true,
  system: true,
};

const CATEGORIES: { key: keyof Prefs; label: string; desc: string }[] = [
  { key: "signals", label: "Signals", desc: "New XAUUSD signal alerts and qualifying opportunities." },
  { key: "trades", label: "Trades", desc: "Execution, fill and position notifications." },
  { key: "billing", label: "Billing", desc: "Invoices, renewals and payment receipts." },
  { key: "referrals", label: "Referrals", desc: "New referrals, commissions and payout events." },
  { key: "system", label: "System", desc: "Maintenance, incidents and security notices." },
];

const STORAGE_KEY = "pat-notification-prefs";

export default function UserNotificationsPage() {
  const [prefs, setPrefs] = useState<Prefs>(() => {
    try {
      const raw = localStorage.getItem(STORAGE_KEY);
      if (raw) return { ...DEFAULTS, ...JSON.parse(raw) };
    } catch {
      /* ignore corrupt storage */
    }
    return DEFAULTS;
  });
  const [savedAt, setSavedAt] = useState<string | null>(null);

  const toggle = (key: keyof Prefs) => setPrefs((p) => ({ ...p, [key]: !p[key] }));

  const save = () => {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(prefs));
      setSavedAt(new Date().toLocaleTimeString());
    } catch {
      /* ignore */
    }
  };

  return (
    <div className="space-y-5">
      <div>
        <h1 className="text-xl font-bold text-pat-text-primary">Notifications</h1>
        <p className="text-sm text-pat-text-secondary mt-1">Choose which categories of alerts you want to receive.</p>
      </div>

      <DegradedNote>
        Server-side notification preferences are not yet persisted by a dedicated backend endpoint. Your selections are
        saved locally in this browser only and will not sync across devices or survive cache clears.
      </DegradedNote>

      <div className="space-y-2">
        {CATEGORIES.map((c) => {
          const on = prefs[c.key];
          return (
            <div key={c.key} className="flex items-center justify-between rounded-lg border border-pat-border bg-pat-bg-surface p-4">
              <div className="flex items-center gap-3">
                <div className={`flex items-center justify-center w-9 h-9 rounded-lg ${on ? "bg-pat-success/10" : "bg-pat-bg-surface-secondary"}`}>
                  <IconBell size={18} className={on ? "text-pat-success" : "text-pat-text-muted"} />
                </div>
                <div>
                  <div className="text-sm font-medium text-pat-text-primary">{c.label}</div>
                  <div className="text-xs text-pat-text-muted">{c.desc}</div>
                </div>
              </div>
              <button
                onClick={() => toggle(c.key)}
                role="switch"
                aria-checked={on}
                className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
                  on ? "bg-pat-success" : "bg-pat-bg-surface-secondary border border-pat-border"
                }`}
              >
                <span className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${on ? "translate-x-6" : "translate-x-1"}`} />
              </button>
            </div>
          );
        })}
      </div>

      <div className="flex items-center gap-3">
        <button onClick={save} className="rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground">
          Save preferences
        </button>
        {savedAt && (
          <span className="flex items-center gap-1.5 text-xs text-pat-success">
            <IconCheck size={14} /> Saved locally at {savedAt}
          </span>
        )}
      </div>
    </div>
  );
}
