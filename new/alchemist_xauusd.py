"""
Alchemist Institutional Liquidity Strategy — XAUUSD reference implementation.

Framework-agnostic: feed it pandas DataFrames (index = tz-aware UTC DatetimeIndex,
columns = open/high/low/close/volume) per timeframe and it returns a Signal or None.

Drop into Predict-A-Trade as:
    from strategies.alchemist_xauusd.alchemist_xauusd import AlchemistXAUUSD
    strat = AlchemistXAUUSD.from_json("strategies/alchemist_xauusd/alchemist_xauusd.json")
    signal = strat.evaluate(bars, now=utc_now, spread_points=..., news=...)

No broker calls, no I/O side effects — pure signal generation.
"""

from __future__ import annotations

import json
from dataclasses import dataclass, field, asdict
from datetime import datetime, time, timedelta
from typing import Dict, List, Optional, Literal

import pandas as pd

Direction = Literal["long", "short"]


# --------------------------------------------------------------------------- #
# Data objects
# --------------------------------------------------------------------------- #
@dataclass
class POI:
    tf: str
    kind: str                 # OB | BB | FVG | RejectionBlock
    direction: Direction      # direction it is expected to push price
    level: float              # close of the originating candle
    zone_low: float
    zone_high: float
    created_at: datetime
    mitigated: bool = False

    def contains(self, price: float) -> bool:
        return self.zone_low <= price <= self.zone_high


@dataclass
class Range:
    high: float
    low: float
    start: datetime
    end: datetime

    @property
    def mid(self) -> float:
        return (self.high + self.low) / 2


@dataclass
class Signal:
    symbol: str
    direction: Direction
    setup_id: str
    entry: float
    sl: float
    tp1: float
    tp2: float
    size_lots: float
    confidence: int
    rationale: List[str] = field(default_factory=list)
    generated_at: Optional[datetime] = None

    def to_dict(self) -> dict:
        d = asdict(self)
        d["generated_at"] = self.generated_at.isoformat() if self.generated_at else None
        return d


# --------------------------------------------------------------------------- #
# Primitives
# --------------------------------------------------------------------------- #
def swing_points(df: pd.DataFrame, left: int = 1, right: int = 1):
    """Classic 3-candle swing highs/lows. Returns (highs, lows) as [(ts, price)]."""
    highs, lows = [], []
    for i in range(left, len(df) - right):
        h = df["high"].iloc[i]
        l = df["low"].iloc[i]
        if h > df["high"].iloc[i - left:i].max() and h > df["high"].iloc[i + 1:i + 1 + right].max():
            highs.append((df.index[i], h))
        if l < df["low"].iloc[i - left:i].min() and l < df["low"].iloc[i + 1:i + 1 + right].min():
            lows.append((df.index[i], l))
    return highs, lows


def last_bos(df: pd.DataFrame, lookback: int = 60) -> Optional[Direction]:
    """Break of structure: close beyond the most recent opposing swing."""
    sub = df.iloc[-lookback:]
    highs, lows = swing_points(sub)
    last_close = sub["close"].iloc[-1]
    if not highs or not lows:
        # Fallback for one-directional legs with no interior swing: use range extremes.
        ref = sub.iloc[:-1]
        if ref.empty:
            return None
        if last_close >= ref["high"].max():
            return "long"
        if last_close <= ref["low"].min():
            return "short"
        return None
    if last_close > highs[-1][1]:
        return "long"
    if last_close < lows[-1][1]:
        return "short"
    return None


def atr(df: pd.DataFrame, period: int = 14) -> float:
    h, l, c = df["high"], df["low"], df["close"].shift(1)
    tr = pd.concat([h - l, (h - c).abs(), (l - c).abs()], axis=1).max(axis=1)
    return float(tr.rolling(period).mean().iloc[-1])


def session_slice(df: pd.DataFrame, day: datetime, start: str, end: str) -> pd.DataFrame:
    s = datetime.combine(day.date(), time.fromisoformat(start), tzinfo=day.tzinfo)
    e = datetime.combine(day.date(), time.fromisoformat(end), tzinfo=day.tzinfo)
    return df.loc[(df.index >= s) & (df.index < e)]


def find_order_blocks(df: pd.DataFrame, tf: str, lookback: int = 80,
                      zone_pips: float = 5, pip: float = 0.10) -> List[POI]:
    """
    Order block = last opposite-colour candle that initiated a displacement leg.
    Level = its CLOSE (per Alchemist rules), zone padded by `zone_pips`.
    """
    out: List[POI] = []
    sub = df.iloc[-lookback:]
    body = (sub["close"] - sub["open"]).abs()
    avg_body = body.mean()
    pad = zone_pips * pip
    for i in range(1, len(sub) - 1):
        cur, nxt = sub.iloc[i], sub.iloc[i + 1]
        disp = abs(nxt["close"] - nxt["open"]) > 1.8 * avg_body
        if not disp:
            continue
        # bullish OB: down candle followed by strong up displacement
        if cur["close"] < cur["open"] and nxt["close"] > nxt["open"]:
            out.append(POI(tf, "OB", "long", float(cur["close"]),
                           float(cur["close"]) - pad, float(cur["close"]) + pad, sub.index[i]))
        # bearish OB: up candle followed by strong down displacement
        if cur["close"] > cur["open"] and nxt["close"] < nxt["open"]:
            out.append(POI(tf, "OB", "short", float(cur["close"]),
                           float(cur["close"]) - pad, float(cur["close"]) + pad, sub.index[i]))
    return out


def mark_mitigated(pois: List[POI], df: pd.DataFrame) -> List[POI]:
    for p in pois:
        after = df.loc[df.index > p.created_at]
        if not after.empty and ((after["low"] <= p.zone_high) & (after["high"] >= p.zone_low)).any():
            p.mitigated = True
    return pois


# --------------------------------------------------------------------------- #
# Strategy
# --------------------------------------------------------------------------- #
class AlchemistXAUUSD:
    def __init__(self, cfg: dict):
        self.cfg = cfg
        self.pip = cfg["pip_size"]
        self.symbol = cfg["symbol"]

    @classmethod
    def from_json(cls, path: str) -> "AlchemistXAUUSD":
        with open(path) as fh:
            return cls(json.load(fh))

    # ---------------- Step 1: top-down bias ----------------
    def htf_bias(self, bars: Dict[str, pd.DataFrame]) -> Optional[Direction]:
        weekly = last_bos(bars["W1"], lookback=40)
        daily = last_bos(bars["D1"], lookback=60)
        if weekly is None or daily is None:
            return None
        if self.cfg["bias"]["require_weekly_and_daily_agreement"] and weekly != daily:
            return None
        return weekly

    # ---------------- Step 2: POIs ----------------
    def valid_pois(self, bars: Dict[str, pd.DataFrame], bias: Direction) -> List[POI]:
        pois: List[POI] = []
        for tf in self.cfg["timeframes"]["refinement"]:
            raw = find_order_blocks(bars[tf], tf,
                                    zone_pips=self.cfg["poi"]["refine_zone_pips"][1],
                                    pip=self.pip)
            pois += mark_mitigated(raw, bars[tf])
        return [p for p in pois if not p.mitigated and p.direction == bias]

    # ---------------- Step 3: Asian range ----------------
    def asian_range(self, m15: pd.DataFrame, now: datetime) -> Optional[Range]:
        s = self.cfg["sessions_utc"]["asia"]
        sl = session_slice(m15, now, s["start"], s["end"])
        if sl.empty:
            return None
        return Range(float(sl["high"].max()), float(sl["low"].min()), sl.index[0], sl.index[-1])

    # ---------------- Step 4: killzone ----------------
    def in_window(self, now: datetime, name: str) -> bool:
        w = self.cfg["sessions_utc"][name]
        t = now.timetz().replace(tzinfo=None)
        return time.fromisoformat(w["start"]) <= t < time.fromisoformat(w["end"])

    # ---------------- Step 5: Judas swing / sweep ----------------
    def judas_swing(self, m5: pd.DataFrame, asia: Range, bias: Direction,
                    now: datetime) -> bool:
        kz = session_slice(m5, now, self.cfg["sessions_utc"]["london_killzone"]["start"],
                           self.cfg["sessions_utc"]["london_killzone"]["end"])
        if kz.empty:
            return False
        # bullish bias -> false move DOWN through Asian low; bearish -> up through Asian high
        return bool(kz["low"].min() < asia.low) if bias == "long" else bool(kz["high"].max() > asia.high)

    # ---------------- Step 6: entry confirmation ----------------
    def confirmed(self, m5: pd.DataFrame, bias: Direction, poi: POI) -> bool:
        tapped = ((m5["low"].iloc[-20:] <= poi.zone_high) &
                  (m5["high"].iloc[-20:] >= poi.zone_low)).any()
        return bool(tapped and last_bos(m5, lookback=self.cfg["entry_confirmation"]
                                        ["max_bars_between_sweep_and_bos"] * 3) == bias)

    # ---------------- Risk ----------------
    def stop_distance(self, m15: pd.DataFrame) -> float:
        r = self.cfg["risk"]
        a = atr(m15, r["sl_atr_period"]) * r["sl_atr_multiplier"]
        lo, hi = r["sl_pips_min"] * self.pip, r["sl_pips_max"] * self.pip
        return max(lo, min(a, hi)) if a else hi

    def position_size(self, equity: float, sl_distance: float,
                      contract_size: float = 100.0) -> float:
        risk_cash = equity * self.cfg["risk"]["risk_per_trade_pct"] / 100
        per_lot = sl_distance * contract_size          # gold: 100 oz per lot
        return round(risk_cash / per_lot, 2) if per_lot else 0.0

    # ---------------- Scoring ----------------
    def score(self, flags: Dict[str, bool]) -> int:
        w = self.cfg["scoring"]["weights"]
        p = self.cfg["scoring"]["penalties"]
        s = sum(v for k, v in w.items() if flags.get(k))
        if flags.get("news_within_30min"):
            s += p["news_within_30min"]
        if flags.get("spread_or_atr_out_of_band"):
            s += p["spread_or_atr_out_of_band"]
        return max(0, min(100, s))

    # ---------------- Main entry point ----------------
    def evaluate(self, bars: Dict[str, pd.DataFrame], now: datetime,
                 equity: float = 10_000.0, spread_points: float = 20.0,
                 high_impact_news_within_min: Optional[int] = None) -> Optional[Signal]:

        if now.timetz().replace(tzinfo=None) >= time.fromisoformat(self.cfg["sessions_utc"]["hard_stop"]):
            return None

        bias = self.htf_bias(bars)
        if bias is None:
            return None                                   # hard reject: no clean HTF bias

        news_day = high_impact_news_within_min is not None and high_impact_news_within_min <= 240
        window = "ny_killzone" if news_day and self.cfg["setups"]["setup_3_news_ny"]["enabled"] \
            else "london_killzone"
        if not self.in_window(now, window):
            return None

        m5, m15 = bars["M5"], bars["M15"]
        asia = self.asian_range(m15, now)
        if asia is None:
            return None

        pois = self.valid_pois(bars, bias)
        if not pois:
            return None

        price = float(m5["close"].iloc[-1])
        poi = min(pois, key=lambda p: abs(p.level - price))

        judas = self.judas_swing(m5, asia, bias, now)
        setup_id = ("setup_3_news_ny" if news_day else
                    "setup_1_london_open_trap" if judas else "setup_2_strong_asia")

        if not self.confirmed(m5, bias, poi):
            return None

        sl_dist = self.stop_distance(m15)
        entry = poi.level
        if bias == "long":
            sl, tp1 = entry - sl_dist, asia.high
            tp2 = float(bars["D1"]["high"].iloc[-10:].max())
        else:
            sl, tp1 = entry + sl_dist, asia.low
            tp2 = float(bars["D1"]["low"].iloc[-10:].min())

        rr = abs(tp2 - entry) / sl_dist if sl_dist else 0
        if rr < self.cfg["risk"]["min_rr_to_erl"]:
            return None

        flags = {
            "htf_bias_alignment": True,
            "poi_quality": not poi.mitigated,
            "liquidity_sweep": judas,
            "bos_confirmation": True,
            "in_killzone": True,
            "rr_ok": rr >= self.cfg["risk"]["min_rr_to_erl"],
            "news_within_30min": bool(high_impact_news_within_min is not None
                                      and high_impact_news_within_min <= 30
                                      and setup_id != "setup_3_news_ny"),
            "spread_or_atr_out_of_band": spread_points > self.cfg["risk"]["max_spread_points"],
        }
        conf = self.score(flags)
        if conf < self.cfg["scoring"]["publish_threshold"]:
            return None

        return Signal(
            symbol=self.symbol,
            direction=bias,
            setup_id=setup_id,
            entry=round(entry, 2),
            sl=round(sl, 2),
            tp1=round(tp1, 2),
            tp2=round(tp2, 2),
            size_lots=self.position_size(equity, sl_dist),
            confidence=conf,
            rationale=[
                f"HTF bias {bias} (W1+D1 BOS agree)",
                f"Fresh {poi.tf} {poi.kind} @ {poi.level:.2f}",
                f"Asian range {asia.low:.2f}–{asia.high:.2f}",
                f"Judas swing: {judas}",
                f"Setup: {setup_id}",
                f"SL {sl_dist / self.pip:.0f} pips, R:R to ERL {rr:.1f}",
            ],
            generated_at=now,
        )


__all__ = ["AlchemistXAUUSD", "Signal", "POI", "Range"]
