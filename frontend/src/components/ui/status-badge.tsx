"use client";

interface StatusBadgeProps {
  status: string;
  size?: "sm" | "md";
}

const statusStyles: Record<string, string> = {
  active: "bg-pat-badge-success-bg text-pat-badge-success-text border-pat-badge-success-bg",
  ACTIVE: "bg-pat-badge-success-bg text-pat-badge-success-text border-pat-badge-success-bg",
  inactive: "bg-neutral-500/10 text-pat-text-secondary border-neutral-500/20",
  INACTIVE: "bg-neutral-500/10 text-pat-text-secondary border-neutral-500/20",
  pending: "bg-pat-badge-warning-bg text-pat-badge-warning-text border-pat-badge-warning-bg",
  PENDING: "bg-pat-badge-warning-bg text-pat-badge-warning-text border-pat-badge-warning-bg",
  suspended: "bg-pat-badge-danger-bg text-pat-badge-danger-text border-pat-badge-danger-bg",
  SUSPENDED: "bg-pat-badge-danger-bg text-pat-badge-danger-text border-pat-badge-danger-bg",
  cancelled: "bg-pat-badge-danger-bg text-pat-badge-danger-text border-pat-badge-danger-bg",
  CANCELLED: "bg-pat-badge-danger-bg text-pat-badge-danger-text border-pat-badge-danger-bg",
  expired: "bg-pat-badge-warning-bg text-pat-badge-warning-text border-pat-badge-warning-bg",
  EXPIRED: "bg-pat-badge-warning-bg text-pat-badge-warning-text border-pat-badge-warning-bg",
  verified: "bg-pat-badge-success-bg text-pat-badge-success-text border-pat-badge-success-bg",
  VERIFIED: "bg-pat-badge-success-bg text-pat-badge-success-text border-pat-badge-success-bg",
  unverified: "bg-pat-badge-warning-bg text-pat-badge-warning-text border-pat-badge-warning-bg",
  UNVERIFIED: "bg-pat-badge-warning-bg text-pat-badge-warning-text border-pat-badge-warning-bg",
  paid: "bg-pat-badge-success-bg text-pat-badge-success-text border-pat-badge-success-bg",
  PAID: "bg-pat-badge-success-bg text-pat-badge-success-text border-pat-badge-success-bg",
  unpaid: "bg-pat-badge-danger-bg text-pat-badge-danger-text border-pat-badge-danger-bg",
  UNPAID: "bg-pat-badge-danger-bg text-pat-badge-danger-text border-pat-badge-danger-bg",
  success: "bg-pat-badge-success-bg text-pat-badge-success-text border-pat-badge-success-bg",
  SUCCESS: "bg-pat-badge-success-bg text-pat-badge-success-text border-pat-badge-success-bg",
  failed: "bg-pat-badge-danger-bg text-pat-badge-danger-text border-pat-badge-danger-bg",
  FAILED: "bg-pat-badge-danger-bg text-pat-badge-danger-text border-pat-badge-danger-bg",
  processing: "bg-pat-badge-info-bg text-pat-badge-info-text border-pat-badge-info-bg",
  PROCESSING: "bg-pat-badge-info-bg text-pat-badge-info-text border-pat-badge-info-bg",
  online: "bg-pat-badge-success-bg text-pat-badge-success-text border-pat-badge-success-bg",
  ONLINE: "bg-pat-badge-success-bg text-pat-badge-success-text border-pat-badge-success-bg",
  offline: "bg-pat-badge-danger-bg text-pat-badge-danger-text border-pat-badge-danger-bg",
  OFFLINE: "bg-pat-badge-danger-bg text-pat-badge-danger-text border-pat-badge-danger-bg",
  healthy: "bg-pat-badge-success-bg text-pat-badge-success-text border-pat-badge-success-bg",
  HEALTHY: "bg-pat-badge-success-bg text-pat-badge-success-text border-pat-badge-success-bg",
  degraded: "bg-pat-badge-warning-bg text-pat-badge-warning-text border-pat-badge-warning-bg",
  DEGRADED: "bg-pat-badge-warning-bg text-pat-badge-warning-text border-pat-badge-warning-bg",
  critical: "bg-pat-badge-danger-bg text-pat-badge-danger-text border-pat-badge-danger-bg",
  CRITICAL: "bg-pat-badge-danger-bg text-pat-badge-danger-text border-pat-badge-danger-bg",
  revoked: "bg-pat-badge-danger-bg text-pat-badge-danger-text border-pat-badge-danger-bg",
  REVOKED: "bg-pat-badge-danger-bg text-pat-badge-danger-text border-pat-badge-danger-bg",
  approved: "bg-pat-badge-success-bg text-pat-badge-success-text border-pat-badge-success-bg",
  APPROVED: "bg-pat-badge-success-bg text-pat-badge-success-text border-pat-badge-success-bg",
  rejected: "bg-pat-badge-danger-bg text-pat-badge-danger-text border-pat-badge-danger-bg",
  REJECTED: "bg-pat-badge-danger-bg text-pat-badge-danger-text border-pat-badge-danger-bg",
  detected: "bg-pat-badge-info-bg text-pat-badge-info-text border-pat-badge-info-bg",
  DETECTED: "bg-pat-badge-info-bg text-pat-badge-info-text border-pat-badge-info-bg",
  confirmed: "bg-pat-badge-success-bg text-pat-badge-success-text border-pat-badge-success-bg",
  CONFIRMED: "bg-pat-badge-success-bg text-pat-badge-success-text border-pat-badge-success-bg",
  candidate: "bg-pat-badge-warning-bg text-pat-badge-warning-text border-pat-badge-warning-bg",
  CANDIDATE: "bg-pat-badge-warning-bg text-pat-badge-warning-text border-pat-badge-warning-bg",
  blocked: "bg-pat-badge-danger-bg text-pat-badge-danger-text border-pat-badge-danger-bg",
  BLOCKED: "bg-pat-badge-danger-bg text-pat-badge-danger-text border-pat-badge-danger-bg",
  invalidated: "bg-pat-badge-danger-bg text-pat-badge-danger-text border-pat-badge-danger-bg",
  INVALIDATED: "bg-pat-badge-danger-bg text-pat-badge-danger-text border-pat-badge-danger-bg",
  triggered: "bg-pat-badge-success-bg text-pat-badge-success-text border-pat-badge-success-bg",
  TRIGGERED: "bg-pat-badge-success-bg text-pat-badge-success-text border-pat-badge-success-bg",
  filled: "bg-pat-badge-success-bg text-pat-badge-success-text border-pat-badge-success-bg",
  FILLED: "bg-pat-badge-success-bg text-pat-badge-success-text border-pat-badge-success-bg",
  stopped: "bg-pat-badge-danger-bg text-pat-badge-danger-text border-pat-badge-danger-bg",
  STOPPED: "bg-pat-badge-danger-bg text-pat-badge-danger-text border-pat-badge-danger-bg",
  closed: "bg-neutral-500/10 text-pat-text-secondary border-neutral-500/20",
  CLOSED: "bg-neutral-500/10 text-pat-text-secondary border-neutral-500/20",
  // Admin infrastructure statuses
  live: "bg-pat-badge-success-bg text-pat-badge-success-text border-pat-badge-success-bg",
  LIVE: "bg-pat-badge-success-bg text-pat-badge-success-text border-pat-badge-success-bg",
  connected: "bg-pat-badge-success-bg text-pat-badge-success-text border-pat-badge-success-bg",
  CONNECTED: "bg-pat-badge-success-bg text-pat-badge-success-text border-pat-badge-success-bg",
  operational: "bg-pat-badge-success-bg text-pat-badge-success-text border-pat-badge-success-bg",
  OPERATIONAL: "bg-pat-badge-success-bg text-pat-badge-success-text border-pat-badge-success-bg",
  stale: "bg-pat-badge-warning-bg text-pat-badge-warning-text border-pat-badge-warning-bg",
  STALE: "bg-pat-badge-warning-bg text-pat-badge-warning-text border-pat-badge-warning-bg",
  reconnecting: "bg-pat-badge-warning-bg text-pat-badge-warning-text border-pat-badge-warning-bg",
  RECONNECTING: "bg-pat-badge-warning-bg text-pat-badge-warning-text border-pat-badge-warning-bg",
  maintenance: "bg-pat-badge-warning-bg text-pat-badge-warning-text border-pat-badge-warning-bg",
  MAINTENANCE: "bg-pat-badge-warning-bg text-pat-badge-warning-text border-pat-badge-warning-bg",
  paused: "bg-pat-badge-warning-bg text-pat-badge-warning-text border-pat-badge-warning-bg",
  PAUSED: "bg-pat-badge-warning-bg text-pat-badge-warning-text border-pat-badge-warning-bg",
  halted: "bg-pat-badge-danger-bg text-pat-badge-danger-text border-pat-badge-danger-bg",
  HALTED: "bg-pat-badge-danger-bg text-pat-badge-danger-text border-pat-badge-danger-bg",
  error: "bg-pat-badge-danger-bg text-pat-badge-danger-text border-pat-badge-danger-bg",
  ERROR: "bg-pat-badge-danger-bg text-pat-badge-danger-text border-pat-badge-danger-bg",
  unknown: "bg-pat-badge-neutral-bg text-pat-badge-neutral-text border-pat-badge-neutral-bg",
  UNKNOWN: "bg-pat-badge-neutral-bg text-pat-badge-neutral-text border-pat-badge-neutral-bg",
  not_configured: "bg-pat-badge-neutral-bg text-pat-badge-neutral-text border-pat-badge-neutral-bg",
  NOT_CONFIGURED: "bg-pat-badge-neutral-bg text-pat-badge-neutral-text border-pat-badge-neutral-bg",
  missing: "bg-pat-text-muted/10 text-pat-text-muted border-pat-border",
  MISSING: "bg-pat-text-muted/10 text-pat-text-muted border-pat-border",
  disabled: "bg-pat-text-muted/10 text-pat-text-muted border-pat-border",
  DISABLED: "bg-pat-text-muted/10 text-pat-text-muted border-pat-border",
  unavailable: "bg-pat-danger/10 text-pat-danger border-pat-danger/20",
  UNAVAILABLE: "bg-pat-danger/10 text-pat-danger border-pat-danger/20",
};

export default function StatusBadge({ status, size = "md" }: StatusBadgeProps) {
  const normalized = status || "unknown";
  const style = statusStyles[normalized] || "bg-neutral-500/10 text-pat-text-secondary border-neutral-500/20";
  const sizeClass = size === "sm" ? "text-[10px] px-1.5 py-0.5" : "text-xs px-2 py-1";

  return (
    <span className={`inline-flex items-center rounded-full border font-medium ${sizeClass} ${style}`}>
      {normalized.charAt(0).toUpperCase() + normalized.slice(1).toLowerCase()}
    </span>
  );
}
