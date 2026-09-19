"use client";
// useMounted — client-side mount detection without setState-in-effect.
// React 19 strict flags `useEffect(() => setMounted(true), [])` as a
// cascading render; useSyncExternalStore provides the sanctioned equivalent:
// server snapshot false, client snapshot true, no state writes.
import { useSyncExternalStore } from "react";

const emptySubscribe = () => () => {};
const mountedStore = {
  subscribe: emptySubscribe,
  getServerSnapshot: () => false,
  getSnapshot: () => true,
};

export function useMounted(): boolean {
  return useSyncExternalStore(mountedStore.subscribe, mountedStore.getSnapshot, mountedStore.getServerSnapshot);
}

// useNow — 1s-ticking timestamp as an external store. Reading Date.now()
// during render is flagged impure by React 19; this moves the moving clock
// into an external store (sanctioned pattern) with listener ref-counting.
const clockListeners = new Set<() => void>();
let clockNow = 0;
let clockTimer: ReturnType<typeof setInterval> | null = null;
function startClock() {
  clockNow = Date.now();
  clockTimer = setInterval(() => {
    clockNow = Date.now();
    for (const l of clockListeners) l();
  }, 1000);
}
function subscribeClock(l: () => void) {
  clockListeners.add(l);
  if (clockTimer === null) startClock();
  return () => {
    clockListeners.delete(l);
    if (clockListeners.size === 0 && clockTimer !== null) {
      clearInterval(clockTimer);
      clockTimer = null;
    }
  };
}
export function useNow(): number {
  return useSyncExternalStore(
    subscribeClock,
    () => clockNow,
    () => 0,
  );
}
