"use client";
import { Toaster } from "sonner";
import { useTheme } from "next-themes";
import { useSyncExternalStore } from "react";

// useSyncExternalStore avoids the setState-in-effect lint error
// and prevents hydration mismatch by returning false on server, true on client
const emptySubscribe = () => () => {};

function useMounted() {
  return useSyncExternalStore(
    emptySubscribe,
    () => true, // client snapshot — always true after mount
    () => false, // server snapshot — always false during SSR
  );
}

export function ThemedToaster() {
  const { resolvedTheme } = useTheme();
  const mounted = useMounted();

  // Don't render until mounted on client — prevents hydration mismatch
  // because useTheme() returns undefined on server but "dark" on client
  if (!mounted) return null;

  return (
    <Toaster
      position="top-right"
      richColors
      theme={resolvedTheme as "light" | "dark" | undefined}
    />
  );
}
