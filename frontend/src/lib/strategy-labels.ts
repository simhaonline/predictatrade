// Strategy display labels — check.md 2026-08-30: MARNIE_FIB rebranded.
// Internal ID stays MARNIE_FIB (DB, signals, telemetry): display name is EQFE.
export const STRATEGY_DISPLAY_NAMES: Record<string, string> = {
  STANDARD_SCALPING: "Standard Scalping",
  ULTRA_SCALPING: "Ultra Scalping",
  STANDARD_SWING: "Standard Swing",
  TREND_SWING: "Trend Swing",
  MARNIE_FIB: "EQFE",
  ATEN: "ATEN",  // Aetherial Technical Engine Node
};

export function strategyLabel(id: string | undefined | null): string {
  if (!id) return "—";
  return STRATEGY_DISPLAY_NAMES[id] ?? id.replace(/_/g, " ");
}

// Human-readable display name map for UI (id → label)
export function displayNameFor(id: string): string { return strategyLabel(id); }

// Replace every raw strategy ID the user can see with its display name.
// Use this when the UI receives an ID from the backend (signals, tabs, tables).
export function pretty(idOrLabel: string): string {
  // if it's already one of the nice labels just return it
  for (const label of Object.values(STRATEGY_DISPLAY_NAMES)) {
    if (idOrLabel === label || idOrLabel.replace(/_/g, " ").toLowerCase() === label.toLowerCase()) return idOrLabel;
  }
  return strategyLabel(idOrLabel);
}
