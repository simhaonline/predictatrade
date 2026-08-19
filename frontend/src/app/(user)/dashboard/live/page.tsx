"use client";
import LiveDashboard from "@/components/trading/live-dashboard";

export default function UserLiveDashboardPage() {
  return (
    <div className="space-y-4">
      <h1 className="text-xl font-bold">Live Dashboard</h1>
      <LiveDashboard isAdmin={false} />
    </div>
  );
}
