"use client";

export default function EmptyState({ message = "No data found" }: { message?: string }) {
  return (
    <div className="text-center py-8 border border-pat-card-border rounded-lg bg-pat-card-bg">
      <div className="text-sm text-pat-text-muted">{message}</div>
    </div>
  );
}
