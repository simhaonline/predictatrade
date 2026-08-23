"use client";
import { IconNews, IconAlertTriangle } from "@tabler/icons-react";

function DegradedBanner({ children }: { children: React.ReactNode }) {
  return (
    <div className="bg-pat-warning/10 border border-pat-warning/20 rounded-lg p-3 flex items-start gap-2">
      <IconAlertTriangle size={16} className="text-pat-warning mt-0.5 shrink-0" />
      <div className="text-xs text-pat-warning">{children}</div>
    </div>
  );
}

function PendingPanel({ title, hint }: { title: string; hint: string }) {
  return (
    <div className="bg-pat-bg-surface border border-pat-border rounded-lg p-4 opacity-80">
      <div className="text-sm font-medium text-pat-text-primary mb-2">{title}</div>
      <div className="text-xs text-pat-text-muted">{hint}</div>
      <div className="text-xs text-pat-warning mt-2">Data source pending — placeholder only</div>
    </div>
  );
}

export default function AdminMacroNewsPage() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-bold text-pat-text-primary">Macro &amp; News</h1>
        <p className="text-sm text-pat-text-secondary mt-1">
          DXY, yields, economic calendar, news, blackouts and provider health. No backend data source is wired for this page.
        </p>
      </div>

      <DegradedBanner>
        Backend data source pending — showing schema/placeholder only. No macro, news, calendar, or blackout
        data is rendered. Do not interpret these panels as live market information.
      </DegradedBanner>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
        <PendingPanel title="DXY" hint="US Dollar Index live value and change." />
        <PendingPanel title="Yields" hint="US 10Y / 2Y treasury yields." />
        <PendingPanel title="Economic Calendar" hint="Upcoming high-impact events (FOMC, NFP, CPI)." />
        <PendingPanel title="News Feed" hint="Aggregated macro headlines with sentiment." />
        <PendingPanel title="Blackout Windows" hint="Trading-restriction windows around events." />
        <PendingPanel title="Provider Health" hint="Macro/news provider connectivity status." />
      </div>

      {/* Blackout config form — degraded */}
      <div className="bg-pat-card-bg border border-pat-card-border rounded-lg p-4 shadow-sm opacity-80">
        <h2 className="text-sm font-medium text-pat-text-primary mb-3">Blackout Configuration (Pending Backend)</h2>
        <form className="grid grid-cols-1 md:grid-cols-4 gap-3" onSubmit={(e) => e.preventDefault()}>
          <input
            disabled
            placeholder="Event label"
            className="rounded-md border border-pat-input-border bg-pat-input-bg px-3 py-2 text-sm text-pat-input-text opacity-60"
          />
          <input
            disabled
            type="datetime-local"
            className="rounded-md border border-pat-input-border bg-pat-input-bg px-3 py-2 text-sm text-pat-input-text opacity-60"
          />
          <input
            disabled
            type="datetime-local"
            className="rounded-md border border-pat-input-border bg-pat-input-bg px-3 py-2 text-sm text-pat-input-text opacity-60"
          />
          <button
            disabled
            className="px-3 py-2 text-sm rounded-md bg-pat-primary text-pat-primary-foreground opacity-50 cursor-not-allowed"
          >
            Save Blackout (disabled)
          </button>
        </form>
      </div>
    </div>
  );
}
