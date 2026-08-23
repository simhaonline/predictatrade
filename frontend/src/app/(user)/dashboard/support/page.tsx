"use client";
import { useState } from "react";
import { toast } from "sonner";
import { IconHeadset, IconMail, IconSend } from "@tabler/icons-react";
import { DegradedNote } from "@/components/ui/tabs";

const SUPPORT_EMAIL = "support@predictatrade.com";

export default function UserSupportPage() {
  const [subject, setSubject] = useState("");
  const [message, setMessage] = useState("");
  const [category, setCategory] = useState("GENERAL");

  const mailto = `mailto:${SUPPORT_EMAIL}?subject=${encodeURIComponent(`[${category}] ${subject}`)}&body=${encodeURIComponent(message)}`;

  const submit = () => {
    if (!subject.trim() || !message.trim()) {
      toast.error("Please add a subject and a message.");
      return;
    }
    window.location.href = mailto;
    toast.success("Opening your email client…");
  };

  return (
    <div className="space-y-5">
      <div>
        <h1 className="text-xl font-bold text-pat-text-primary">Support</h1>
        <p className="text-sm text-pat-text-secondary mt-1">Get help with your account, billing or technical issues.</p>
      </div>

      <DegradedNote>
        An in-app support ticket backend is not available in this build. This form opens your email client pre-addressed
        to our support team — no ticket is created or tracked inside the app.
      </DegradedNote>

      <div className="rounded-lg border border-pat-border bg-pat-bg-surface p-5 space-y-4">
        <div className="flex items-center gap-2 text-sm font-medium text-pat-text-primary">
          <IconHeadset size={18} className="text-pat-info" /> Contact form
        </div>

        <div>
          <label className="text-xs text-pat-text-secondary">Category</label>
          <select
            value={category}
            onChange={(e) => setCategory(e.target.value)}
            className="mt-1 w-full rounded-md border border-pat-input-border bg-pat-input-bg px-3 py-2 text-sm text-pat-input-text outline-none focus:border-primary"
          >
            <option value="GENERAL">General</option>
            <option value="BILLING">Billing</option>
            <option value="TECHNICAL">Technical / MT4-MT5</option>
            <option value="SECURITY">Security</option>
            <option value="REFERRALS">Referrals / Payouts</option>
          </select>
        </div>

        <div>
          <label className="text-xs text-pat-text-secondary">Subject</label>
          <input
            value={subject}
            onChange={(e) => setSubject(e.target.value)}
            placeholder="Brief summary of your issue"
            className="mt-1 w-full rounded-md border border-pat-input-border bg-pat-input-bg px-3 py-2 text-sm text-pat-input-text outline-none focus:border-primary"
          />
        </div>

        <div>
          <label className="text-xs text-pat-text-secondary">Message</label>
          <textarea
            value={message}
            onChange={(e) => setMessage(e.target.value)}
            rows={5}
            placeholder="Describe what happened and any steps to reproduce…"
            className="mt-1 w-full rounded-md border border-pat-input-border bg-pat-input-bg px-3 py-2 text-sm text-pat-input-text outline-none focus:border-primary"
          />
        </div>

        <div className="flex flex-wrap items-center gap-3">
          <button onClick={submit} className="flex items-center gap-1.5 rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground">
            <IconSend size={16} /> Open email to support
          </button>
          <a href={`mailto:${SUPPORT_EMAIL}`} className="flex items-center gap-1.5 text-xs text-pat-info hover:underline">
            <IconMail size={14} /> {SUPPORT_EMAIL}
          </a>
        </div>
      </div>
    </div>
  );
}
