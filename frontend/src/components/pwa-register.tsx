"use client";
import { useEffect } from "react";

export default function PwaRegister() {
  useEffect(() => {
    if (typeof window === "undefined") return;
    if (!("serviceWorker" in navigator)) return;
    if (window.location.protocol !== "https:" && window.location.hostname !== "localhost") return;

    const register = () => {
      navigator.serviceWorker.register("/sw.js", { scope: "/" }).then(async (reg) => {
        // Build-mismatch auto-recover (no user action): poll /api/build-id; if
        // the SW sees a different build than the server, it purges all caches
        // and this page reloads ONCE. Fixes stale-UI across deploys.
        try {
          const res = await fetch('/api/build-id', { cache: 'no-store' });
          const { buildId } = await res.json();
          if (reg.active) reg.active.postMessage({ type: 'BUILD_ID', buildId });
          navigator.serviceWorker.addEventListener('message', (ev) => {
            if (ev.data?.type === 'PURGED' && !sessionStorage.getItem('pat-reloaded')) {
              sessionStorage.setItem('pat-reloaded', '1');
              window.location.reload();
            }
          });
        } catch { /* best-effort */ }
      }).catch(() => {
        // Registration is best-effort; the app works fully without it.
      });
    };

    if (document.readyState === "complete") register();
    else window.addEventListener("load", register, { once: true });
  }, []);

  return null;
}
