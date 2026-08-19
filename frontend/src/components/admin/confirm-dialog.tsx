"use client";
import { IconAlertTriangle } from "@tabler/icons-react";

interface ConfirmDialogProps {
  open: boolean;
  title: string;
  message: string;
  confirmLabel: string;
  onConfirm: () => void;
  onCancel: () => void;
  loading?: boolean;
}

export default function ConfirmDialog({ open, title, message, confirmLabel, onConfirm, onCancel, loading }: ConfirmDialogProps) {
  if (!open) return null;
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50" onClick={onCancel}>
      <div className="bg-pat-bg-surface border border-pat-border rounded-lg shadow-xl max-w-sm w-full mx-4 p-5" onClick={(e) => e.stopPropagation()}>
        <div className="flex items-start gap-3 mb-4">
          <IconAlertTriangle size={24} className="text-pat-warning flex-shrink-0" />
          <div>
            <h3 className="text-sm font-semibold text-pat-text-primary">{title}</h3>
            <p className="text-xs text-pat-text-secondary mt-1">{message}</p>
          </div>
        </div>
        <div className="flex justify-end gap-2">
          <button onClick={onCancel} className="px-3 py-1.5 text-xs font-medium border border-pat-border-strong rounded-md text-pat-text-secondary hover:bg-pat-bg-surface-secondary transition-colors">Cancel</button>
          <button onClick={onConfirm} disabled={loading} className="px-3 py-1.5 text-xs font-medium bg-pat-danger text-white rounded-md hover:opacity-90 disabled:opacity-50 transition-opacity">{loading ? "Processing..." : confirmLabel}</button>
        </div>
      </div>
    </div>
  );
}
