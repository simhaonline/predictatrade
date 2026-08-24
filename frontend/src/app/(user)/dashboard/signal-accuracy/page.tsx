"use client";
import { useQuery } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import { IconChartBar } from "@tabler/icons-react";

interface RankRow {
  strategyId: string;
  total: number;
  resolved: number;
  wins: number;
  losses: number;
  winRate: number | null;
  avgPnl: number;
  totalPnl: number;
}

export default function UserSignalAccuracyPage() {
  const { data, isLoading, error } = useQuery<{ generatedAt: string; strategies: RankRow[] }>({
    queryKey: ["user-signal-accuracy"],
    queryFn: async () => (await customInstance.get("/signal-accuracy")).data,
    refetchInterval: 30000,
  });

  const rows = data?.strategies ?? [];

  return (
    <div className="space-y-4">
      <div>
        <h1 className="text-xl font-bold text-pat-text-primary">Signal Accuracy Ranking</h1>
        <p className="text-sm text-pat-text-secondary mt-1">
          Real win rates per strategy — derived from resolved signals with realized P&amp;L. Transparency builds confidence in the engine.
        </p>
      </div>

      {isLoading ? (
        <div className="h-32 bg-pat-bg-surface-secondary/50 rounded animate-pulse" />
      ) : error ? (
        <div className="text-sm text-pat-danger">Failed to load ranking.</div>
      ) : rows.length === 0 ? (
        <div className="text-sm text-pat-text-muted border border-pat-border rounded-lg p-6">
          No resolved signals yet. Accuracy ranking populates as signals close with realized P&amp;L.
        </div>
      ) : (
        <div className="overflow-x-auto border border-pat-border rounded-lg">
          <table className="w-full text-sm">
            <thead className="bg-pat-bg-surface text-pat-text-secondary uppercase text-xs">
              <tr>
                <th className="px-3 py-3 text-left">Rank</th>
                <th className="px-3 py-3 text-left">Strategy</th>
                <th className="px-3 py-3 text-right">Resolved</th>
                <th className="px-3 py-3 text-right">Wins</th>
                <th className="px-3 py-3 text-right">Losses</th>
                <th className="px-3 py-3 text-right">Win Rate</th>
                <th className="px-3 py-3 text-right">Avg P&amp;L</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-neutral-800">
              {rows.map((r, i) => (
                <tr key={r.strategyId} className="hover:bg-pat-table-hover">
                  <td className="px-3 py-3 text-pat-text-muted">{i + 1}</td>
                  <td className="px-3 py-3 text-pat-text-primary font-medium">{r.strategyId.replace(/_/g, " ")}</td>
                  <td className="px-3 py-3 text-right tabular-nums">{r.resolved}</td>
                  <td className="px-3 py-3 text-right tabular-nums text-pat-success">{r.wins}</td>
                  <td className="px-3 py-3 text-right tabular-nums text-pat-danger">{r.losses}</td>
                  <td className="px-3 py-3 text-right tabular-nums font-semibold text-pat-text-primary">
                    {r.winRate !== null ? `${r.winRate.toFixed(1)}%` : "—"}
                  </td>
                  <td className="px-3 py-3 text-right tabular-nums">{r.avgPnl.toFixed(2)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
