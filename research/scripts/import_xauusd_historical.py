#!/usr/bin/env python3
"""Import XAUUSD historical data from Kaggle CSVs into PostgreSQL market.candles table."""
from __future__ import annotations

import argparse
import csv
import os
import sys
from datetime import datetime, timezone

import psycopg2
from psycopg2.extras import execute_values

KAGGLE_FILES = {
    "M1":  "XAU_1m_data.csv",
    "M5":  "XAU_5m_data.csv",
    "M15": "XAU_15m_data.csv",
    "M30": "XAU_30m_data.csv",
    "H1":  "XAU_1h_data.csv",
    "H4":  "XAU_4h_data.csv",
    "D1":  "XAU_1d_data.csv",
    "W1":  "XAU_1w_data.csv",
    "MN":  "XAU_1Month_data.csv",
}

DB_CANDLES_TABLE = "market.candles"
BATCH_SIZE = 10000
SOURCE_TAG = "KAGGLE_NOVANDRAANUGRAH_2004_2026"


def parse_timestamp(raw: str) -> datetime:
    """Parse '2004.06.11 07:15' to UTC datetime."""
    dt = datetime.strptime(raw.strip(), "%Y.%m.%d %H:%M")
    return dt.replace(tzinfo=timezone.utc)


def load_csv(filepath: str, start, end):
    """Yield (time, open, high, low, close, volume) tuples from Kaggle CSV."""
    with open(filepath, "r", newline="") as f:
        reader = csv.DictReader(f, delimiter=";")
        for row in reader:
            try:
                ts = parse_timestamp(row["Date"])
            except (ValueError, KeyError):
                continue
            if start and ts < start:
                continue
            if end and ts > end:
                continue
            yield (
                ts,
                float(row["Open"]),
                float(row["High"]),
                float(row["Low"]),
                float(row["Close"]),
                int(row.get("Volume", 0) or 0),
            )


def import_timeframe(conn, tf: str, filepath: str, start, end):
    """Import one timeframe into market.candles."""
    if not os.path.exists(filepath):
        print(f"  [!] {tf}: file not found, skipping")
        return 0

    batch = []
    total = 0
    min_ts = max_ts = None

    for ts, o, h, l, c, v in load_csv(filepath, start, end):
        if min_ts is None:
            min_ts = ts
        max_ts = ts
        batch.append((ts, "XAUUSD", tf, o, h, l, c, v,
                      SOURCE_TAG, "COMPLETE", "BROKER_ALIGNED_UTC", True))
        if len(batch) >= BATCH_SIZE:
            write_batch(conn, batch)
            total += len(batch)
            batch = []
            print(f"  {tf}: {total:,} rows...", end="\r")

    if batch:
        write_batch(conn, batch)
        total += len(batch)

    print(f"  [OK] {tf:4s}: {total:>10,} rows | {min_ts} -> {max_ts}")
    return total


def write_batch(conn, batch):
    with conn.cursor() as cur:
        execute_values(cur, f"""
            INSERT INTO {DB_CANDLES_TABLE}
              (time, symbol, timeframe, open, high, low, close, volume,
               source, quality, alignment_profile, is_closed)
            VALUES %s
            ON CONFLICT (time, symbol, timeframe, source) DO NOTHING
        """, batch)
    conn.commit()


def main():
    parser = argparse.ArgumentParser(description="Import XAUUSD historical data from Kaggle CSVs")
    parser.add_argument("--data-dir", required=True)
    parser.add_argument("--timeframes", default="M1,M5,M15,M30,H1,H4,D1,W1,MN")
    parser.add_argument("--start", default=None, help="Start date (YYYY-MM-DD)")
    parser.add_argument("--end", default=None, help="End date (YYYY-MM-DD)")
    parser.add_argument("--db-url", default=None)
    args = parser.parse_args()

    db_url = args.db_url
    if not db_url:
        url_file = "/srv/predictatrade/xauusd/database_url.txt"
        if os.path.exists(url_file):
            with open(url_file) as f:
                db_url = f.read().strip()
        else:
            print("ERROR: No --db-url and no database_url.txt")
            return 1

    start = datetime.fromisoformat(args.start + "T00:00:00+00:00") if args.start else None
    end = datetime.fromisoformat(args.end + "T23:59:59+00:00") if args.end else None
    timeframes = args.timeframes.split(",")

    print(f"{'='*70}")
    print(f"XAUUSD Historical Data Import")
    print(f"  Source: Kaggle (novandraanugrah/xauusd-gold-price-historical-data-2004-2024)")
    print(f"  Timeframes: {timeframes}")
    print(f"  Date range: {start or 'beginning'} -> {end or 'end'}")
    print(f"{'='*70}")

    conn = psycopg2.connect(db_url)
    total_all = 0
    for tf in timeframes:
        filename = KAGGLE_FILES.get(tf)
        if not filename:
            print(f"  [!] Unknown timeframe {tf}")
            continue
        filepath = os.path.join(args.data_dir, filename)
        total_all += import_timeframe(conn, tf, filepath, start, end)

    conn.close()
    print(f"\n{'='*70}")
    print(f"Import complete: {total_all:,} total rows")
    print(f"{'='*70}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
