"use client";
import BacktestPanel from "@/components/backtest/backtest-panel";
import PlanPerformanceCards from "@/components/backtest/plan-performance-cards";

export default function AdminBacktestingPage() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-bold text-pat-text-primary">Backtesting Reports</h1>
        <p className="text-sm text-pat-text-secondary mt-1">
          Run backtests with real historical data. Select strategy, timeframe, and date range to generate reports.
        </p>
      </div>
      <PlanPerformanceCards />
      <BacktestPanel isAdmin />
    </div>
  );
}