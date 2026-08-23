"use client";
import { IconFlag, IconAlertTriangle } from "@tabler/icons-react";

function DegradedBanner({ children }: { children: React.ReactNode }) {
  return (
    <div className="bg-pat-warning/10 border border-pat-warning/20 rounded-lg p-3 flex items-start gap-2">
      <IconAlertTriangle size={16} className="text-pat-warning mt-0.5 shrink-0" />
      <div className="text-xs text-pat-warning">{children}</div>
    </div>
  );
}

interface FlagSchema {
  key: string;
  value: string;
  env: string;
  description: string;
}

const SCHEMA_FLAGS: FlagSchema[] = [
  { key: "signals.enabled", value: "true", env: "all", description: "Master switch for signal generation." },
  { key: "auto_execution.enabled", value: "false", env: "production", description: "Broker auto-execution gate." },
  { key: "news_blackout.enforced", value: "true", env: "all", description: "Enforce macro/news blackout windows." },
  { key: "replay.labeled", value: "true", env: "demo", description: "Require explicit demo/replay labeling." },
  { key: "payout.min_approval", value: "2", env: "production", description: "Minimum approvals for payout release." },
];

export default function AdminFeatureFlagsPage() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-bold text-pat-text-primary">Feature Flags &amp; Configuration</h1>
        <p className="text-sm text-pat-text-secondary mt-1">
          Configuration backend is not wired. The list below is a static schema sample; toggles are disabled.
        </p>
      </div>

      <DegradedBanner>
        Configuration backend pending. Toggles are read-only placeholders and do NOT change any live behavior.
        Values shown are illustrative schema entries, not live state.
      </DegradedBanner>

      <div className="bg-pat-card-bg border border-pat-card-border rounded-lg p-4 shadow-sm opacity-80">
        <h2 className="text-sm font-medium text-pat-text-primary mb-3 flex items-center gap-2">
          <IconFlag size={16} /> Flags (Pending Backend)
        </h2>
        <div className="space-y-2">
          {SCHEMA_FLAGS.map((f) => (
            <div
              key={f.key}
              className="flex items-center justify-between rounded-md bg-pat-bg-surface-secondary px-3 py-2"
            >
              <div>
                <span className="text-xs text-pat-text-primary font-mono">{f.key}</span>
                <span className="text-xs text-pat-text-muted ml-2">[{f.env}]</span>
                <div className="text-xs text-pat-text-muted">{f.description}</div>
              </div>
              <button
                disabled
                className="text-xs px-3 py-1 rounded-md border border-pat-border text-pat-text-muted opacity-60 cursor-not-allowed"
              >
                {f.value === "true" ? "On" : "Off"} (disabled)
              </button>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
