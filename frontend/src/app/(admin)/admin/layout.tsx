"use client";
import AppShell from "@/components/layout/app-shell";
import { MarketStatusBanner } from "@/components/market-status-banner";

export default function AdminLayout({ children }: { children: React.ReactNode }) {
  return <AppShell>{children}</AppShell>;
}
