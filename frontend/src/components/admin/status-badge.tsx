"use client";

const styles: Record<string, string> = {
  healthy: "bg-pat-badge-success-bg text-pat-badge-success-text border-pat-badge-success-bg",
  live: "bg-pat-badge-success-bg text-pat-badge-success-text border-pat-badge-success-bg",
  online: "bg-pat-badge-success-bg text-pat-badge-success-text border-pat-badge-success-bg",
  active: "bg-pat-badge-success-bg text-pat-badge-success-text border-pat-badge-success-bg",
  connected: "bg-pat-badge-success-bg text-pat-badge-success-text border-pat-badge-success-bg",
  operational: "bg-pat-badge-success-bg text-pat-badge-success-text border-pat-badge-success-bg",
  degraded: "bg-pat-badge-warning-bg text-pat-badge-warning-text border-pat-badge-warning-bg",
  stale: "bg-pat-badge-warning-bg text-pat-badge-warning-text border-pat-badge-warning-bg",
  reconnecting: "bg-pat-badge-warning-bg text-pat-badge-warning-text border-pat-badge-warning-bg",
  maintenance: "bg-pat-badge-warning-bg text-pat-badge-warning-text border-pat-badge-warning-bg",
  paused: "bg-pat-badge-warning-bg text-pat-badge-warning-text border-pat-badge-warning-bg",
  halted: "bg-pat-badge-danger-bg text-pat-badge-danger-text border-pat-badge-danger-bg",
  offline: "bg-pat-badge-danger-bg text-pat-badge-danger-text border-pat-badge-danger-bg",
  error: "bg-pat-badge-danger-bg text-pat-badge-danger-text border-pat-badge-danger-bg",
  suspended: "bg-pat-badge-danger-bg text-pat-badge-danger-text border-pat-badge-danger-bg",
  unknown: "bg-pat-badge-neutral-bg text-pat-badge-neutral-text border-pat-badge-neutral-bg",
  not_configured: "bg-pat-badge-neutral-bg text-pat-badge-neutral-text border-pat-badge-neutral-bg",
};

export default function StatusBadge({ status, size = "sm" }: { status: string; size?: "sm" | "md" }) {
  const normalized = (status || "unknown").toLowerCase();
  const style = styles[normalized] || styles.unknown;
  const sizeClass = size === "sm" ? "text-[10px] px-1.5 py-0.5" : "text-xs px-2 py-1";
  return (
    <span className={`inline-flex items-center rounded-full border font-medium ${sizeClass} ${style}`}>
      {normalized.charAt(0).toUpperCase() + normalized.slice(1)}
    </span>
  );
}
