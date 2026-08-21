"use client";
import BacktestPanel from "@/components/backtest/backtest-panel";

export default function UserBacktestPage() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-xl font-bold text-pat-text-primary">Backtest</h1>
        <p className="text-sm text-pat-text-secondary mt-1">
          Test strategies on historical data before trading live. 
          Results use the same production engine that generates live signals.
        </p>
      </div>
      <BacktestPanel />
    </div>
  );
}
