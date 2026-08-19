"use client";
import { createContext, useContext, useState, useEffect, useCallback } from "react";
import Link from "next/link";
import { IconX } from "@tabler/icons-react";

interface CookieConsent {
  version: string;
  necessary: boolean;
  preferences: boolean;
  analytics: boolean;
  marketing: boolean;
  updatedAt: string;
}

interface CookieConsentContextValue {
  consent: CookieConsent | null;
  setConsent: (c: CookieConsent) => void;
  hasConsented: (category: keyof CookieConsent) => boolean;
}

const CookieContext = createContext<CookieConsentContextValue | undefined>(undefined);
const STORAGE_KEY = 'pat_cookie_consent';
const CURRENT_VERSION = '1';

function loadConsent(): CookieConsent | null {
  if (typeof window === 'undefined') return null;
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return null;
    const parsed = JSON.parse(raw) as CookieConsent;
    if (parsed.version !== CURRENT_VERSION) return null;
    return parsed;
  } catch {
    return null;
  }
}

function saveConsent(c: CookieConsent) {
  if (typeof window === 'undefined') return;
  localStorage.setItem(STORAGE_KEY, JSON.stringify(c));
}

export function CookieConsentProvider({ children }: { children: React.ReactNode }) {
  const [consent, setConsentState] = useState<CookieConsent | null>(null);
  const [showBanner, setShowBanner] = useState(false);
  const [showSettings, setShowSettings] = useState(false);

  useEffect(() => {
    const existing = loadConsent();
    queueMicrotask(() => {
      setConsentState(existing);
      if (!existing) {
        setShowBanner(true);
      }
    });
  }, []);

  // Listen for "open cookie settings" events from footer
  useEffect(() => {
    const handler = () => {
      const existing = loadConsent();
      if (existing) {
        setConsentState(existing);
      }
      setShowSettings(true);
    };
    window.addEventListener('pat:open-cookie-settings', handler);
    return () => window.removeEventListener('pat:open-cookie-settings', handler);
  }, []);

  const setConsent = useCallback((c: CookieConsent) => {
    saveConsent(c);
    setConsentState(c);
    setShowBanner(false);
    setShowSettings(false);
  }, []);

  const hasConsented = useCallback((category: keyof CookieConsent): boolean => {
    if (!consent) return false;
    if (category === 'version') return true;
    return consent[category] as boolean;
  }, [consent]);

  const rejectAll = () => {
    setConsent({
      version: CURRENT_VERSION,
      necessary: true,
      preferences: false,
      analytics: false,
      marketing: false,
      updatedAt: new Date().toISOString(),
    });
  };

  const allowAll = () => {
    setConsent({
      version: CURRENT_VERSION,
      necessary: true,
      preferences: true,
      analytics: true,
      marketing: true,
      updatedAt: new Date().toISOString(),
    });
  };

  return (
    <CookieContext.Provider value={{ consent, setConsent, hasConsented }}>
      {children}

      {/* Cookie consent banner */}
      {showBanner && !showSettings && (
        <div className="fixed bottom-0 left-0 right-0 z-[100] bg-pat-bg-surface border-t border-pat-border shadow-lg">
          <div className="max-w-7xl mx-auto px-4 py-4">
            <div className="flex flex-col sm:flex-row items-start sm:items-center justify-between gap-4">
              <p className="text-sm text-pat-text-secondary flex-1">
                We use cookies to enhance site navigation, personalise content and ads, and analyse site usage. You can change your cookie settings at any time. For more information, please see our{" "}
                <Link href="/cookies" className="text-pat-primary hover:underline">Cookie Policy</Link>.
              </p>
              <div className="flex items-center gap-2 flex-shrink-0">
                <button
                  onClick={rejectAll}
                  className="px-3 py-1.5 text-xs font-medium border border-pat-border-strong rounded-md text-pat-text-secondary hover:bg-pat-bg-surface-secondary transition-colors"
                >
                  Reject All
                </button>
                <button
                  onClick={() => setShowSettings(true)}
                  className="px-3 py-1.5 text-xs font-medium border border-pat-border-strong rounded-md text-pat-text-secondary hover:bg-pat-bg-surface-secondary transition-colors"
                >
                  Cookie Settings
                </button>
                <button
                  onClick={allowAll}
                  className="px-3 py-1.5 text-xs font-medium bg-pat-primary text-pat-primary-foreground rounded-md hover:bg-pat-primary-hover transition-colors"
                >
                  Allow All Cookies
                </button>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Cookie settings modal */}
      {showSettings && (
        <CookieSettingsModal
          currentConsent={consent}
          onSave={setConsent}
          onClose={() => { setShowSettings(false); setShowBanner(false); }}
        />
      )}
    </CookieContext.Provider>
  );
}

function CookieSettingsModal({
  currentConsent,
  onSave,
  onClose,
}: {
  currentConsent: CookieConsent | null;
  onSave: (c: CookieConsent) => void;
  onClose: () => void;
}) {
  const [prefs, setPrefs] = useState({
    necessary: true,
    preferences: currentConsent?.preferences ?? true,
    analytics: currentConsent?.analytics ?? false,
    marketing: currentConsent?.marketing ?? false,
  });

  const categories = [
    { key: 'necessary' as const, label: 'Strictly Necessary', desc: 'Required for authentication and core platform function. Cannot be disabled.', disabled: true },
    { key: 'preferences' as const, label: 'Preferences', desc: 'Theme preference, display settings, and accessibility options.', disabled: false },
    { key: 'analytics' as const, label: 'Analytics', desc: 'Anonymous usage statistics to improve platform quality.', disabled: false },
    { key: 'marketing' as const, label: 'Marketing', desc: 'Advertising and promotional content personalization.', disabled: false },
  ];

  return (
    <div className="fixed inset-0 z-[110] flex items-center justify-center bg-black/50" onClick={onClose}>
      <div
        className="bg-pat-bg-surface border border-pat-border rounded-lg shadow-xl max-w-lg w-full mx-4 max-h-[80vh] overflow-y-auto"
        onClick={(e) => e.stopPropagation()}
        role="dialog"
        aria-label="Cookie Settings"
      >
        <div className="flex items-center justify-between p-4 border-b border-pat-border">
          <h2 className="text-base font-semibold text-pat-text-primary">Cookie Settings</h2>
          <button onClick={onClose} className="p-1 hover:bg-pat-bg-surface-secondary rounded text-pat-text-muted" aria-label="Close">
            <IconX size={18} />
          </button>
        </div>
        <div className="p-4 space-y-4">
          {categories.map((cat) => (
            <div key={cat.key} className="flex items-start justify-between gap-4 py-2 border-b border-pat-border last:border-0">
              <div className="flex-1">
                <div className="text-sm font-medium text-pat-text-primary">{cat.label}</div>
                <div className="text-xs text-pat-text-muted mt-1">{cat.desc}</div>
              </div>
              <label className="relative inline-flex items-center cursor-pointer flex-shrink-0 mt-1">
                <input
                  type="checkbox"
                  checked={prefs[cat.key]}
                  disabled={cat.disabled}
                  onChange={(e) => !cat.disabled && setPrefs(p => ({ ...p, [cat.key]: e.target.checked }))}
                  className="sr-only peer"
                />
                <div className="w-10 h-5 bg-pat-bg-surface-secondary peer-checked:bg-pat-primary rounded-full peer-disabled:opacity-50 transition-colors relative">
                  <div className={`absolute top-0.5 left-0.5 w-4 h-4 bg-white rounded-full transition-transform ${prefs[cat.key] ? 'translate-x-5' : ''}`} />
                </div>
              </label>
            </div>
          ))}
        </div>
        <div className="flex items-center justify-end gap-2 p-4 border-t border-pat-border">
          <button onClick={onClose} className="px-3 py-1.5 text-xs font-medium border border-pat-border-strong rounded-md text-pat-text-secondary hover:bg-pat-bg-surface-secondary transition-colors">
            Cancel
          </button>
          <button
            onClick={() => onSave({ version: CURRENT_VERSION, ...prefs, updatedAt: new Date().toISOString() })}
            className="px-3 py-1.5 text-xs font-medium bg-pat-primary text-pat-primary-foreground rounded-md hover:bg-pat-primary-hover transition-colors"
          >
            Save Preferences
          </button>
        </div>
      </div>
    </div>
  );
}

export function useCookieConsent() {
  const ctx = useContext(CookieContext);
  if (!ctx) throw new Error('useCookieConsent must be inside CookieConsentProvider');
  return ctx;
}
