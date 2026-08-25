"use client";
import { useState, Suspense, useMemo } from "react";
import Link from "next/link";
import { customInstance } from "@/lib/axios-instance";
import { useRouter, useSearchParams } from "next/navigation";
import { getApiErrorMessage } from "@/lib/errors";
import { IconLoader2, IconEye, IconEyeOff, IconCheck, IconShieldCheck } from "@tabler/icons-react";

function RegisterForm() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [confirmPassword, setConfirmPassword] = useState("");
  const [fullName, setFullName] = useState("");
  const [referralCode, setReferralCode] = useState(searchParams.get("ref") || "");
  const [showPassword, setShowPassword] = useState(false);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  // Consent checkboxes
  const [agreeToTerms, setAgreeToTerms] = useState(false);
  const [acknowledgePrivacy, setAcknowledgePrivacy] = useState(false);
  const [acknowledgeDataProcessing, setAcknowledgeDataProcessing] = useState(false);
  const [optInEmail, setOptInEmail] = useState(false);
  const [optInSms, setOptInSms] = useState(false);
  const [optInPhone, setOptInPhone] = useState(false);

  // Password strength calculation (memoized — no setState in effect)
  const passwordStrength = useMemo(() => {
    let score = 0;
    if (password.length >= 8) score++;
    if (password.length >= 12) score++;
    if (/[A-Z]/.test(password)) score++;
    if (/[0-9]/.test(password)) score++;
    if (/[^A-Za-z0-9]/.test(password)) score++;
    return score;
  }, [password]);

  const requiredConsents = agreeToTerms && acknowledgePrivacy && acknowledgeDataProcessing;
  const passwordsMatch = password === confirmPassword;
  const canSubmit = email && password.length >= 8 && passwordsMatch && requiredConsents && !loading;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError("");
    if (!passwordsMatch) { setError("Passwords do not match"); return; }
    if (password.length < 8) { setError("Password must be at least 8 characters"); return; }
    if (!requiredConsents) { setError("You must accept the required terms to continue"); return; }
    setLoading(true);
    try {
      await customInstance.post("/auth/register", {
        email,
        password,
        displayName: fullName || email.split("@")[0],
        ...(referralCode.trim() ? { referralCode: referralCode.trim() } : {}),
        agreeToTerms,
        acknowledgePrivacyPolicy: acknowledgePrivacy,
        acknowledgeDataProcessing: acknowledgeDataProcessing,
        optInEmailMarketing: optInEmail,
        optInSmsMarketing: optInSms,
        optInPhoneMarketing: optInPhone,
      });
      const redirect = new URLSearchParams(window.location.search).get("redirect");
      window.location.href = redirect === "live" ? "/login?registered=1&redirect=live" : "/login?registered=1";
    } catch (err: unknown) {
      setError(getApiErrorMessage(err, "Registration failed"));
    } finally {
      setLoading(false);
    }
  };

  const strengthLabels = ["Very Weak", "Weak", "Fair", "Good", "Strong", "Very Strong"];
  const strengthColors = ["#ef4444", "#f97316", "#eab308", "#84cc16", "#22c55e", "#16a34a"];

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: "16px" }}>
      {/* Heading */}
      <div>
        <h1 style={{ fontSize: "28px", fontWeight: 700, color: "#171a22", lineHeight: "1.1", margin: "0 0 8px 0" }}>
          Create your account
        </h1>
        <p style={{ fontSize: "14px", color: "#6c707a", margin: "0", lineHeight: "1.5" }}>
          Join Predict-A-Trade — XAUUSD real-time signal intelligence.
        </p>
      </div>

      {/* Auth card */}
      <div style={{
        border: "1px solid #d7d3c9",
        borderRadius: "10px",
        padding: "24px",
        background: "#ffffff",
        boxShadow: "0 4px 24px rgba(0,0,0,0.06)",
      }}>
        <form onSubmit={handleSubmit} style={{ display: "flex", flexDirection: "column", gap: "14px" }}>
          {error && (
            <div style={{ fontSize: "13px", color: "#b42318", background: "#fbe3e5", border: "1px solid #f3c3c7", borderRadius: "8px", padding: "10px 12px", lineHeight: 1.4 }}>
              {error}
            </div>
          )}

          {/* Full Name */}
          <div>
            <label style={{ display: "block", fontSize: "12px", fontWeight: 600, color: "#343842", marginBottom: "6px" }}>Full name</label>
            <input type="text" autoComplete="name" value={fullName} onChange={(e) => setFullName(e.target.value)}
              style={inputStyle}
              onFocus={(e) => e.target.style.borderColor = "#205fdc"}
              onBlur={(e) => e.target.style.borderColor = "#d7d3c9"}
              placeholder="John Doe" />
          </div>

          {/* Email */}
          <div>
            <label style={{ display: "block", fontSize: "12px", fontWeight: 600, color: "#343842", marginBottom: "6px" }}>Email address</label>
            <input type="email" required autoComplete="email" value={email} onChange={(e) => setEmail(e.target.value)}
              style={inputStyle}
              onFocus={(e) => e.target.style.borderColor = "#205fdc"}
              onBlur={(e) => e.target.style.borderColor = "#d7d3c9"}
              placeholder="you@example.com" />
          </div>

          {/* Password */}
          <div>
            <label style={{ display: "block", fontSize: "12px", fontWeight: 600, color: "#343842", marginBottom: "6px" }}>Create password</label>
            <div style={{ position: "relative" }}>
              <input type={showPassword ? "text" : "password"} required autoComplete="new-password" value={password} onChange={(e) => setPassword(e.target.value)}
                style={{ ...inputStyle, padding: "0 40px 0 14px" }}
                onFocus={(e) => e.target.style.borderColor = "#205fdc"}
                onBlur={(e) => e.target.style.borderColor = "#d7d3c9"}
                placeholder="Minimum 8 characters" />
              <button type="button" onClick={() => setShowPassword(!showPassword)}
                style={{ position: "absolute", right: "8px", top: "50%", transform: "translateY(-50%)", background: "none", border: "none", cursor: "pointer", padding: "4px", color: "#6c707a", display: "flex", alignItems: "center" }}>
                {showPassword ? <IconEyeOff size={18} /> : <IconEye size={18} />}
              </button>
            </div>
            {/* Password strength bar */}
            {password.length > 0 && (
              <div style={{ marginTop: "6px", display: "flex", alignItems: "center", gap: "6px" }}>
                <div style={{ flex: 1, height: "4px", background: "#e8e5dd", borderRadius: "2px", overflow: "hidden" }}>
                  <div style={{ width: `${(passwordStrength / 5) * 100}%`, height: "100%", background: strengthColors[passwordStrength], borderRadius: "2px", transition: "width 0.3s, background 0.3s" }} />
                </div>
                <span style={{ fontSize: "10px", color: strengthColors[passwordStrength], fontWeight: 600, minWidth: "70px" }}>{strengthLabels[passwordStrength]}</span>
              </div>
            )}
          </div>

          {/* Confirm Password */}
          <div>
            <label style={{ display: "block", fontSize: "12px", fontWeight: 600, color: "#343842", marginBottom: "6px" }}>Confirm password</label>
            <input type={showPassword ? "text" : "password"} required autoComplete="new-password" value={confirmPassword} onChange={(e) => setConfirmPassword(e.target.value)}
              style={{ ...inputStyle, borderColor: confirmPassword && !passwordsMatch ? "#ef4444" : undefined }}
              onFocus={(e) => e.target.style.borderColor = "#205fdc"}
              onBlur={(e) => e.target.style.borderColor = confirmPassword && !passwordsMatch ? "#ef4444" : "#d7d3c9"}
              placeholder="Re-enter password" />
            {confirmPassword && !passwordsMatch && (
              <span style={{ fontSize: "11px", color: "#ef4444", marginTop: "4px", display: "block" }}>Passwords do not match</span>
            )}
          </div>

          {/* Referral Code */}
          <div>
            <label style={{ display: "block", fontSize: "12px", fontWeight: 600, color: "#343842", marginBottom: "6px" }}>
              Referral code <span style={{ fontWeight: 400, color: "#9ba3b4" }}>(optional)</span>
            </label>
            <input type="text" autoComplete="off" value={referralCode} onChange={(e) => setReferralCode(e.target.value)}
              style={inputStyle}
              onFocus={(e) => e.target.style.borderColor = "#205fdc"}
              onBlur={(e) => e.target.style.borderColor = "#d7d3c9"}
              placeholder="PAT-XXXX..." />
          </div>

          {/* ── Divider ── */}
          <div style={{ height: "1px", background: "#e8e5dd", margin: "4px 0" }} />

          {/* ── Required Consent Checkboxes ── */}
          <div style={{ display: "flex", flexDirection: "column", gap: "10px" }}>
            <p style={{ fontSize: "12px", fontWeight: 700, color: "#343842", margin: "0", textTransform: "uppercase", letterSpacing: "0.5px" }}>
              Required Agreements
            </p>

            <ConsentCheckbox
              checked={agreeToTerms}
              onChange={setAgreeToTerms}
              required
              label={<>I agree to the <Link href="/terms" target="_blank" style={{ color: "#205fdc", textDecoration: "none", fontWeight: 600 }}>Terms of Use</Link> and <Link href="/privacy" target="_blank" style={{ color: "#205fdc", textDecoration: "none", fontWeight: 600 }}>Privacy Policy</Link></>}
            />

            <ConsentCheckbox
              checked={acknowledgePrivacy}
              onChange={setAcknowledgePrivacy}
              required
              label={<>I confirm that I have read and acknowledge the Simha FinTech <Link href="/terms" target="_blank" style={{ color: "#205fdc", textDecoration: "none", fontWeight: 600 }}>Terms of Service</Link></>}
            />

            <ConsentCheckbox
              checked={acknowledgeDataProcessing}
              onChange={setAcknowledgeDataProcessing}
              required
              label={<>I confirm that I have read and acknowledge the <Link href="/data-processing-agreement" target="_blank" style={{ color: "#205fdc", textDecoration: "none", fontWeight: 600 }}>Privacy Policy and Data Processing and Security Agreement</Link></>}
            />
          </div>

          {/* ── Marketing Preferences (optional) ── */}
          <div style={{ display: "flex", flexDirection: "column", gap: "8px", marginTop: "4px", padding: "12px", background: "#f7f6f2", borderRadius: "8px" }}>
            <p style={{ fontSize: "12px", fontWeight: 700, color: "#343842", margin: "0 0 2px 0", textTransform: "uppercase", letterSpacing: "0.5px" }}>
              Marketing Preferences <span style={{ fontWeight: 400, color: "#9ba3b4", textTransform: "none", letterSpacing: "0" }}>(optional)</span>
            </p>

            <ConsentCheckbox checked={optInEmail} onChange={setOptInEmail} label="I want to receive news and promotional offers by email." />
            <ConsentCheckbox checked={optInSms} onChange={setOptInSms} label="I want to receive news and promotional offers by SMS." />
            <ConsentCheckbox checked={optInPhone} onChange={setOptInPhone} label="I want to receive news and promotional offers by phone call." />

            <p style={{ fontSize: "10px", color: "#9ba3b4", margin: "2px 0 0 0", lineHeight: 1.4 }}>
              You can opt out of marketing communications at any time. Registration does not require marketing opt-in.
            </p>
          </div>

          {/* Submit button */}
          <button type="submit" disabled={!canSubmit}
            style={{
              width: "100%",
              height: "44px",
              background: !canSubmit ? "#c5c9d0" : loading ? "#1749ae" : "#205fdc",
              color: "#ffffff",
              border: "none",
              borderRadius: "8px",
              fontSize: "14px",
              fontWeight: 600,
              cursor: !canSubmit ? "not-allowed" : loading ? "wait" : "pointer",
              display: "flex",
              alignItems: "center",
              justifyContent: "center",
              gap: "8px",
              opacity: loading ? 0.8 : 1,
              transition: "background 0.2s, opacity 0.2s",
              marginTop: "4px",
            }}
            onMouseEnter={(e) => { if (canSubmit && !loading) e.currentTarget.style.background = "#1749ae"; }}
            onMouseLeave={(e) => { if (canSubmit && !loading) e.currentTarget.style.background = "#205fdc"; }}>
            {loading ? (<><IconLoader2 size={16} className="animate-spin" /> Creating account...</>) : (<><IconShieldCheck size={18} /> Create my account</>)}
          </button>
        </form>
      </div>

      {/* Switch to login */}
      <p style={{ textAlign: "center", fontSize: "13px", color: "#6c707a", margin: "0" }}>
        Already have an account? <Link href="/login" style={{ color: "#205fdc", fontWeight: 600, textDecoration: "none" }}>Sign in</Link>
      </p>
    </div>
  );
}

// ── Reusable consent checkbox component ──
function ConsentCheckbox({
  checked, onChange, label, required,
}: {
  checked: boolean;
  onChange: (v: boolean) => void;
  label: React.ReactNode;
  required?: boolean;
}) {
  return (
    <label style={{ display: "flex", alignItems: "flex-start", gap: "8px", cursor: "pointer", fontSize: "13px", color: "#343842", lineHeight: 1.45 }}>
      <div
        onClick={() => onChange(!checked)}
        style={{
          width: "18px",
          height: "18px",
          minWidth: "18px",
          borderRadius: "4px",
          border: checked ? "2px solid #205fdc" : "2px solid #c5c9d0",
          background: checked ? "#205fdc" : "#ffffff",
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          marginTop: "1px",
          transition: "all 0.15s",
        }}
      >
        {checked && <IconCheck size={13} color="#ffffff" strokeWidth={3} />}
      </div>
      <span>
        {label}
        {required && <span style={{ color: "#ef4444", marginLeft: "2px" }}>*</span>}
      </span>
    </label>
  );
}

const inputStyle: React.CSSProperties = {
  width: "100%",
  height: "44px",
  border: "1px solid #d7d3c9",
  borderRadius: "8px",
  background: "#ffffff",
  color: "#171a22",
  padding: "0 14px",
  fontSize: "14px",
  outline: "none",
  boxSizing: "border-box",
  transition: "border-color 0.15s",
};

export default function RegisterPage() {
  return (
    <Suspense fallback={<div style={{ display: "flex", alignItems: "center", justifyContent: "center", minHeight: "200px" }}><IconLoader2 size={24} className="animate-spin" style={{ color: "#6c707a" }} /></div>}>
      <RegisterForm />
    </Suspense>
  );
}
