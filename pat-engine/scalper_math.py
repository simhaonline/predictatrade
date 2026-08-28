#!/usr/bin/env python3
"""scalper_math.py — reproduce the cost / R:R / win-rate math from read.md.

Run:  python3 scalper_math.py

Gold scalping is a cost-and-selectivity problem, not an entry-quality problem.
These numbers let you verify, from first principles, why:
  - sub-$1.00/oz targets are structurally negative-expectancy,
  - at 2:1 R:R you only need ~39.7% win rate to break even,
  - cost-as-%-of-gross must stay under ~10-12%.
Edit the CONSTANTS block to match your own broker before trusting the output.
"""
from __future__ import annotations

# ─── Edit these to match YOUR broker (read.md §2) ───
SPREAD_POINTS = 12.0      # typical XAUUSD spread in points (1 point = $0.10/oz at 100oz/lot)
COMMISSION_PER_LOT = 7.0  # round-turn commission $/lot
CONTRACT_OZ = 100.0       # ounces per standard lot


def cost_per_oz() -> float:
    """Round-turn transaction cost per ounce of exposure.

    XAUUSD convention (read.md §1): 1 point = $0.01/oz, so a 12-point spread is
    $0.12/oz; adding $7/lot commission (= $0.07/oz at 100oz/lot) gives ~$0.19/oz
    round-turn, i.e. ~$19 per 1.0 lot.
    """
    spread_cost_per_oz = SPREAD_POINTS * 0.01
    commission_per_oz = COMMISSION_PER_LOT / CONTRACT_OZ
    return spread_cost_per_oz + commission_per_oz


def break_even_win_rate(rr: float) -> float:
    """Win rate needed to break even at a given reward:risk (ignoring cost)."""
    return 1.0 / (1.0 + rr)


def cost_as_pct_of_gross(target_per_oz: float) -> float:
    c = cost_per_oz()
    return (c / (target_per_oz - c)) * 100.0 if target_per_oz > c else float("inf")


def position_size(equity: float, risk_pct: float, stop_per_oz: float) -> float:
    risk_dollars = equity * risk_pct
    oz_per_lot = CONTRACT_OZ
    lots = risk_dollars / (stop_per_oz * oz_per_lot)
    return lots


def main() -> None:
    c = cost_per_oz()
    print(f"Round-turn cost per oz : ${c:.2f}  (spread ${SPREAD_POINTS/10:.2f} + comm ${COMMISSION_PER_LOT/CONTRACT_OZ:.2f})")
    print(f"Cost per 1.0 lot       : ${c*CONTRACT_OZ:.2f}")
    print()
    print("Break-even win rate (cost excluded):")
    for rr in (1.0, 1.5, 2.0, 2.5):
        print(f"  R:R {rr:>3}: {break_even_win_rate(rr)*100:5.1f}%")
    print()
    print("Cost as % of GROSS profit by target (must stay < ~10-12%):")
    for tgt in (0.50, 1.00, 1.50, 2.50, 4.00):
        pct = cost_as_pct_of_gross(tgt)
        flag = "  <-- NEGATIVE-EXPECTANCY TARGET" if pct == float("inf") else ""
        print(f"  ${tgt:>4}/oz target: {pct:5.1f}%{flag}")
    print()
    print("Position size @ 1% risk:")
    for stop in (1.0, 1.5, 2.0, 2.5):
        print(f"  ${stop}/oz stop on $10k: {position_size(10000, 0.01, stop):.2f} lots")


if __name__ == "__main__":
    main()
