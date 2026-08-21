"use client";
import { useState, useEffect } from "react";
import Link from "next/link";
import { IconX, IconLoader2, IconShieldCheck } from "@tabler/icons-react";
import { getApiErrorMessage } from "@/lib/errors";
import { useGuestPreview } from "@/lib/guest-preview-api";
import BrandLogo from "@/components/brand-logo";

const BROKERS = ["Exness", "IC Markets", "XM", "Pepperstone", "Deriv", "Other/None"];

/**
 * Registration modal — the lock screen shown when the guest preview expires.
 *
 * Single screen, fields in spec order: Full name → Email → Phone (optional) →
 * Broker (optional dropdown with free-text "Other"). Three SEPARATE, UNCHECKED
 * consent checkboxes (UAE PDPL compliant): Terms (required), Risk (required),
 * Marketing (optional, visually marked). Links to Privacy Policy + Terms.
 *
 * After submit: email a 6-digit OTP (hashed server-side), 10-min expiry, max 5
 * attempts, 60s resend cooldown. On success: authenticated session + unlock.
 *
 * Generic error messages everywhere — never reveals whether an email exists.
 */
export function RegistrationModal({ onClose }: { onClose?: () => void }) {
  const { register, resend, verify } = useGuestPreview();
  const [step, setStep] = useState<"form" | "otp">("form");
  const [email, setEmail] = useState("");
  const [fullName, setFullName] = useState("");
  const [phone, setPhone] = useState("");
  const [broker, setBroker] = useState("");
  const [brokerOther, setBrokerOther] = useState("");
  const [terms, setTerms] = useState(false);
  const [risk, setRisk] = useState(false);
  const [marketing, setMarketing] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [info, setInfo] = useState("");

  // OTP step state
  const [code, setCode] = useState("");
  const [cooldown, setCooldown] = useState(0);

  useEffect(() => {
    if (cooldown <= 0) return;
    const t = setInterval(() => setCooldown((c) => Math.max(0, c - 1)), 1000);
    return () => clearInterval(t);
  }, [cooldown]);

  const selectedBroker = broker === "Other/None" ? brokerOther.trim() : broker;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setInfo("");
    // Client-side guard mirrors server-side enforcement (defense in depth).
    if (!terms) { setError("Please accept the Terms & Conditions and Privacy Policy."); return; }
    if (!risk) { setError("Please acknowledge the risk disclosure."); return; }
    const normalizedEmail = email.trim().toLowerCase();
    if (!normalizedEmail) { setError("Email is required."); return; }
    setLoading(true);
    try {
      const res = await register({
        fullName: fullName.trim(),
        email: normalizedEmail,
        phone: phone.trim() || undefined,
        broker: selectedBroker || undefined,
        termsAccepted: terms,
        riskAcknowledged: risk,
        marketingOptIn: marketing, // optional
      });
      setInfo(res.message);
      setStep("otp");
      setCooldown(60);
    } catch (err: unknown) {
      setError(getApiErrorMessage(err, "Registration failed. Please try again."));
    } finally {
      setLoading(false);
    }
  };

  const handleVerify = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    setLoading(true);
    try {
      await verify(email.trim().toLowerCase(), code);
      // On success the access token is set; notify the gate to show the social step.
      if (typeof window !== "undefined") {
        window.dispatchEvent(new Event("pat:guest-registered"));
      }
    } catch (err: unknown) {
      setError(getApiErrorMessage(err, "Verification failed."));
    } finally {
      setLoading(false);
    }
  };

  const handleResend = async () => {
    if (cooldown > 0) return;
    setError("");
    setInfo("");
    setLoading(true);
    try {
      const res = await resend(email.trim().toLowerCase());
      setInfo(res.message);
      setCooldown(60);
    } catch (err: unknown) {
      setError(getApiErrorMessage(err, "Could not resend code. Please wait and try again."));
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="fixed inset-0 z-[200] flex items-center justify-center bg-black/70 backdrop-blur-sm" role="dialog" aria-modal="true" aria-label="Register to continue">
      <div className="bg-pat-bg-surface border border-pat-card-border rounded-xl shadow-2xl w-full max-w-md mx-4 max-h-[90vh] overflow-y-auto">
        <div className="flex items-center justify-between p-4 border-b border-pat-border">
          <div className="flex items-center gap-2">
            <BrandLogo />
            <span className="text-sm font-semibold text-pat-text-primary">Register to keep access</span>
          </div>
          {onClose && (
            <button onClick={onClose} className="p-1 hover:bg-pat-bg-surface-secondary rounded text-pat-text-muted" aria-label="Close">
              <IconX size={18} />
            </button>
          )}
        </div>

        <div className="p-4 space-y-4">
          {step === "form" && (
            <form onSubmit={handleSubmit} className="space-y-3">
              <p className="text-xs text-pat-text-secondary">
                Your free preview has ended. Create a free account to keep full access to the XAUUSD dashboard.
              </p>

              {error && <div className="text-pat-danger rounded-md px-3 py-2 text-xs" style={{ background: "hsl(var(--pat-badge-danger-bg) / 0.1)", border: "1px solid hsl(var(--pat-badge-danger-bg) / 0.2)" }}>{error}</div>}

              <div>
                <label className="block text-pat-text-primary mb-1 text-xs font-medium">Full name <span className="text-pat-danger">*</span></label>
                <input type="text" required value={fullName} onChange={(e) => setFullName(e.target.value)}
                  className="w-full rounded-md border border-pat-input-border bg-pat-input-bg text-pat-input-text px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-pat-primary"
                  placeholder="Your full name" autoComplete="name" />
              </div>

              <div>
                <label className="block text-pat-text-primary mb-1 text-xs font-medium">Email <span className="text-pat-danger">*</span></label>
                <input type="email" required value={email} onChange={(e) => setEmail(e.target.value)}
                  className="w-full rounded-md border border-pat-input-border bg-pat-input-bg text-pat-input-text px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-pat-primary"
                  placeholder="you@example.com" autoComplete="email" />
              </div>

              <div>
                <label className="block text-pat-text-primary mb-1 text-xs font-medium">Phone / WhatsApp (optional — for trade alerts)</label>
                <input type="tel" value={phone} onChange={(e) => setPhone(e.target.value)}
                  className="w-full rounded-md border border-pat-input-border bg-pat-input-bg text-pat-input-text px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-pat-primary"
                  placeholder="+971 50 000 0000" autoComplete="tel" />
              </div>

              <div>
                <label className="block text-pat-text-primary mb-1 text-xs font-medium">Trading broker (optional)</label>
                <select value={broker} onChange={(e) => setBroker(e.target.value)}
                  className="w-full rounded-md border border-pat-input-border bg-pat-input-bg text-pat-input-text px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-pat-primary">
                  <option value="">Select broker…</option>
                  {BROKERS.map((b) => <option key={b} value={b}>{b}</option>)}
                </select>
                {broker === "Other/None" && (
                  <input type="text" value={brokerOther} onChange={(e) => setBrokerOther(e.target.value)}
                    className="w-full rounded-md border border-pat-input-border bg-pat-input-bg text-pat-input-text px-3 py-2 text-sm mt-2 focus:outline-none focus:ring-2 focus:ring-pat-primary"
                    placeholder="Specify your broker" />
                )}
              </div>

              {/* ─── Three separate, unchecked consent checkboxes (PDPL) ─── */}
              <div className="space-y-2 pt-1">
                <ConsentCheckbox checked={terms} onChange={setTerms} required
                  label={<>I accept the <Link href="/terms" className="text-pat-primary hover:underline" target="_blank">Terms &amp; Conditions</Link> and <Link href="/privacy" className="text-pat-primary hover:underline" target="_blank">Privacy Policy</Link>.</>} />

                <ConsentCheckbox checked={risk} onChange={setRisk} required
                  label={<>I understand this platform is for informational/educational purposes only and is not investment advice. Trading CFDs/FX/gold (XAUUSD) carries a high risk of loss.</>} />

                <ConsentCheckbox checked={marketing} onChange={setMarketing}
                  label={<>I agree to receive marketing communications and offers via email and WhatsApp. <span className="text-pat-text-muted">(optional)</span></>} />
              </div>

              <button type="submit" disabled={loading}
                className="w-full rounded-md bg-pat-primary text-pat-primary-foreground hover:bg-pat-primary-hover disabled:opacity-50 transition-colors flex items-center justify-center gap-2 font-semibold py-2 text-sm">
                {loading ? <><IconLoader2 size={16} className="animate-spin" /> Sending code…</> : "Send verification code"}
              </button>
            </form>
          )}

          {step === "otp" && (
            <form onSubmit={handleVerify} className="space-y-3">
              <p className="text-xs text-pat-text-secondary">
                Enter the 6-digit code we sent to <span className="font-medium text-pat-text-primary">{email.trim().toLowerCase()}</span>.
              </p>

              {info && <div className="text-pat-success rounded-md px-3 py-2 text-xs" style={{ background: "hsl(var(--pat-badge-success-bg) / 0.1)", border: "1px solid hsl(var(--pat-badge-success-bg) / 0.2)" }}>{info}</div>}
              {error && <div className="text-pat-danger rounded-md px-3 py-2 text-xs" style={{ background: "hsl(var(--pat-badge-danger-bg) / 0.1)", border: "1px solid hsl(var(--pat-badge-danger-bg) / 0.2)" }}>{error}</div>}

              <div>
                <label htmlFor="otp-code" className="block text-pat-text-primary mb-1 text-xs font-medium">Verification code</label>
                <input id="otp-code" type="text" inputMode="numeric" pattern="\d{6}" maxLength={6} required value={code}
                  onChange={(e) => setCode(e.target.value.replace(/\D/g, ""))}
                  placeholder="000000"
                  className="w-full rounded-md border border-pat-input-border bg-pat-input-bg text-pat-input-text px-3 py-2 text-center font-mono tracking-widest text-lg focus:outline-none focus:ring-2 focus:ring-pat-primary" />
              </div>

              <button type="submit" disabled={loading || code.length !== 6}
                className="w-full rounded-md bg-pat-primary text-pat-primary-foreground hover:bg-pat-primary-hover disabled:opacity-50 transition-colors flex items-center justify-center gap-2 font-semibold py-2 text-sm">
                {loading ? <><IconLoader2 size={16} className="animate-spin" /> Verifying…</> : "Verify & unlock"}
              </button>

              <div className="flex items-center justify-between text-xs">
                <button type="button" onClick={() => setStep("form")} className="text-pat-text-muted hover:text-pat-text-primary">← Back</button>
                <button type="button" onClick={handleResend} disabled={cooldown > 0 || loading}
                  className="text-pat-primary hover:underline disabled:opacity-50 disabled:no-underline">
                  {cooldown > 0 ? `Resend in ${cooldown}s` : "Resend code"}
                </button>
              </div>
            </form>
          )}

          <div className="flex items-start gap-1.5 pt-2 border-t border-pat-border">
            <IconShieldCheck size={14} className="text-pat-text-muted flex-shrink-0 mt-0.5" />
            <p className="text-[10px] text-pat-text-muted leading-relaxed">
              We respect your privacy under UAE PDPL. Your consent choices are logged with a timestamp and version. You can unsubscribe at any time.
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}

/** A single distinct consent checkbox (never combined, never pre-ticked). */
function ConsentCheckbox({
  checked, onChange, label, required,
}: {
  checked: boolean;
  onChange: (v: boolean) => void;
  label: React.ReactNode;
  required?: boolean;
}) {
  return (
    <label className="flex items-start gap-2 cursor-pointer">
      <input type="checkbox" checked={checked} onChange={(e) => onChange(e.target.checked)}
        className="mt-0.5 h-4 w-4 rounded border-pat-input-border text-pat-primary focus:ring-pat-primary flex-shrink-0" />
      <span className="text-xs text-pat-text-secondary leading-relaxed">
        {label}
        {required && <span className="text-pat-danger"> *</span>}
      </span>
    </label>
  );
}
