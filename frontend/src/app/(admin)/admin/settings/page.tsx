"use client";
import { useState, useEffect } from "react";
import { useAuth } from "@/providers/auth-provider";
import { customInstance } from "@/lib/axios-instance";
import { useTheme } from "next-themes";
import { toast } from "sonner";
import { IconUser, IconLock, IconBell, IconEye, IconEyeOff, IconShieldCheck, IconCopy, IconCheck } from "@tabler/icons-react";
import { QRCodeSVG } from "qrcode.react";
import AccessibilitySettings from "@/components/accessibility-settings";
import StatusBadge from "@/components/admin/status-badge";

type Tab = "profile" | "password" | "mfa" | "notifications" | "accessibility";

export default function AdminSettingsPage() {
  const { user, refreshUser } = useAuth();
  const { theme, setTheme } = useTheme();
  // Deep-link support: ?tab=mfa (used by the login enrollment redirect)
  const initialTab = typeof window !== "undefined" && new URLSearchParams(window.location.search).get("tab");
  const [tab, setTab] = useState<Tab>(initialTab === "mfa" ? "mfa" : "profile");
  const [displayName, setDisplayName] = useState("");
  const [timezone, setTimezone] = useState("");
  const [savingProfile, setSavingProfile] = useState(false);

  // Password state
  const [currentPassword, setCurrentPassword] = useState("");
  const [newPassword, setNewPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [showPasswords, setShowPasswords] = useState(false);
  const [savingPassword, setSavingPassword] = useState(false);

  // MFA setup flow state
  const [mfaSetup, setMfaSetup] = useState<{ secret: string; otpauth: string; recoveryCodes: string[] } | null>(null);
  const [mfaCode, setMfaCode] = useState("");
  const [mfaVerifying, setMfaVerifying] = useState(false);
  const [mfaRecoveryCodes, setMfaRecoveryCodes] = useState<string[] | null>(null);
  const [mfaError, setMfaError] = useState<string | null>(null);

  const startMfaSetup = async () => {
    setMfaError(null);
    try {
      const data = await customInstance.post<{ secret: string; otpauth: string; recoveryCodes: string[] }>("/auth/mfa/setup");
      setMfaSetup(data.data);
      setMfaRecoveryCodes(null);
      setMfaCode("");
    } catch {
      setMfaError("Failed to start MFA setup");
      toast.error("Failed to start MFA setup");
    }
  };

  const verifyMfaCode = async () => {
    if (!mfaCode.trim()) { setMfaError("Enter the 6-digit code from your authenticator app"); return; }
    setMfaVerifying(true);
    setMfaError(null);
    try {
      const data = await customInstance.post<{ mfaEnabled: boolean; recoveryCodes: string[] }>("/auth/mfa/verify", { code: mfaCode.trim() });
      setMfaRecoveryCodes(data.data.recoveryCodes);
      setMfaSetup(null);
      await refreshUser();
      toast.success("MFA enabled");
    } catch {
      setMfaError("Invalid code — try again");
      toast.error("Invalid MFA code");
    } finally {
      setMfaVerifying(false);
    }
  };

  const copyRecoveryCodes = () => {
    if (!mfaRecoveryCodes) return;
    navigator.clipboard?.writeText(mfaRecoveryCodes.join("\n")).catch(() => {});
    toast.success("Recovery codes copied");
  };

  useEffect(() => {
    queueMicrotask(() => {
      if (user) {
        setDisplayName(user.name || "");
        setTimezone(Intl.DateTimeFormat().resolvedOptions().timeZone || "");
      }
    });
  }, [user]);

  const saveProfile = async () => {
    setSavingProfile(true);
    try {
      await customInstance.patch("/users/me", { displayName, timezone });
      await refreshUser();
      toast.success("Profile updated");
    } catch {
      toast.error("Failed to update profile");
    } finally {
      setSavingProfile(false);
    }
  };

  const changePassword = async () => {
    if (newPassword !== confirmPassword) { toast.error("Passwords do not match"); return; }
    if (newPassword.length < 8) { toast.error("Password must be at least 8 characters"); return; }
    setSavingPassword(true);
    try {
      await customInstance.post("/auth/change-password", { currentPassword, newPassword });
      toast.success("Password changed successfully");
      setCurrentPassword(""); setNewPassword(""); setConfirmPassword("");
    } catch {
      toast.error("Failed to change password");
    } finally {
      setSavingPassword(false);
    }
  };

  const tabs: { id: Tab; label: string; icon: typeof IconUser }[] = [
    { id: "profile", label: "Profile", icon: IconUser },
    { id: "password", label: "Password", icon: IconLock },
    { id: "mfa", label: "MFA", icon: IconShieldCheck },
    { id: "notifications", label: "Notifications", icon: IconBell },
    { id: "accessibility", label: "Accessibility", icon: IconEye },
  ];

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-bold text-pat-text-primary">Settings</h1>
        <p className="text-sm text-pat-text-secondary mt-1">Manage your admin account and preferences.</p>
      </div>

      <div className="flex gap-2 flex-wrap">
        {tabs.map((t) => (
          <button key={t.id} onClick={() => setTab(t.id)}
            className={`flex items-center gap-1.5 text-xs px-3 py-1.5 rounded transition-colors ${tab === t.id ? "bg-pat-primary text-pat-primary-foreground" : "bg-pat-bg-surface-secondary text-pat-text-secondary border border-pat-border"}`}>
            <t.icon size={14} /> {t.label}
          </button>
        ))}
      </div>

      {tab === "profile" && (
        <div className="bg-pat-card-bg border border-pat-card-border rounded-lg p-5 shadow-sm max-w-lg space-y-4">
          <div>
            <label className="block text-sm font-medium text-pat-text-primary mb-1">Display Name</label>
            <input type="text" value={displayName} onChange={(e) => setDisplayName(e.target.value)}
              className="w-full rounded-md border border-pat-input-border bg-pat-input-bg px-3 py-2 text-sm text-pat-input-text focus:outline-none focus:ring-2 focus:ring-pat-primary" />
          </div>
          <div>
            <label className="block text-sm font-medium text-pat-text-primary mb-1">Email</label>
            <input type="email" value={user?.email || ""} disabled
              className="w-full rounded-md border border-pat-input-border bg-pat-bg-surface-secondary px-3 py-2 text-sm text-pat-text-muted" />
          </div>
          <div>
            <label className="block text-sm font-medium text-pat-text-primary mb-1">Role</label>
            <input type="text" value={user?.role || ""} disabled
              className="w-full rounded-md border border-pat-input-border bg-pat-bg-surface-secondary px-3 py-2 text-sm text-pat-text-muted" />
          </div>
          <div>
            <label className="block text-sm font-medium text-pat-text-primary mb-1">Timezone</label>
            <input type="text" value={timezone} onChange={(e) => setTimezone(e.target.value)}
              className="w-full rounded-md border border-pat-input-border bg-pat-input-bg px-3 py-2 text-sm text-pat-input-text focus:outline-none focus:ring-2 focus:ring-pat-primary" />
          </div>
          <div>
            <label className="block text-sm font-medium text-pat-text-primary mb-2">Display Theme</label>
            <div className="flex gap-2">
              {[
                { value: "system", label: "System Mode" },
                { value: "light", label: "Light Mode" },
                { value: "dark", label: "Dark Mode" },
              ].map((opt) => (
                <button key={opt.value} onClick={() => setTheme(opt.value)}
                  className={`text-xs px-3 py-1.5 rounded-md font-medium transition-colors ${theme === opt.value ? "bg-pat-primary text-pat-primary-foreground" : "bg-pat-bg-surface-secondary text-pat-text-secondary border border-pat-border"}`}>
                  {opt.label}
                </button>
              ))}
            </div>
          </div>
          <button onClick={saveProfile} disabled={savingProfile}
            className="px-4 py-2 text-sm font-medium bg-pat-primary text-pat-primary-foreground rounded-md hover:bg-pat-primary-hover disabled:opacity-50 transition-colors">
            {savingProfile ? "Saving..." : "Save Profile"}
          </button>
        </div>
      )}

      {tab === "password" && (
        <div className="bg-pat-card-bg border border-pat-card-border rounded-lg p-5 shadow-sm max-w-lg space-y-4">
          <div>
            <label className="block text-sm font-medium text-pat-text-primary mb-1">Current Password</label>
            <div className="relative">
              <input type={showPasswords ? "text" : "password"} value={currentPassword} onChange={(e) => setCurrentPassword(e.target.value)} autoComplete="current-password"
                className="w-full rounded-md border border-pat-input-border bg-pat-input-bg px-3 py-2 text-sm text-pat-input-text focus:outline-none focus:ring-2 focus:ring-pat-primary pr-10" />
              <button onClick={() => setShowPasswords(!showPasswords)} className="absolute right-2 top-1/2 -translate-y-1/2 text-pat-text-muted p-1" aria-label="Toggle password visibility">
                {showPasswords ? <IconEyeOff size={16} /> : <IconEye size={16} />}
              </button>
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium text-pat-text-primary mb-1">New Password</label>
            <input type={showPasswords ? "text" : "password"} value={newPassword} onChange={(e) => setNewPassword(e.target.value)} autoComplete="new-password"
              className="w-full rounded-md border border-pat-input-border bg-pat-input-bg px-3 py-2 text-sm text-pat-input-text focus:outline-none focus:ring-2 focus:ring-pat-primary" placeholder="Minimum 8 characters" />
          </div>
          <div>
            <label className="block text-sm font-medium text-pat-text-primary mb-1">Confirm New Password</label>
            <input type={showPasswords ? "text" : "password"} value={confirmPassword} onChange={(e) => setConfirmPassword(e.target.value)} autoComplete="new-password"
              className="w-full rounded-md border border-pat-input-border bg-pat-input-bg px-3 py-2 text-sm text-pat-input-text focus:outline-none focus:ring-2 focus:ring-pat-primary" />
          </div>
          <button onClick={changePassword} disabled={savingPassword || !currentPassword || !newPassword}
            className="px-4 py-2 text-sm font-medium bg-pat-primary text-pat-primary-foreground rounded-md hover:bg-pat-primary-hover disabled:opacity-50 transition-colors">
            {savingPassword ? "Changing..." : "Change Password"}
          </button>
        </div>
      )}

      {tab === "mfa" && (
        <div className="bg-pat-card-bg border border-pat-card-border rounded-lg p-5 shadow-sm max-w-lg space-y-4">
          <div className="flex items-center justify-between">
            <div>
              <h3 className="text-sm font-medium text-pat-text-primary">Multi-Factor Authentication</h3>
              <p className="text-xs text-pat-text-muted mt-1">Add an extra layer of security to your account.</p>
            </div>
            <StatusBadge status={user?.mfaEnabled ? "active" : "inactive"} />
          </div>

          {mfaRecoveryCodes ? (
            <div className="space-y-3">
              <p className="text-sm text-pat-text-primary font-medium">MFA enabled — save these recovery codes</p>
              <p className="text-xs text-pat-text-muted">Each code can be used once if you lose access to your authenticator. Store them somewhere safe.</p>
              <div className="bg-pat-bg-surface-secondary rounded-md p-3 font-mono text-sm space-y-1">
                {mfaRecoveryCodes.map((c) => <div key={c}>{c}</div>)}
              </div>
              <button onClick={copyRecoveryCodes}
                className="inline-flex items-center gap-2 px-3 py-2 text-sm rounded-md bg-pat-primary text-pat-primary-foreground hover:bg-pat-primary-hover">
                <IconCopy size={14} /> Copy recovery codes
              </button>
            </div>
          ) : mfaSetup ? (
            <div className="space-y-4">
              <p className="text-xs text-pat-text-secondary">Scan this QR code with Google Authenticator / Authy, then enter the 6-digit code to confirm.</p>
              <div className="flex justify-center bg-white rounded-md p-3 w-fit">
                <QRCodeSVG value={mfaSetup.otpauth} size={176} />
              </div>
              <div>
                <p className="text-xs text-pat-text-muted mb-1">Or enter this secret manually:</p>
                <code className="block bg-pat-bg-surface-secondary rounded px-2 py-1 text-xs break-all">{mfaSetup.secret}</code>
              </div>
              <div className="space-y-2">
                <input
                  value={mfaCode}
                  onChange={(e) => setMfaCode(e.target.value)}
                  placeholder="123456"
                  inputMode="numeric"
                  maxLength={6}
                  className="w-full rounded-md border border-pat-input-border bg-pat-input-bg px-3 py-2 text-sm text-pat-input-text"
                />
                {mfaError && <p className="text-xs text-pat-danger">{mfaError}</p>}
                <div className="flex gap-2">
                  <button onClick={verifyMfaCode} disabled={mfaVerifying}
                    className="px-4 py-2 text-sm font-medium bg-pat-primary text-pat-primary-foreground rounded-md hover:bg-pat-primary-hover disabled:opacity-50">
                    {mfaVerifying ? "Verifying…" : "Verify & Enable"}
                  </button>
                  <button onClick={() => { setMfaSetup(null); setMfaCode(""); setMfaError(null); }}
                    className="px-4 py-2 text-sm rounded-md border border-pat-border text-pat-text-secondary">
                    Cancel
                  </button>
                </div>
              </div>
            </div>
          ) : user?.mfaEnabled ? (
            <p className="text-xs text-pat-text-secondary">MFA is currently enabled. Contact support to disable.</p>
          ) : (
            <button onClick={startMfaSetup}
              className="px-4 py-2 text-sm font-medium bg-pat-primary text-pat-primary-foreground rounded-md hover:bg-pat-primary-hover transition-colors">
              Setup MFA
            </button>
          )}
        </div>
      )}

      {tab === "notifications" && (
        <div className="bg-pat-card-bg border border-pat-card-border rounded-lg p-5 shadow-sm max-w-lg">
          <p className="text-sm text-pat-text-muted">Notification preferences will be available in a future update.</p>
        </div>
      )}

      {tab === "accessibility" && <AccessibilitySettings />}
    </div>
  );
}
