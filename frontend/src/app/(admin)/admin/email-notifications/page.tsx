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

const TYPE_STYLE: Record<string, string> = {
  alert: "bg-pat-info/15 text-pat-info",
  newsletter: "bg-pat-success/15 text-pat-success",
  marketing: "bg-pat-warning/15 text-pat-warning",
};

const STATUS_STYLE: Record<string, string> = {
  draft: "bg-white/5 text-pat-text-muted",
  sending: "bg-pat-info/15 text-pat-info",
  sent: "bg-pat-success/15 text-pat-success",
  failed: "bg-red-500/15 text-red-400",
  cancelled: "bg-white/5 text-pat-text-muted",
};

const AUDIENCE_LABEL: Record<Audience, string> = {
  all_clients: "All Clients",
  active_subscribers: "Active Subscribers",
  marketing_opt_in: "Marketing Opt-in",
};

const TYPE_LABEL: Record<CampaignType, string> = {
  alert: "Service Alert",
  newsletter: "Newsletter",
  marketing: "Marketing",
};

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
    try {
      const [a, l] = await Promise.all([
        authedFetch("/admin/email-campaigns/audience"),
        authedFetch("/admin/email-campaigns?limit=50"),
      ]);
      setCounts(a);
      setCampaigns(l.campaigns ?? []);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load");
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
        headers: { "Content-Type": "application/json" },
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

  return (
    <div className="p-6 max-w-5xl mx-auto space-y-6">
      <div>
        <h1 className="text-xl font-semibold text-pat-text">Email Notifications</h1>
        <p className="text-sm text-pat-text-muted mt-1">
          Send service alerts, newsletters and marketing campaigns to clients via the platform mail relay.
        </p>
      </div>

      {/* Audience counts */}
      <div className="grid grid-cols-3 gap-4">
        {counts ? (
          <>
            <div className="rounded-lg border border-pat-border bg-pat-surface p-4">
              <div className="text-2xl font-semibold text-pat-text">{counts.all_clients}</div>
              <div className="text-xs text-pat-text-muted mt-1">All Clients (active accounts)</div>
            </div>
            <div className="rounded-lg border border-pat-surface p-4">
              <div className="text-2xl font-semibold text-pat-text">{counts.active_subscribers}</div>
              <div className="text-xs text-pat-text-muted mt-1">Active Subscribers</div>
            </div>
            <div className="rounded-lg border border-pat-surface p-4">
              <div className="text-2xl font-semibold text-pat-text">{counts.marketing_opt_in}</div>
              <div className="text-xs text-pat-text-muted mt-1">Marketing Opt-in (consented)</div>
            </div>
          </>
        ) : (
          <div className="col-span-3 text-sm text-pat-text-muted">Loading audience…</div>
        )}
      </div>

      {/* Composer */}
      <div className="rounded-lg border border-pat-surface bg-pat-surface/50 p-5 space-y-4">
        <h2 className="text-sm font-medium text-pat-text">Compose Campaign</h2>

        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-xs text-pat-text-muted mb-1">Campaign Type</label>
            <select
              value={campaignType}
              onChange={(e) => setCampaignType(e.target.value as CampaignType)}
              className="w-full bg-black/20 border border-pat-surface rounded px-3 py-2 text-sm text-pat-text"
            >
              <option value="alert">Service Alert (all clients, operational)</option>
              <option value="newsletter">Newsletter (respects unsubscribes)</option>
              <option value="marketing">Marketing / Promotion (respects unsubscribes)</option>
            </select>
          </div>
          <div>
            <label className="block text-xs text-pat-text-muted mb-1">Audience</label>
            <select
              value={audience}
              onChange={(e) => setAudience(e.target.value as Audience)}
              className="w-full bg-black/20 border border-pat-surface rounded px-3 py-2 text-sm text-pat-text"
            >
              <option value="all_clients">All Clients{counts ? ` (${counts.all_clients})` : ""}</option>
              <option value="active_subscribers">Active Subscribers{counts ? ` (${counts.active_subscribers})` : ""}</option>
              <option value="marketing_opt_in">Marketing Opt-in{counts ? ` (${counts.marketing_opt_in})` : ""}</option>
            </select>
          </div>
        </div>

        <div>
          <label className="block text-xs text-pat-text-muted mb-1">Subject</label>
          <input
            value={subject}
            onChange={(e) => setSubject(e.target.value)}
            placeholder="e.g. Scheduled maintenance window — Saturday 02:00 UTC"
            className="w-full bg-black/20 border border-pat-surface rounded px-3 py-2 text-sm text-pat-text"
          />
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-xs text-pat-text-muted mb-1">HTML Body</label>
            <textarea
              value={bodyHtml}
              onChange={(e) => setBodyHtml(e.target.value)}
              rows={10}
              placeholder="<p>Hello {{name}}, …</p>"
              className="w-full bg-black/20 border border-pat-surface rounded px-3 py-2 text-xs font-mono text-pat-text"
            />
          </div>
          <div>
            <label className="block text-xs text-pat-text-muted mb-1">Plain-Text Body (required fallback)</label>
            <textarea
              value={bodyText}
              onChange={(e) => setBodyText(e.target.value)}
              rows={10}
              placeholder="Hello, …"
              className="w-full bg-black/20 border border-pat-surface rounded px-3 py-2 text-xs text-pat-text"
            />
          </div>
        </div>

        <div className="text-xs text-pat-text-muted bg-white/5 rounded p-3">{complianceNote}</div>

        <div className="flex items-center gap-3">
          <button
            onClick={() => setConfirmOpen(true)}
            disabled={!subject.trim() || !bodyHtml.trim() || !bodyText.trim() || sending}
            className="px-4 py-2 rounded bg-pat-primary text-white text-sm font-medium disabled:opacity-40"
          >
            {sending ? "Sending…" : "Review & Send"}
          </button>
          {error && <span className="text-xs text-red-400">{error}</span>}
          {notice && <span className="text-xs text-pat-success">{notice}</span>}
        </div>

        {confirmOpen && (
          <div className="border border-pat-warning/40 bg-pat-warning/10 rounded p-4 text-sm space-y-2">
            <div className="text-pat-text font-medium">
              Send “{subject || "(no subject)"}” as {TYPE_LABELS[campaignType]} to{" "}
              {AUDIENCE_LABEL[audience]}?
            </div>
            <div className="text-xs text-pat-text-muted">
              This emails real clients through the platform relay. Unsubscribed addresses are excluded for
              newsletter/marketing types. The campaign is logged to the audit trail.
            </div>
            <div className="flex gap-2 pt-1">
              <button
                onClick={sendNow}
                disabled={sending}
                className="px-3 py-1.5 rounded bg-pat-warning text-black text-xs font-semibold disabled:opacity-40"
              >
                {sending ? "Sending…" : "Confirm Send"}
              </button>
              <button
                onClick={() => setConfirmOpen(false)}
                disabled={sending}
                className="px-3 py-1.5 rounded border border-pat-surface text-xs"
              >
                Cancel
              </button>
            </div>
          </div>
        )}
      </div>

      {/* History */}
      <div className="rounded-lg border border-pat-surface">
        <div className="px-5 py-3 border-b border-pat-surface text-sm font-medium text-pat-text">
          Campaign History
        </div>
        {campaigns.length === 0 ? (
          <div className="px-5 py-6 text-sm text-pat-text-muted">No campaigns sent yet.</div>
        ) : (
          <table className="w-full text-sm">
            <thead>
              <tr className="text-left text-xs text-pat-text-muted border-b border-pat-surface">
                <th className="px-5 py-2">Subject</th>
                <th className="px-3 py-2">Type</th>
                <th className="px-3 py-2">Audience</th>
                <th className="px-3 py-2">Status</th>
                <th className="px-3 py-2">Sent / Total</th>
                <th className="px-3 py-2">Date</th>
              </tr>
            </thead>
            <tbody>
              {campaigns.map((c) => (
                <tr key={c.id} className="border-b border-pat-surface/50 last:border-0">
                  <td className="px-5 py-2.5 text-pat-text">{c.subject}</td>
                  <td className="px-3 py-2.5">
                    <span className={`text-[10px] px-1.5 py-0.5 rounded-full ${TYPE_STYLE[c.campaign_type] ?? ""}`}>
                      {TYPE_LABELS[c.campaign_type as CampaignType] ?? c.campaign_type}
                    </span>
                  </td>
                  <td className="px-3 py-2.5 text-xs text-pat-text-muted">
                    {AUDIENCE_LABEL[c.audience as Audience] ?? c.audience}
                  </td>
                  <td className="px-3 py-2.5">
                    <span className={`text-[10px] px-1.5 py-0.5 rounded-full ${STATUS_STYLE[c.status] ?? ""}`}>
                      {c.status}
                    </span>
                  </td>
                  <td className="px-3 py-2.5 text-xs text-pat-text-muted">
                    {c.sent_count}/{c.total_recipients}
                    {c.failed_count > 0 && <span className="text-red-400"> · {c.failed_count} failed</span>}
                    {c.skipped_count > 0 && <span> · {c.skipped_count} skipped</span>}
                  </td>
                  <td className="px-3 py-2.5 text-xs text-pat-text-muted">
                    {new Date(c.created_at).toLocaleString("en-GB", { timeZone: "UTC" })} UTC
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  );
}

const TYPE_LABELS: Record<string, string> = {
  alert: "Service Alert",
  newsletter: "Newsletter",
  marketing: "Marketing",
};