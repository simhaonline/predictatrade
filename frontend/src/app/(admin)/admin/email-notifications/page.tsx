"use client";

import { useCallback, useEffect, useState } from "react";
import { getAccessToken } from "@/lib/auth";

const API_BASE = process.env.NEXT_PUBLIC_API_BASE_URL || "/api/v1";

type CampaignType = "alert" | "newsletter" | "marketing";
type Audience = "all_clients" | "active_subscribers" | "marketing_opt_in";

interface AudienceCounts {
  all_clients: number;
  active_subscribers: number;
  marketing_opt_in: number;
}

interface CampaignRow {
  id: string;
  subject: string;
  campaign_type: CampaignType;
  audience: Audience;
  status: string;
  total_recipients: number;
  sent_count: number;
  failed_count: number;
  skipped_count: number;
  created_at: string;
  sent_at: string | null;
  created_by_email: string | null;
}

const TYPE_LABELS: Record<string, string> = {
  alert: "Service Alert",
  newsletter: "Newsletter",
  marketing: "Marketing",
};

// Badge style contract used across the admin console (10px pill + border).
const TYPE_BADGE: Record<string, string> = {
  alert: "bg-pat-info/10 text-pat-info border-pat-info/30",
  newsletter: "bg-pat-success/10 text-pat-success border-pat-success/30",
  marketing: "bg-pat-warning/10 text-pat-warning border-pat-warning/30",
};

const STATUS_BADGE: Record<string, string> = {
  draft: "bg-pat-badge-neutral-bg text-pat-badge-neutral-text border-pat-border",
  sending: "bg-pat-info/10 text-pat-info border-pat-info/30",
  sent: "bg-pat-success/10 text-pat-success border-pat-success/30",
  failed: "bg-red-500/10 text-red-500 border-red-500/30",
  cancelled: "bg-pat-badge-neutral-bg text-pat-badge-neutral-text border-pat-border",
};

const AUDIENCE_LABEL: Record<Audience, string> = {
  all_clients: "All Clients",
  active_subscribers: "Active Subscribers",
  marketing_opt_in: "Marketing Opt-in",
};

const TYPE_LABEL = (t: string) => TYPE_LABELS[t] ?? t;

export default function EmailNotificationsPage() {
  const [counts, setCounts] = useState<AudienceCounts | null>(null);
  const [campaigns, setCampaigns] = useState<CampaignRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [sending, setSending] = useState(false);

  // compose form
  const [campaignType, setCampaignType] = useState<CampaignType>("newsletter");
  const [audience, setAudience] = useState<Audience>("all_clients");
  const [subject, setSubject] = useState("");
  const [bodyHtml, setBodyHtml] = useState("");
  const [bodyText, setBodyText] = useState("");

  const authedFetch = useCallback(
    async (path: string, init?: RequestInit) => {
      const token = getAccessToken();
      const res = await fetch(`${API_BASE}${path}`, {
        ...init,
        headers: {
          "Content-Type": "application/json",
          ...(token ? { Authorization: `Bearer ${token}` } : {}),
          ...(init?.headers ?? {}),
        },
      });
      if (!res.ok) {
        const data = await res.json().catch(() => ({}));
        throw new Error(data.message || `Request failed (${res.status})`);
      }
      return res.json();
    },
    [],
  );

  const refresh = useCallback(async () => {
    setLoading(true);
    try {
      const [a, l] = await Promise.all([
        authedFetch("/admin/email-campaigns/audience"),
        authedFetch("/admin/email-campaigns?limit=50"),
      ]);
      setCounts(a);
      setCampaigns(l.campaigns ?? []);
      setError(null);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load");
    } finally {
      setLoading(false);
    }
  }, [authedFetch]);

  useEffect(() => {
    refresh();
  }, [refresh]);

  const sendNow = async () => {
    setSending(true);
    setError(null);
    setNotice(null);
    try {
      const { id } = await authedFetch("/admin/email-campaigns", {
        method: "POST",
        body: JSON.stringify({ subject, bodyHtml, bodyText, campaignType, audience }),
      });
      const result = await authedFetch(`/admin/email-campaigns/${id}/send`, { method: "POST" });
      setNotice(
        `Sent: ${result.sent} delivered, ${result.failed} failed, ${result.skipped} skipped (unsubscribed) of ${result.total} recipients.`,
      );
      setSubject("");
      setBodyHtml("");
      setBodyText("");
      setConfirmOpen(false);
      await refresh();
    } catch (e) {
      setError(e instanceof Error ? e.message : "Send failed");
      setConfirmOpen(false);
    } finally {
      setSending(false);
    }
  };

  const complianceNote =
    campaignType === "alert"
      ? "Service alerts are delivered to the selected audience without opt-out filtering (operational communication)."
      : "Newsletter/marketing sends automatically exclude addresses that unsubscribed, and carry a List-Unsubscribe header.";

  const canSend = subject.trim() !== "" && bodyHtml.trim() !== "" && bodyText.trim() !== "" && !sending;

  return (
    <div className="space-y-4">
      {/* Page header — matches sibling admin pages */}
      <div>
        <h1 className="text-xl font-bold text-pat-text-primary">Email Notifications</h1>
        <p className="text-sm text-pat-text-secondary mt-1">
          Send service alerts, newsletters and marketing campaigns to clients via the platform mail relay.
        </p>
      </div>

      {/* Audience counts — same tile pattern as MT Client Connectivity */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
        <div className="rounded-lg border border-pat-border bg-pat-card p-4">
          <div className="text-xs text-pat-text-muted">All Clients (active accounts)</div>
          <div className="text-2xl font-bold text-pat-text-primary">
            {loading ? "…" : counts?.all_clients ?? "—"}
          </div>
        </div>
        <div className="rounded-lg border border-pat-border bg-pat-card p-4">
          <div className="text-xs text-pat-text-muted">Active Subscribers</div>
          <div className="text-2xl font-bold text-emerald-500">
            {loading ? "…" : counts?.active_subscribers ?? "—"}
          </div>
        </div>
        <div className="rounded-lg border border-pat-border bg-pat-card p-4">
          <div className="text-xs text-pat-text-muted">Marketing Opt-in (consented)</div>
          <div className="text-2xl font-bold text-pat-info">
            {loading ? "…" : counts?.marketing_opt_in ?? "—"}
          </div>
        </div>
      </div>

      {/* Composer */}
      <div className="rounded-lg border border-pat-border bg-pat-card p-4 space-y-4">
        <div className="text-sm font-medium text-pat-text-primary">Compose Campaign</div>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
          <div>
            <label className="block text-xs text-pat-text-muted mb-1.5">Campaign Type</label>
            <select
              value={campaignType}
              onChange={(e) => setCampaignType(e.target.value as CampaignType)}
              className="w-full rounded-md border border-pat-input-border bg-pat-input-bg px-3 py-2 text-sm text-pat-input-text"
            >
              <option value="alert">Service Alert (all clients, operational)</option>
              <option value="newsletter">Newsletter (respects unsubscribes)</option>
              <option value="marketing">Marketing / Promotion (respects unsubscribes)</option>
            </select>
          </div>
          <div>
            <label className="block text-xs text-pat-text-muted mb-1.5">Audience</label>
            <select
              value={audience}
              onChange={(e) => setAudience(e.target.value as Audience)}
              className="w-full rounded-md border border-pat-input-border bg-pat-input-bg px-3 py-2 text-sm text-pat-input-text"
            >
              <option value="all_clients">All Clients{counts ? ` (${counts.all_clients})` : ""}</option>
              <option value="active_subscribers">Active Subscribers{counts ? ` (${counts.active_subscribers})` : ""}</option>
              <option value="marketing_opt_in">Marketing Opt-in{counts ? ` (${counts.marketing_opt_in})` : ""}</option>
            </select>
          </div>
        </div>

        <div>
          <label className="block text-xs text-pat-text-muted mb-1.5">Subject</label>
          <input
            value={subject}
            onChange={(e) => setSubject(e.target.value)}
            placeholder="e.g. Scheduled maintenance window — Saturday 02:00 UTC"
            className="w-full rounded-md border border-pat-input-border bg-pat-input-bg px-3 py-2 text-sm text-pat-input-text placeholder:text-pat-text-muted"
          />
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
          <div>
            <label className="block text-xs text-pat-text-muted mb-1.5">HTML Body</label>
            <textarea
              value={bodyHtml}
              onChange={(e) => setBodyHtml(e.target.value)}
              rows={10}
              placeholder="<p>Hello {{name}}, …</p>"
              className="w-full rounded-md border border-pat-input-border bg-pat-input-bg px-3 py-2 text-xs font-mono text-pat-input-text placeholder:text-pat-text-muted"
            />
          </div>
          <div>
            <label className="block text-xs text-pat-text-muted mb-1.5">Plain-Text Body (required fallback)</label>
            <textarea
              value={bodyText}
              onChange={(e) => setBodyText(e.target.value)}
              rows={10}
              placeholder="Hello, …"
              className="w-full rounded-md border border-pat-input-border bg-pat-input-bg px-3 py-2 text-xs text-pat-input-text placeholder:text-pat-text-muted"
            />
          </div>
        </div>

        <div className="rounded-md bg-pat-bg-surface-secondary px-3 py-2.5 text-xs text-pat-text-secondary">
          {complianceNote}
        </div>

        <div className="flex flex-wrap items-center gap-3">
          <button
            onClick={() => setConfirmOpen(true)}
            disabled={!canSend}
            className="text-sm px-4 py-2 rounded-md bg-pat-primary text-pat-primary-foreground hover:bg-pat-primary-hover disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            {sending ? "Sending…" : "Review & Send"}
          </button>
          {error && (
            <span className="text-xs text-red-500 rounded-md border border-red-500/30 bg-red-500/10 px-2 py-1">
              {error}
            </span>
          )}
          {notice && (
            <span className="text-xs text-pat-success rounded-md border border-pat-success/30 bg-pat-success/10 px-2 py-1">
              {notice}
            </span>
          )}
        </div>

        {confirmOpen && (
          <div className="rounded-md border border-pat-warning/40 bg-pat-warning/10 p-4 space-y-2">
            <div className="text-sm font-medium text-pat-text-primary">
              Send “{subject || "(no subject)"}” as {TYPE_LABEL(campaignType)} to {AUDIENCE_LABEL[audience]}?
            </div>
            <div className="text-xs text-pat-text-secondary">
              This emails real clients through the platform relay. Unsubscribed addresses are excluded for
              newsletter/marketing types. The campaign is recorded in the audit trail.
            </div>
            <div className="flex gap-2 pt-1">
              <button
                onClick={sendNow}
                disabled={sending}
                className="text-xs px-3 py-1.5 rounded-md bg-pat-warning text-pat-text-inverse font-semibold disabled:opacity-40 transition-colors"
              >
                {sending ? "Sending…" : "Confirm Send"}
              </button>
              <button
                onClick={() => setConfirmOpen(false)}
                disabled={sending}
                className="text-xs px-3 py-1.5 rounded-md border border-pat-border text-pat-text-secondary hover:bg-pat-bg-surface-secondary transition-colors"
              >
                Cancel
              </button>
            </div>
          </div>
        )}
      </div>

      {/* History — same table pattern as MT Client Connectivity */}
      <div className="overflow-x-auto border border-pat-border rounded-lg">
        <div className="px-4 py-3 border-b border-pat-border bg-pat-bg-surface text-sm font-medium text-pat-text-primary">
          Campaign History
        </div>
        <table className="w-full text-sm text-left">
          <thead className="bg-pat-bg-surface text-pat-text-secondary uppercase text-xs">
            <tr>
              <th className="px-4 py-3 font-medium">Subject</th>
              <th className="px-3 py-3 font-medium">Type</th>
              <th className="px-3 py-3 font-medium">Audience</th>
              <th className="px-3 py-3 font-medium">Status</th>
              <th className="px-3 py-3 font-medium">Sent / Total</th>
              <th className="px-3 py-3 font-medium">Date</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-pat-border">
            {campaigns.length === 0 ? (
              <tr>
                <td colSpan={6} className="px-4 py-8 text-center text-pat-text-muted text-sm">
                  {loading ? "Loading campaigns…" : "No campaigns sent yet"}
                </td>
              </tr>
            ) : (
              campaigns.map((c) => (
                <tr key={c.id} className="hover:bg-pat-table-hover transition-colors">
                  <td className="px-4 py-3 text-xs text-pat-text-primary">{c.subject}</td>
                  <td className="px-3 py-3">
                    <span className={`text-[10px] px-1.5 py-0.5 rounded-full border font-medium ${TYPE_BADGE[c.campaign_type] ?? ""}`}>
                      {TYPE_LABEL(c.campaign_type)}
                    </span>
                  </td>
                  <td className="px-3 py-3 text-xs text-pat-text-secondary">
                    {AUDIENCE_LABEL[c.audience as Audience] ?? c.audience}
                  </td>
                  <td className="px-3 py-3">
                    <span className={`text-[10px] px-1.5 py-0.5 rounded-full border font-medium ${STATUS_BADGE[c.status] ?? STATUS_BADGE.draft}`}>
                      {c.status}
                    </span>
                  </td>
                  <td className="px-3 py-3 text-xs tabular-nums text-pat-text-secondary">
                    {c.sent_count}/{c.total_recipients}
                    {c.failed_count > 0 && <span className="text-red-500"> · {c.failed_count} failed</span>}
                    {c.skipped_count > 0 && <span className="text-pat-text-muted"> · {c.skipped_count} skipped</span>}
                  </td>
                  <td className="px-3 py-3 text-xs tabular-nums text-pat-text-muted">
                    {new Date(c.created_at).toLocaleString("en-GB", { timeZone: "UTC" })} UTC
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      <p className="text-[11px] text-pat-text-secondary">
        Delivered via pat-mail-relay (DKIM-signed) · marketing sends respect unsubscribes · every campaign is audit-logged
      </p>
    </div>
  );
}