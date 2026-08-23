/**
 * Client Telemetry Collector — Predict-A-Trade
 * Collects browser/device information for compliance audit logging.
 * All values are untrusted client telemetry — server validates and sanitizes.
 * Telemetry failure NEVER blocks prediction submission.
 */

export interface ClientTelemetryPayload {
  user_agent: string;
  language: string;
  languages: string[];
  timezone: string;
  timezone_offset_minutes: number;
  screen: {
    width: number;
    height: number;
    avail_width: number;
    avail_height: number;
  };
  viewport: {
    width: number;
    height: number;
  };
  device_pixel_ratio: number;
  color_depth: number;
  touch_points: number;
  platform: string;
  client_hints?: Record<string, string | boolean>;
}

/**
 * Collect browser telemetry safely. Never throws.
 * Returns null if collection fails (graceful degradation).
 */
export function collectClientTelemetry(): ClientTelemetryPayload | null {
  try {
    if (typeof window === 'undefined') return null;

    const nav = navigator as Navigator & {
      userAgentData?: { platform?: string; mobile?: boolean; getHighEntropyValues?: (hints: string[]) => Promise<Record<string, unknown>> };
      maxTouchPoints?: number;
    };

    // Collect Client Hints if available
    let clientHints: Record<string, string | boolean> | undefined;
    if (nav.userAgentData && typeof nav.userAgentData === 'object') {
      clientHints = {};
      const uaData = nav.userAgentData;
      if (uaData.platform) clientHints.platform = uaData.platform;
      if (uaData.mobile !== undefined) clientHints.mobile = uaData.mobile;
      // High entropy values (may require permission)
      if (typeof uaData.getHighEntropyValues === 'function') {
        try {
          // Don't await — collect synchronously available hints only
          // Full high entropy values would require async collection
        } catch {}
      }
    }

    return {
      user_agent: nav.userAgent || '',
      language: nav.language || '',
      languages: Array.isArray(nav.languages) ? nav.languages.slice(0, 10) : [],
      timezone: Intl.DateTimeFormat().resolvedOptions().timeZone || '',
      timezone_offset_minutes: -new Date().getTimezoneOffset(),
      screen: {
        width: screen.width || 0,
        height: screen.height || 0,
        avail_width: screen.availWidth || 0,
        avail_height: screen.availHeight || 0,
      },
      viewport: {
        width: window.innerWidth || 0,
        height: window.innerHeight || 0,
      },
      device_pixel_ratio: window.devicePixelRatio || 1,
      color_depth: screen.colorDepth || 24,
      touch_points: nav.maxTouchPoints || 0,
      platform: nav.platform || '',
      client_hints: clientHints,
    };
  } catch {
    // Telemetry collection must NEVER break the application
    return null;
  }
}

/**
 * Send telemetry to server. Non-blocking, fire-and-forget.
 * Never throws, never blocks prediction submission.
 */
export function sendTelemetry(): void {
  try {
    const telemetry = collectClientTelemetry();
    if (!telemetry) return;

    // Fire and forget — use sendBeacon or fetch with keepalive
    const body = JSON.stringify(telemetry);

    if (navigator.sendBeacon) {
      navigator.sendBeacon('/api/v1/telemetry/client', body);
    } else {
      fetch('/api/v1/telemetry/client', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body,
        keepalive: true,
      }).catch(() => {}); // Silently ignore failures
    }
  } catch {
    // Never throw
  }
}
