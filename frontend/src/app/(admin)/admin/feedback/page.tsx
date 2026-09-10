"use client";

import { useState } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import StatusBadge from "@/components/ui/status-badge";
import { format } from "date-fns";
import { IconStar, IconStarFilled, IconEye, IconEyeOff, IconChecks } from "@tabler/icons-react";

interface FeedbackRow {
  id: string;
  email: string;
  full_name: string | null;
  category: string;
  rating: number;
  message: string;
  status: string;
  featured: boolean;
  featured_at: string | null;
  admin_note: string | null;
  created_at: string;
  reviewed_at: string | null;
}

interface Stats {
  total: number;
  new: number;
  hidden: number;
  featured: number;
  avg_rating: string;
  detractors: number;
  promoters: number;
}

const CATEGORY_LABEL: Record<string, string> = {
  bug: "Bug", feature_request: "Feature", usability: "Usability",
  performance: "Performance", billing: "Billing", other: "Other",
};

export default function AdminFeedbackPage() {
  const [statusFilter, setStatusFilter] = useState<string>("ALL");
  const [busyId, setBusyId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const qc = useQueryClient();

  const statsQ = useQuery<Stats>({
    queryKey: ["admin-feedback-stats"],
    queryFn: async () => (await customInstance.get("/admin/feedback/stats")).data,
  });

  const listQ = useQuery<{ items: FeedbackRow[]; total: number }>({
    queryKey: ["admin-feedback", statusFilter],
    queryFn: async () => {
      const p = statusFilter === "ALL" ? "" : `&status=${statusFilter}`;
      return (await customInstance.get(`/admin/feedback?limit=50${p}`)).data;
    },
  });

  const invalidate = () => {
    qc.invalidateQueries({ queryKey: ["admin-feedback"] });
    qc.invalidateQueries({ queryKey: ["admin-feedback-stats"] });
  };

  const act = async (id: string, action: string, body?: Record<string, unknown>) => {
    setBusyId(id);
    setError(null);
    try {
      await customInstance.post(`/admin/feedback/${id}/${action}`, body ?? {});
      invalidate();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Action failed");
    } finally {
      setBusyId(null);
    }
  };

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-xl font-bold text-pat-text-primary">Customer Feedback</h1>
        <p className="text-sm text-pat-text-secondary mt-1">
          Moderation + the Featured toggle. Featured items can be surfaced on marketing surfaces (first name only).
        </p>
      </div>

      {/* Stats tiles */}
      <div className="grid grid-cols-2 md:grid-cols-5 gap-3">
        <div className="rounded-lg border border-pat-border bg-pat-card p-4">
          <div className="text-xs text-pat-text-muted">Total</div>
          <div className="text-2xl font-bold text-pat-text-primary">{statsQ.data?.total ?? "…"}</div>
        </div>
        <div className="rounded-lg border border-pat-border bg-pat-card p-4">
          <div className="text-xs text-pat-text-muted">New (unreviewed)</div>
          <div className={`text-2xl font-bold ${(statsQ.data?.new ?? 0) > 0 ? "text-pat-info" : "text-pat-text-muted"}`}>
            {statsQ.data?.new ?? "…"}
          </div>
        </div>
        <div className="rounded-lg border border-pat-border bg-pat-card p-4">
          <div className="text-xs text-pat-text-muted">Average rating</div>
          <div className="text-2xl font-bold text-pat-warning">{statsQ.data?.avg_rating ?? "…"}</div>
        </div>
        <div className="rounded-lg border border-pat-border bg-pat-card p-4">
          <div className="text-xs text-pat-text-muted">Featured</div>
          <div className="text-2xl font-bold text-pat-success">{statsQ.data?.featured ?? "…"}</div>
        </div>
        <div className="rounded-lg border border-pat-border bg-pat-card p-4">
          <div className="text-xs text-pat-text-muted">Detractors (1-2★)</div>
          <div className={`text-2xl font-bold ${(statsQ.data?.detractors ?? 0) > 0 ? "text-red-500" : "text-pat-success"}`}>
            {statsQ.data?.detractors ?? "…"}
          </div>
        </div>
      </div>

      {/* Status filter */}
      <div className="flex flex-wrap gap-2">
        {["ALL", "new", "reviewed", "hidden"].map((f) => (
          <button key={f} onClick={() => setStatusFilter(f)}
            className={`text-xs px-3 py-1.5 rounded transition-colors ${
              statusFilter === f ? "bg-primary text-primary-foreground" : "bg-pat-bg-surface-secondary text-pat-text-primary hover:bg-pat-bg-surface-secondary"
            }`}>
            {f === "ALL" ? "All" : f}
          </button>
        ))}
        <span className="text-xs text-pat-text-muted ml-auto self-center">{listQ.data?.total ?? 0} items</span>
      </div>

      {error && (
        <div className="text-xs text-red-500 rounded-md border border-red-500/30 bg-red-500/10 px-3 py-2">{error}</div>
      )}

      {/* Feedback table */}
      <div className="overflow-x-auto border border-pat-border rounded-lg">
        <table className="w-full text-sm text-left">
          <thead className="bg-pat-bg-surface text-pat-text-secondary uppercase text-xs">
            <tr>
              <th className="px-4 py-3 font-medium">Client</th>
              <th className="px-3 py-3 font-medium">Category</th>
              <th className="px-3 py-3 font-medium">Rating</th>
              <th className="px-3 py-3 font-medium">Message</th>
              <th className="px-3 py-3 font-medium">Status</th>
              <th className="px-3 py-3 font-medium">Date</th>
              <th className="px-3 py-3 font-medium">Moderate</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-pat-border">
            {(listQ.data?.items?.length ?? 0) === 0 ? (
              <tr><td colSpan={7} className="px-4 py-8 text-center text-pat-text-muted text-sm">No feedback matches this filter</td></tr>
            ) : (
              listQ.data!.items.map((f) => (
                <tr key={f.id} className={`hover:bg-pat-table-hover transition-colors ${f.status === "hidden" ? "opacity-60" : ""}`}>
                  <td className="px-4 py-3">
                    <div className="text-xs text-pat-text-primary">{f.full_name || f.email}</div>
                    <div className="text-[10px] text-pat-text-muted">{f.email}</div>
                  </td>
                  <td className="px-3 py-3 text-xs text-pat-text-secondary">{CATEGORY_LABEL[f.category] ?? f.category}</td>
                  <td className="px-3 py-3 text-xs text-pat-warning tabular-nums">{f.rating}/5</td>
                  <td className="px-3 py-3 text-xs text-pat-text-secondary max-w-[280px]" title={f.message}>
                    <span className="line-clamp-2">{f.message}</span>
                    {f.admin_note && (
                      <span className="block mt-1 text-[10px] text-pat-info" title={f.admin_note}>Note: {f.admin_note}</span>
                    )}
                  </td>
                  <td className="px-3 py-3">
                    <StatusBadge status={f.status === "hidden" ? "HIDDEN" : f.status === "reviewed" ? "REVIEWED" : "NEW"} />
                    {f.featured && (
                      <span className="mt-1 flex items-center gap-1 text-[10px] px-1.5 py-0.5 rounded-full border border-pat-warning/40 bg-pat-warning/10 text-pat-warning font-medium w-fit">
                        <IconStarFilled size={10} /> Featured
                      </span>
                    )}
                  </td>
                  <td className="px-3 py-3 text-xs tabular-nums text-pat-text-muted">
                    {format(new Date(f.created_at), "MMM d, HH:mm")} UTC
                  </td>
                  <td className="px-3 py-3">
                    <div className="flex items-center gap-1">
                      {/* Featured toggle — the requirement */}
                      <button
                        onClick={() => act(f.id, "featured", { featured: !f.featured })}
                        disabled={busyId === f.id}
                        title={f.featured ? "Remove from Featured" : "Mark as Featured"}
                        className={`text-xs px-2 py-1 rounded border font-medium disabled:opacity-40 transition-colors ${
                          f.featured
                            ? "bg-pat-warning/15 text-pat-warning border-pat-warning/40"
                            : "bg-transparent text-pat-text-muted border-pat-border hover:border-pat-warning/40 hover:text-pat-warning"
                        }`}
                      >
                        {busyId === f.id ? "…" : f.featured ? <IconStarFilled size={14} /> : <IconStar size={14} />}
                      </button>
                      {f.status !== "reviewed" && (
                        <button
                          onClick={() => act(f.id, "status", { status: "reviewed" })}
                          disabled={busyId === f.id}
                          title="Mark reviewed"
                          className="text-xs px-2 py-1 rounded border border-pat-border text-pat-text-secondary hover:text-pat-success hover:border-pat-success/40 disabled:opacity-40 transition-colors"
                        >
                          <IconChecks size={14} />
                        </button>
                      )}
                      {f.status !== "hidden" ? (
                        <button
                          onClick={() => act(f.id, "status", { status: "hidden" })}
                          disabled={busyId === f.id}
                          title="Hide (soft-hide, reversible)"
                          className="text-xs px-2 py-1 rounded border border-pat-border text-pat-text-secondary hover:text-red-500 hover:border-red-500/40 disabled:opacity-40 transition-colors"
                        >
                          <IconEyeOff size={14} />
                        </button>
                      ) : (
                        <button
                          onClick={() => act(f.id, "status", { status: "reviewed" })}
                          disabled={busyId === f.id}
                          title="Un-hide (back to reviewed)"
                          className="text-xs px-2 py-1 rounded border border-pat-border text-pat-text-secondary hover:text-pat-success hover:border-pat-success/40 disabled:opacity-40 transition-colors"
                        >
                          <IconEye size={14} />
                        </button>
                      )}
                    </div>
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      <p className="text-[11px] text-pat-text-secondary">
        Hiding is reversible (soft-hide, history preserved) · every moderation action is audit-logged · featured items are
        ranked first in the list.
      </p>
    </div>
  );
}