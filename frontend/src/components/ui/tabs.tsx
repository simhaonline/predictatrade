"use client";
import React from "react";

export interface TabItem {
  id: string;
  label: string;
  icon?: React.ComponentType<{ size?: number; className?: string }>;
}

export function Tabs({
  tabs,
  active,
  onChange,
}: {
  tabs: TabItem[];
  active: string;
  onChange: (id: string) => void;
}) {
  return (
    <div className="flex flex-wrap gap-1.5 border-b border-pat-border pb-3">
      {tabs.map((t) => {
        const Icon = t.icon;
        const isActive = active === t.id;
        return (
          <button
            key={t.id}
            onClick={() => onChange(t.id)}
            className={`flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-xs font-medium transition-colors ${
              isActive
                ? "bg-primary text-primary-foreground"
                : "text-pat-text-secondary hover:bg-pat-bg-surface-secondary hover:text-pat-text-primary"
            }`}
          >
            {Icon ? <Icon size={15} /> : null}
            {t.label}
          </button>
        );
      })}
    </div>
  );
}

export function DegradedNote({ children }: { children: React.ReactNode }) {
  return (
    <div className="rounded-lg border border-pat-warning/30 bg-pat-warning/5 p-4 text-sm text-pat-text-secondary">
      <span className="font-semibold text-pat-warning">Degraded · </span>
      {children}
    </div>
  );
}

export function EmptyState({ title, hint }: { title: string; hint?: string }) {
  return (
    <div className="flex flex-col items-center justify-center gap-1 py-10 text-center">
      <div className="text-sm font-medium text-pat-text-secondary">{title}</div>
      {hint ? <div className="text-xs text-pat-text-muted">{hint}</div> : null}
    </div>
  );
}
