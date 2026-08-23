"use client";
import Link from "next/link";
import { useState } from "react";
import { toast } from "sonner";
import AccessibilitySettings from "@/components/accessibility-settings";
import {
  IconShieldLock,
  IconBell,
  IconHeadset,
  IconWallet,
  IconLicense,
  IconKey,
  IconArrowRight,
} from "@tabler/icons-react";
import { DegradedNote } from "@/components/ui/tabs";
import { updateMyProfile } from "@/lib/admin-api";

const QUICK_LINKS = [
  { href: "/dashboard/security", label: "Security & MFA", desc: "MFA, sessions, trusted devices", icon: IconShieldLock },
  { href: "/dashboard/notifications", label: "Notifications", desc: "Alert category preferences", icon: IconBell },
  { href: "/dashboard/support", label: "Support", desc: "Contact the support team", icon: IconHeadset },
  { href: "/dashboard/payouts", label: "Payouts", desc: "Request & review payouts", icon: IconWallet },
  { href: "/dashboard/license", label: "License", desc: "License key & activations", icon: IconLicense },
];

export default function UserSettingsPage() {
  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const [busy, setBusy] = useState(false);

  const changePassword = async () => {
    if (!current || !next) {
      toast.error("Enter both current and new password.");
      return;
    }
    if (next.length < 8) {
      toast.error("New password must be at least 8 characters.");
      return;
    }
    setBusy(true);
    try {
      // PATCH /users/me currently only supports displayName/timezone; password
      // change is handled via the account recovery (forgot/reset) flow.
      await updateMyProfile({ displayName: "" });
      toast.error("Password change is not supported by this endpoint yet. Use Forgot Password to reset.");
    } catch {
      toast.error("Password change is not supported by this endpoint yet.");
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-bold text-pat-text-primary">Settings</h1>
        <p className="text-sm text-pat-text-secondary mt-1">Account preferences, security and support shortcuts.</p>
      </div>

      <div>
        <h2 className="mb-3 text-sm font-semibold text-pat-text-primary">Quick links</h2>
        <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
          {QUICK_LINKS.map((q) => (
            <Link key={q.href} href={q.href} className="flex items-center justify-between rounded-lg border border-pat-border bg-pat-bg-surface p-4 hover:border-pat-border/80 transition-colors">
              <div className="flex items-center gap-3">
                <div className="flex items-center justify-center w-9 h-9 rounded-lg bg-pat-info/10">
                  <q.icon size={18} className="text-pat-info" />
                </div>
                <div>
                  <div className="text-sm font-medium text-pat-text-primary">{q.label}</div>
                  <div className="text-xs text-pat-text-muted">{q.desc}</div>
                </div>
              </div>
              <IconArrowRight size={16} className="text-pat-text-muted" />
            </Link>
          ))}
        </div>
      </div>

      <div className="rounded-lg border border-pat-border bg-pat-bg-surface p-5 space-y-4">
        <div className="flex items-center gap-2 text-sm font-medium text-pat-text-primary">
          <IconKey size={18} className="text-pat-warning" /> Password
        </div>
        <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
          <input type="password" value={current} onChange={(e) => setCurrent(e.target.value)} placeholder="Current password" className="rounded-md border border-pat-input-border bg-pat-input-bg px-3 py-2 text-sm text-pat-input-text outline-none focus:border-primary" />
          <input type="password" value={next} onChange={(e) => setNext(e.target.value)} placeholder="New password (min 8)" className="rounded-md border border-pat-input-border bg-pat-input-bg px-3 py-2 text-sm text-pat-input-text outline-none focus:border-primary" />
        </div>
        <button onClick={changePassword} disabled={busy} className="rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground disabled:opacity-50">
          {busy ? "Checking…" : "Change password"}
        </button>
        <DegradedNote>
          The profile endpoint does not yet accept a password field. To change your password, use the
          <strong> Forgot Password</strong> flow on the login page, which securely resets credentials.
        </DegradedNote>
      </div>

      <div>
        <h2 className="mb-3 text-sm font-semibold text-pat-text-primary">Accessibility</h2>
        <AccessibilitySettings />
      </div>
    </div>
  );
}
