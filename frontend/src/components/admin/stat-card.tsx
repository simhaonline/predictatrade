"use client";
import type { Icon } from "@tabler/icons-react";

interface StatCardProps {
  label: string;
  value: string | number;
  sub?: string;
  icon: Icon;
  color?: string;
}

export default function StatCard({ label, value, sub, icon: Icon, color = "text-pat-text-secondary" }: StatCardProps) {
  return (
    <div className="bg-pat-card-bg border border-pat-card-border rounded-lg p-4 shadow-sm">
      <div className="flex items-center justify-between mb-2">
        <span className="text-xs text-pat-text-muted">{label}</span>
        <Icon size={18} className={color} />
      </div>
      <div className="text-xl font-bold text-pat-text-primary">{typeof value === "number" ? value.toLocaleString() : value}</div>
      {sub && <div className="text-xs text-pat-text-muted mt-1">{sub}</div>}
    </div>
  );
}
