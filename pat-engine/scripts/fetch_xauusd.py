#!/usr/bin/env python3
"""
fetch_xauusd.py — download REAL XAUUSD 1-minute bars from the Dukascopy public
historical feed and write them to a CSV the pat-engine backtester consumes
(time,open,high,low,close,spread — all in price units, UTC).

Why this exists: the backtester defaults to SYNTHETIC data, which has no real
edge and must never be reported as a genuine result. Run this to obtain real
market history (real prices AND real bid/ask spread) and point BARS_CSV at it.

Notes
-----
* Source: https://www.dukascopy.com/datafeed/<SYMBOL>/<YYYY>/<MM>/<DD>/{BID,ASK}_candles_min_1.bi5
  These are LZMA-compressed binary files, 24-byte big-endian records:
  (time_min:int32, open:int32, high:int32, low:int32, close:int32, vol:int32).
  Prices are scaled by 1/--divisor (default 1000, verified for 2024-2025 gold).
* Dukascopy's free feed lags "today" by some days/weeks, so the most recent
  calendar dates may 404. The script skips missing days and reports coverage.
* A per-day continuity check drops days whose median price jumps >30% vs the
  previous valid day (guards against the occasional corrupted/instrument-mismatched
  file the public endpoint sometimes serves).

Usage
-----
  python3 scripts/fetch_xauusd.py --start 2025-01-01 --end 2025-04-30 \
      --out data/xauusd_m1.csv

Then:
  BARS_CSV=data/xauusd_m1.csv go run ./cmd/backtest
"""
import argparse
import csv
import lzma
import struct
import sys
import time as time_mod
import urllib.request
import datetime as dt

BASE = "https://www.dukascopy.com/datafeed/{symbol}/{y}/{m}/{d}/{side}_candles_min_1.bi5"
REC = 24
DIVISOR = 1000
UA = {"User-Agent": "Mozilla/5.0 (pat-engine backtest data fetch)"}


def fetch(url, tries=6, timeout=30):
    """Fetch a URL with polite rate-limiting and exponential backoff on 503.

    Dukascopy's public endpoint load-balances and returns intermittent 503s under
    bulk access; backing off and retrying is required for a full download.
    """
    import random
    last = None
    for attempt in range(tries):
        try:
            time_mod.sleep(0.3 + random.random() * 0.5)
            req = urllib.request.Request(url, headers=UA)
            with urllib.request.urlopen(req, timeout=timeout) as r:
                data = r.read()
            if data[:1] == b"<":  # HTML error page
                return None
            return data
        except urllib.error.HTTPError as e:
            last = e
            if e.code == 503:
                time_mod.sleep(min(20, 2 ** (attempt + 1)))
                continue
            return None
        except Exception as e:  # noqa
            last = e
            time_mod.sleep(min(20, 2 ** (attempt + 1)))
            continue
    sys.stderr.write(f"  WARN fetch failed {url}: {last}\n")
    return None


def decode_bi5(blob):
    raw = lzma.decompress(blob)
    n = len(raw) // REC
    out = []
    for k in range(n):
        t, o, h, l, c, _v = struct.unpack(">iiiiii", raw[k * REC:(k + 1) * REC])
        out.append((t, o, h, l, c))
    return out


def day_bars(symbol, date, divisor):
    y, m, d = date.strftime("%Y"), date.strftime("%m"), date.strftime("%d")
    bid = fetch(BASE.format(symbol=symbol, y=y, m=m, d=d, side="BID"))
    ask = fetch(BASE.format(symbol=symbol, y=y, m=m, d=d, side="ASK"))
    if bid is None:
        return None
    bid = decode_bi5(bid)
    if ask is None:
        # No ask feed for this day: build bid-only bars, spread left as 0 (the
        # backtester will then use its configured typical spread as a fallback).
        ask = None
    else:
        ask = decode_bi5(ask)
        if len(ask) != len(bid):
            ask = None

    day_start = int(dt.datetime.combine(date, dt.time(0, 0)).timestamp())
    rows = []
    for k, (tmin, o, h, l, c) in enumerate(bid):
        ts = day_start + tmin * 60
        if ask is not None:
            ao, ah, al, ac = ask[k][1], ask[k][2], ask[k][3], ask[k][4]
            spread = (ac - c) / divisor
            if spread < 0:
                spread = (max(ah, h) - min(al, l)) / divisor  # safety
            o2, h2, l2, c2 = o / divisor, h / divisor, l / divisor, c / divisor
        else:
            spread = 0.0
            o2, h2, l2, c2 = o / divisor, h / divisor, l / divisor, c / divisor
        rows.append((ts, o2, h2, l2, c2, round(spread, 4)))
    return rows


def median(seq):
    s = sorted(seq)
    n = len(s)
    if n == 0:
        return 0
    return s[n // 2]


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--symbol", default="XAUUSD")
    ap.add_argument("--start", required=True)
    ap.add_argument("--end", required=True)
    ap.add_argument("--out", required=True)
    ap.add_argument("--divisor", type=float, default=DIVISOR)
    ap.add_argument("--max-gap", type=float, default=0.30,
                    help="drop days whose median price jumps >30%% vs previous valid day")
    args = ap.parse_args()

    start = dt.datetime.strptime(args.start, "%Y-%m-%d").date()
    end = dt.datetime.strptime(args.end, "%Y-%m-%d").date()

    prev_med = None
    written = 0
    days_ok = 0
    days_skip = 0
    price_min, price_max = 1e18, -1e18
    spread_sum, spread_n = 0.0, 0

    tmp = args.out + ".tmp"
    with open(tmp, "w", newline="") as f:
        w = csv.writer(f)
        w.writerow(["time", "open", "high", "low", "close", "spread"])
        cur = start
        while cur <= end:
            rows = day_bars(args.symbol, cur, args.divisor)
            cur += dt.timedelta(days=1)
            if not rows:
                days_skip += 1
                continue
            med = median([r[4] for r in rows])
            if prev_med is not None and med > 0:
                ratio = med / prev_med
                if ratio < (1 - args.max_gap) or ratio > (1 + args.max_gap):
                    sys.stderr.write(
                        f"  SKIP {cur}: median {med:.1f} jumps {ratio:.2f}x vs prev {prev_med:.1f}\n")
                    days_skip += 1
                    continue
            prev_med = med
            for r in rows:
                w.writerow(r)
                written += 1
                price_min = min(price_min, r[4])
                price_max = max(price_max, r[4])
                if r[5] > 0:
                    spread_sum += r[5]
                    spread_n += 1
            days_ok += 1

    import os
    os.replace(tmp, args.out)

    print(f"WROTE {written} bars ({days_ok} days, {days_skip} skipped) -> {args.out}")
    if written:
        print(f"price range: {price_min:.2f} .. {price_max:.2f}")
        if spread_n:
            print(f"avg real spread: {spread_sum / spread_n:.3f}")
        else:
            print("avg real spread: n/a (ask feed unavailable; backtester uses configured typical spread)")


if __name__ == "__main__":
    main()
