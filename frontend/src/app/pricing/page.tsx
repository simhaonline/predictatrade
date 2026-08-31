"use client";

import { useQuery } from "@tanstack/react-query";
import { customInstance } from "@/lib/axios-instance";
import { strategyLabel } from "@/lib/strategy-labels";
import Link from "next/link";
import { IconCheck, IconStar } from "@tabler/icons-react";

interface Plan {
  id: string;
  code: string;
  name: string;
  description: string | null;
  monthly_price: string;
  annual_price: string | null;
  allowed_strategies: string[];
  max_active_strategy_slots: number;
  max_signals_per_day: number | null;
  annual_savings_percent: number | null;
  referral_eligible: boolean;
}

const PLAN_ORDER = ["FREE", "STANDARD", "PRO", "ELITE"];

const PLAN_DISPLAY: Record<string, { title: string; blurb: string; highlight?: boolean }> = {
  FREE: {
    title: "Free",
    blurb: "Start with our foundational scalping strategy. No card required.",
  },
  BASIC: {
    title: "Basic",
    blurb: "Foundational strategies plus Arcanist (IMLR) — institutional liquidity reversals.",
  },
  STANDARD: {
    title: "Standard",
    blurb: "Two core strategies plus Arcanist (IMLR) institutional liquidity reversals.",
  },
  PRO: {
    title: "Pro",
    blurb: "All four core strategies plus Arcanist (IMLR) — our 7th engine.",
  },
  ELITE: {
    title: "Elite Pro 499",
    blurb: "Everything in Pro, plus EQFE, ATEN and Arcanist (IMLR) — our premium engines.",
    highlight: true,
  },
};

export default function PricingPage() {
  const { data, isLoading } = useQuery({
    queryKey: ["public-plans"],
    queryFn: async () => (await customInstance.get("/public/plans")).data as Plan[],
  });

  const plans = (data ?? [])
    .slice()
    .sort((a, b) => PLAN_ORDER.indexOf(a.code) - PLAN_ORDER.indexOf(b.code));

  return (
    <main style={{ minHeight: "100vh", background: "#0e1116", color: "#e6e8ec", padding: "48px 16px" }}>
      <div style={{ maxWidth: 1100, margin: "0 auto" }}>
        <div style={{ textAlign: "center", marginBottom: 8 }}>
          <span style={{ fontSize: 13, fontWeight: 700, letterSpacing: 1, color: "#7aa2ff", textTransform: "uppercase" }}>
            Predict-A-Trade Plans
          </span>
          <h1 style={{ fontSize: 34, fontWeight: 800, margin: "8px 0 6px" }}>
            Trade XAUUSD with AI-grade signal intelligence
          </h1>
          <p style={{ color: "#9aa3b2", maxWidth: 620, margin: "0 auto", fontSize: 15, lineHeight: 1.5 }}>
            Four tiers, from a free starter strategy to the full Elite suite. Cancel anytime.
            Referrals reward you only when someone you refer upgrades to a paid plan.
          </p>
        </div>

        {isLoading && (
          <p style={{ textAlign: "center", color: "#9aa3b2", marginTop: 40 }}>Loading plans…</p>
        )}

        <div
          style={{
            display: "grid",
            gridTemplateColumns: "repeat(auto-fit, minmax(240px, 1fr))",
            gap: 18,
            marginTop: 36,
          }}
        >
          {plans.map((plan) => {
            const meta = PLAN_DISPLAY[plan.code] ?? { title: plan.name, blurb: plan.description ?? "" };
            const price = Number(plan.monthly_price || 0);
            const annual = plan.annual_price != null;
            const displayPrice = annual ? Number(plan.annual_price) : price;
            return (
              <div
                key={plan.id}
                style={{
                  background: meta.highlight ? "#161b2e" : "#15181f",
                  border: meta.highlight ? "1px solid #3b5bdb" : "1px solid #232733",
                  borderRadius: 14,
                  padding: 22,
                  display: "flex",
                  flexDirection: "column",
                  position: "relative",
                }}
              >
                {meta.highlight && (
                  <span
                    style={{
                      position: "absolute",
                      top: 14,
                      right: 14,
                      fontSize: 11,
                      fontWeight: 700,
                      color: "#cdd9ff",
                      background: "#1e2a52",
                      border: "1px solid #3b5bdb",
                      borderRadius: 999,
                      padding: "3px 10px",
                    }}
                  >
                    MOST POPULAR
                  </span>
                )}
                <h2 style={{ fontSize: 20, fontWeight: 700, margin: 0 }}>{meta.title}</h2>
                <div style={{ marginTop: 10, fontSize: 30, fontWeight: 800 }}>
                  ${displayPrice.toFixed(0)}
                  <span style={{ fontSize: 13, fontWeight: 500, color: "#9aa3b2" }}>
                    /{annual ? "year" : "month"}
                  </span>
                </div>
                {annual && plan.annual_savings_percent != null && (
                  <div style={{ fontSize: 12, color: "#5bd6a0", marginTop: 2 }}>
                    Save {plan.annual_savings_percent}% with annual billing
                  </div>
                )}
                <p style={{ fontSize: 13, color: "#9aa3b2", marginTop: 12, minHeight: 54, lineHeight: 1.45 }}>
                  {meta.blurb}
                </p>

                <div style={{ fontSize: 12, color: "#7e879a", margin: "10px 0 6px", textTransform: "uppercase", letterSpacing: 0.5 }}>
                  Included strategies
                </div>
                <ul style={{ listStyle: "none", padding: 0, margin: 0, display: "flex", flexDirection: "column", gap: 6 }}>
                  {plan.allowed_strategies.map((s) => (
                    <li key={s} style={{ display: "flex", alignItems: "center", gap: 8, fontSize: 13 }}>
                      <IconCheck size={15} color="#5bd6a0" />
                      {strategyLabel(s)}
                    </li>
                  ))}
                </ul>

                <div style={{ marginTop: 14, fontSize: 12, color: "#7e879a", lineHeight: 1.5 }}>
                  {plan.max_signals_per_day
                    ? `Limited to ${plan.max_signals_per_day} signals / day`
                    : "Unlimited signals"}
                  {" · "}
                  {plan.max_active_strategy_slots} active strateg{plan.max_active_strategy_slots === 1 ? "y" : "ies"}
                </div>

                {plan.referral_eligible && (
                  <div style={{ marginTop: 8, fontSize: 12, color: "#c9a4ff", display: "flex", alignItems: "center", gap: 6 }}>
                    <IconStar size={14} /> Referral-eligible plan
                  </div>
                )}

                <Link
                  href={plan.code === "FREE" ? "/register" : `/register`}
                  style={{
                    marginTop: 18,
                    display: "block",
                    textAlign: "center",
                    background: meta.highlight ? "#3b5bdb" : "#222838",
                    color: "#fff",
                    border: meta.highlight ? "none" : "1px solid #2e3548",
                    borderRadius: 10,
                    padding: "11px 0",
                    fontSize: 14,
                    fontWeight: 600,
                    textDecoration: "none",
                  }}
                >
                  {plan.code === "FREE" ? "Create free account" : "Get started"}
                </Link>
              </div>
            );
          })}
        </div>

        <p style={{ textAlign: "center", color: "#6b7280", fontSize: 12, marginTop: 36, lineHeight: 1.6 }}>
          Prices in USD. EQFE (MARNIE_FIB), ATEN and Arcanist (IMLR) are proprietary engines available on all paid plans.
          <br />
          Already a member? <Link href="/login" style={{ color: "#7aa2ff" }}>Sign in</Link> to manage your subscription.
        </p>
      </div>
    </main>
  );
}
