"use client";

import { useCallback, useEffect, useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import { IconMessage2, IconStar, IconStarFilled } from "@tabler/icons-react";

type Category = "bug" | "feature_request" | "usability" | "performance" | "billing" | "other";

const CATEGORIES: { value: Category; label: string }[] = [
  { value: "feature_request", label: "Feature request" },
  { value: "usability", label: "Usability / UX" },
  { value: "performance", label: "Performance" },
  { value: "bug", label: "Bug report" },
  { value: "billing", label: "Billing" },
  { value: "other", label: "Other" },
];

interface MyFeedback {
  id: string;
  category: string;
  rating: number;
  message: string;
  status: string;
  featured: boolean;
  admin_note: string | null;
  created_at: string;
}

const STATUS_LABEL: Record<string, string> = { new: "Received", reviewed: "Reviewed", hidden: "Hidden" };

export default function FeedbackPage() {
  const [category, setCategory] = useState<Category>("feature_request");
  const [rating, setRating] = useState(5);
  const [hoverRating, setHoverRating] = useState(0);
  const [message, setMessage] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  const mineQ = useQuery<{ items: MyFeedback[] }>({
    queryKey: ["my-feedback"],
    queryFn: async () => (await customInstance.get("/feedback/mine")).data,
  });

  const submit = useCallback(async () => {
    setError(null);
    setSuccess(null);
    if (message.trim().length < 10) {
      setError("Message must be at least 10 characters.");
      return;
    }
    setSubmitting(true);
    try {
      const res = (await customInstance.post("/feedback", { category, rating, message })).data as { message?: string };
      setSuccess(res.message || "Thank you — your feedback was received.");
      setMessage("");
      setRating(5);
      mineQ.refetch();
    } catch (e) {
      const msg = e instanceof Error ? e.message : "Submit failed";
      setError(msg.includes("limit reached") ? msg : msg);
    } finally {
      setSubmitting(false);
    }
  }, [category, rating, message, mineQ]);

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-xl font-bold text-pat-text-primary">Feedback</h1>
        <p className="text-sm text-pat-text-secondary mt-1">
          Tell us what works, what doesn&apos;t, and what you want next. Feedback goes straight to the product team.
        </p>
      </div>

      {/* Composer */}
      <div className="rounded-lg border border-pat-border bg-pat-card p-4 space-y-4">
        <div className="flex items-center gap-2 text-sm font-medium text-pat-text-primary">
          <IconMessage2 size={18} className="text-pat-info" /> Share your feedback
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
          <div>
            <label className="block text-xs text-pat-text-muted mb-1.5">Category</label>
            <select
              value={category}
              onChange={(e) => setCategory(e.target.value as Category)}
              className="w-full rounded-md border border-pat-input-border bg-pat-input-bg px-3 py-2 text-sm text-pat-input-text"
            >
              {CATEGORIES.map((c) => (
                <option key={c.value} value={c.value}>{c.label}</option>
              ))}
            </select>
          </div>
          <div>
            <label className="block text-xs text-pat-text-muted mb-1.5">Overall rating</label>
            <div className="flex items-center gap-1 h-[38px]">
              {[1, 2, 3, 4, 5].map((n) => (
                <button
                  key={n}
                  type="button"
                  onClick={() => setRating(n)}
                  onMouseEnter={() => setHoverRating(n)}
                  onMouseLeave={() => setHoverRating(0)}
                  aria-label={`Rate ${n} of 5`}
                  className="p-0.5 transition-transform hover:scale-110"
                >
                  {n <= (hoverRating || rating) ? (
                    <IconStarFilled size={22} className="text-pat-warning" />
                  ) : (
                    <IconStar size={22} className="text-pat-text-muted" />
                  )}
                </button>
              ))}
              <span className="ml-2 text-xs text-pat-text-secondary">{rating}/5</span>
            </div>
          </div>
        </div>

        <div>
          <label className="block text-xs text-pat-text-muted mb-1.5">
            Message <span className="text-pat-text-muted">({message.length}/2000, min 10)</span>
          </label>
          <textarea
            value={message}
            onChange={(e) => setMessage(e.target.value.slice(0, 2000))}
            rows={6}
            placeholder="What should we improve? What works well? Be specific — concrete feedback ships faster."
            className="w-full rounded-md border border-pat-input-border bg-pat-input-bg px-3 py-2 text-sm text-pat-input-text placeholder:text-pat-text-muted"
          />
        </div>

        <div className="flex flex-wrap items-center gap-3">
          <button
            onClick={submit}
            disabled={submitting || message.trim().length < 10}
            className="text-sm px-4 py-2 rounded-md bg-pat-primary text-pat-primary-foreground hover:bg-pat-primary-hover disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            {submitting ? "Submitting…" : "Submit feedback"}
          </button>
          {error && (
            <span className="text-xs text-red-500 rounded-md border border-red-500/30 bg-red-500/10 px-2 py-1">{error}</span>
          )}
          {success && (
            <span className="text-xs text-pat-success rounded-md border border-pat-success/30 bg-pat-success/10 px-2 py-1">{success}</span>
          )}
        </div>

        <p className="text-[11px] text-pat-text-muted">
          Limit: 5 submissions per 24 hours. Feedback can&apos;t be edited after submission — the team may feature it on
          the site (first name only, never your email).
        </p>
      </div>

      {/* My submissions */}
      <div className="overflow-x-auto border border-pat-border rounded-lg">
        <div className="px-4 py-3 border-b border-pat-border bg-pat-bg-surface text-sm font-medium text-pat-text-primary">
          My submissions
        </div>
        <table className="w-full text-sm text-left">
          <thead className="bg-pat-bg-surface text-pat-text-secondary uppercase text-xs">
            <tr>
              <th className="px-4 py-3 font-medium">Category</th>
              <th className="px-3 py-3 font-medium">Rating</th>
              <th className="px-3 py-3 font-medium">Message</th>
              <th className="px-3 py-3 font-medium">Status</th>
              <th className="px-3 py-3 font-medium">Date</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-pat-border">
            {(mineQ.data?.items?.length ?? 0) === 0 ? (
              <tr><td colSpan={5} className="px-4 py-8 text-center text-pat-text-muted text-sm">No feedback submitted yet</td></tr>
            ) : (
              mineQ.data!.items.map((f) => (
                <tr key={f.id} className="hover:bg-pat-table-hover transition-colors">
                  <td className="px-4 py-3 text-xs text-pat-text-primary">{CATEGORIES.find((c) => c.value === f.category)?.label ?? f.category}</td>
                  <td className="px-3 py-3 text-xs text-pat-warning tabular-nums">{f.rating}/5</td>
                  <td className="px-3 py-3 text-xs text-pat-text-secondary max-w-[320px] truncate" title={f.message}>{f.message}</td>
                  <td className="px-3 py-3">
                    <span className={`text-[10px] px-1.5 py-0.5 rounded-full border font-medium ${
                      f.status === "hidden" ? "bg-red-500/10 text-red-500 border-red-500/30"
                      : f.status === "reviewed" ? "bg-pat-success/10 text-pat-success border-pat-success/30"
                      : "bg-pat-info/10 text-pat-info border-pat-info/30"
                    }`}>
                      {STATUS_LABEL[f.status] ?? f.status}
                    </span>
                    {f.featured && (
                      <span className="ml-1 text-[10px] px-1.5 py-0.5 rounded-full border border-pat-warning/40 bg-pat-warning/10 text-pat-warning font-medium">
                        ★ Featured
                      </span>
                    )}
                    {f.admin_note && (
                      <span className="block mt-1 text-[10px] text-pat-text-muted max-w-[180px] leading-tight" title={f.admin_note}>
                        Team note: {f.admin_note}
                      </span>
                    )}
                  </td>
                  <td className="px-3 py-3 text-xs tabular-nums text-pat-text-muted">
                    {new Date(f.created_at).toLocaleString("en-GB", { timeZone: "UTC" })} UTC
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}