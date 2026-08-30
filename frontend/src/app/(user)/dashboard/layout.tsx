"use client";
import AppShell from "@/components/layout/app-shell";
import { MarketStatusBanner } from "@/components/market-status-banner";

export default function UserLayout({ children }: { children: React.ReactNode }) {
  return <AppShell><MarketStatusBanner />{children}</AppShell>;
}
