"use client";
import { useQuery } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import { IconNews } from "@tabler/icons-react";
import { DegradedBanner } from "@/components/ui/degraded-banner";

interface MacroNewsRow {
  id: number;
  event_id: string;
  provider: string;
  event_name: string;
  country: string;
  currency: string;
  impact: string;
  scheduled_at_utc: string | null;
  actual: string | null;
  forecast: string | null;
  previous: string | null;
  received_at: string | null;
}

interface MacroNewsData {
  items: MacroNewsRow[];
  note?: string;
}

function fmt(ts: string | null) {
  return ts ? new Date(ts).toISOString().slice(0, 19).replace("T", " ") : "—";
}

export default function AdminMacroNewsPage() {
  const { data, isLoading, isError, error } = useQuery<MacroNewsData>({
    queryKey: ["admin-macro-news"],
    queryFn: async () => (await customInstance.get("/news")).data as MacroNewsData,
  });

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-bold text-pat-text-primary">Macro Calendar</h1>
        <p className="text-sm text-pat-text-secondary mt-1">
          Economic calendar and macro/news events. Honest read-only view from the event registry.
        </p>
      </div>

      {isLoading && <DegradedBanner>Loading macro/news data from backend…</DegradedBanner>}
      {isError && (
        <DegradedBanner>
          Macro/news backend degraded: {error instanceof Error ? error.message : "unable to reach endpoint"}.
          No calendar, news, or blackout data is rendered.
        </DegradedBanner>
      )}

      {data && (
        <div className="bg-pat-card-bg border border-pat-card-border rounded-lg p-4 shadow-sm">
          <h2 className="text-sm font-medium text-pat-text-primary mb-3 flex items-center gap-2">
            <IconNews size={16} /> Economic Calendar
          </h2>
          {data.note && !data.items.length ? (
            <div className="px-3 py-6 text-center text-xs text-pat-text-muted">{data.note}</div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="text-left text-xs text-pat-text-muted border-b border-pat-border">
                    <th className="px-3 py-2 font-medium">Event</th>
                    <th className="px-3 py-2 font-medium">Country</th>
                    <th className="px-3 py-2 font-medium">Currency</th>
                    <th className="px-3 py-2 font-medium">Impact</th>
                    <th className="px-3 py-2 font-medium">Scheduled (UTC)</th>
                    <th className="px-3 py-2 font-medium">Actual</th>
                    <th className="px-3 py-2 font-medium">Forecast</th>
                    <th className="px-3 py-2 font-medium">Previous</th>
                  </tr>
                </thead>
                <tbody>
                  {data.items.map((r) => (
                    <tr key={r.id} className="border-b border-pat-border/60">
                      <td className="px-3 py-2 text-pat-text-primary">{r.event_name}</td>
                      <td className="px-3 py-2 text-pat-text-secondary">{r.country || "—"}</td>
                      <td className="px-3 py-2">{r.currency || "—"}</td>
                      <td className="px-3 py-2">{r.impact || "—"}</td>
                      <td className="px-3 py-2 text-pat-text-muted">{fmt(r.scheduled_at_utc)}</td>
                      <td className="px-3 py-2">{r.actual || "—"}</td>
                      <td className="px-3 py-2">{r.forecast || "—"}</td>
                      <td className="px-3 py-2">{r.previous || "—"}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </div>
      )}

      <div className="bg-pat-card-bg border border-pat-card-border rounded-lg p-4 shadow-sm opacity-80">
        <h2 className="text-sm font-medium text-pat-text-primary mb-3">Blackout Configuration</h2>
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
