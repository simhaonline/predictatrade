// Package store persists bars and signals to TimescaleDB (database pat_engine) and
// publishes live signals through Valkey. It is intentionally DEGRADABLE: if Postgres
// or Valkey is unreachable the engine keeps running on an in-memory ring buffer, so a
// missing datastore never blocks signal generation ("never gets stuck").
package store

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"pat-engine/internal/backtest"
)

const (
	redisLatestFmt = "pat:signal:latest:%s" // strategy_id
	redisChan      = "pat:signals"          // pub/sub for live pushes
)

// SignalRecord is the audit row for every decision (executable or blocked).
type SignalRecord struct {
	ID          string
	TS          time.Time
	Symbol      string
	StrategyID  string
	Direction   string
	Entry       float64
	SL          float64
	TP1         float64
	TP2         float64
	TP3         float64
	RawScore    float64
	Grade       string
	SignalClass string
	Status      string // EXECUTABLE | BLOCKED
	Reasons     []string
}

// Store wraps Postgres + Valkey with an in-memory fallback.
type Store struct {
	pg  *pgxpool.Pool
	rdb *redis.Client

	mu      sync.Mutex
	memBars []backtest.Bar
	memSig  []SignalRecord
}

// New connects to Postgres and Valkey. On failure it logs and returns a degraded
// store that only uses the in-memory buffer.
func New(ctx context.Context, pgDSN, redisURL string) *Store {
	s := &Store{}
	if pgDSN != "" {
		pool, err := pgxpool.New(ctx, pgDSN)
		if err != nil {
			log.Printf("store: postgres unavailable (%v) — running degraded (in-memory only)", err)
		} else {
			s.pg = pool
		}
	}
	if redisURL != "" {
		rdb := redis.NewClient(&redis.Options{Addr: addrFromURL(redisURL)})
		if err := rdb.Ping(ctx).Err(); err != nil {
			log.Printf("store: valkey unavailable (%v) — skipping cache/pubsub", err)
		} else {
			s.rdb = rdb
		}
	}
	return s
}

// Healthy reports whether both backends are connected.
func (s *Store) Healthy() (pg, valkey bool) {
	return s.pg != nil, s.rdb != nil
}

// SaveBar persists an ingested bar.
func (s *Store) SaveBar(ctx context.Context, b backtest.Bar) {
	s.mu.Lock()
	s.memBars = append(s.memBars, b)
	if len(s.memBars) > 5000 {
		s.memBars = s.memBars[len(s.memBars)-5000:]
	}
	s.mu.Unlock()

	if s.pg == nil {
		return
	}
	ts := time.Now()
	if b.Time > 0 {
		ts = time.Unix(b.Time, 0)
	}
	_, err := s.pg.Exec(ctx,
		`INSERT INTO bars(ts,symbol,open,high,low,close,spread) VALUES($1,'XAUUSD',$2,$3,$4,$5,$6)`,
		ts, b.Open, b.High, b.Low, b.Close, b.Spread)
	if err != nil {
		log.Printf("store: save bar: %v", err)
	}
}

// SaveSignal persists a decision and, if Valkey is up, caches + publishes it.
func (s *Store) SaveSignal(ctx context.Context, r SignalRecord) {
	s.mu.Lock()
	s.memSig = append(s.memSig, r)
	if len(s.memSig) > 5000 {
		s.memSig = s.memSig[len(s.memSig)-5000:]
	}
	s.mu.Unlock()

		if s.pg != nil {
		reasons, _ := json.Marshal(r.Reasons)
		_, err := s.pg.Exec(ctx,
			`INSERT INTO signals(id,ts,symbol,strategy_id,direction,entry,sl,tp1,tp2,tp3,raw_score,grade,signal_class,status,reasons)
			 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`,
			r.ID, r.TS, r.Symbol, r.StrategyID, r.Direction, r.Entry, r.SL, r.TP1, r.TP2, r.TP3,
			r.RawScore, r.Grade, r.SignalClass, r.Status, string(reasons))
		if err != nil {
			log.Printf("store: save signal: %v", err)
		}
	}

	if s.rdb != nil {
		b, _ := json.Marshal(r)
		if err := s.rdb.Set(ctx, fmt.Sprintf(redisLatestFmt, r.StrategyID), b, 24*time.Hour).Err(); err != nil {
			log.Printf("store: cache signal: %v", err)
		}
		if err := s.rdb.Publish(ctx, redisChan, b).Err(); err != nil {
			log.Printf("store: publish signal: %v", err)
		}
	}
}

// RecentSignals returns the latest N decisions (from Postgres if available).
func (s *Store) RecentSignals(ctx context.Context, n int) []SignalRecord {
	if s.pg != nil {
		rows, err := s.pg.Query(ctx,
			`SELECT id,ts,strategy_id,direction,entry,sl,tp1,raw_score,grade,status,reasons
			 FROM signals ORDER BY ts DESC LIMIT $1`, n)
		if err == nil {
			defer rows.Close()
			var out []SignalRecord
			for rows.Next() {
				var r SignalRecord
				var reasons []byte
				_ = rows.Scan(&r.ID, &r.TS, &r.StrategyID, &r.Direction, &r.Entry, &r.SL, &r.TP1, &r.RawScore, &r.Grade, &r.Status, &reasons)
				_ = json.Unmarshal(reasons, &r.Reasons)
				out = append(out, r)
			}
			return out
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]SignalRecord, 0, n)
	for i := len(s.memSig) - 1; i >= 0 && len(out) < n; i-- {
		out = append(out, s.memSig[i])
	}
	return out
}

// Close releases connections.
func (s *Store) Close() {
	if s.pg != nil {
		s.pg.Close()
	}
	if s.rdb != nil {
		_ = s.rdb.Close()
	}
}

// ---- Device / telemetry (license misuse control + monitoring) ----

// Device is a registered agent installation bound to a license via hardware fingerprint.
type Device struct {
	ID          string
	LicenseID   string
	Fingerprint string // hash
	Components  string // json
	InstallID   string
	Hostname    string
	OS          string
}

// Telemetry is one agent heartbeat sample.
type Telemetry struct {
	DeviceID       string
	LatencyMs      float64
	MT4Conn        bool
	MT5Conn        bool
	Broker         string
	AccountMasked  string
	Equity         float64
	Balance        float64
	OpenPositions  int
	FloatingPnL    float64
	CPU            float64
	RAM            float64
	Version        string
	Status         string
}

// UpsertDevice records/refreshes a device binding.
func (s *Store) UpsertDevice(ctx context.Context, d Device) {
	if s.pg == nil {
		return
	}
	_, err := s.pg.Exec(ctx,
		`INSERT INTO devices(id,license_id,fingerprint_hash,fingerprint_components,installation_id,hostname,os,last_seen)
		 VALUES($1,$2,$3,$4,$5,$6,$7,now())
		 ON CONFLICT (id) DO UPDATE SET license_id=$2, fingerprint_hash=$3, hostname=$6, os=$7, last_seen=now()`,
		d.ID, d.LicenseID, d.Fingerprint, d.Components, d.InstallID, d.Hostname, d.OS)
	if err != nil {
		log.Printf("store: upsert device: %v", err)
	}
}

// GetDevice returns a registered device by id, or nil if not found/absent.
func (s *Store) GetDevice(ctx context.Context, id string) *Device {
	if s.pg == nil || id == "" {
		return nil
	}
	var d Device
	err := s.pg.QueryRow(ctx,
		`SELECT id,license_id,fingerprint_hash,fingerprint_components,installation_id,hostname,os
		 FROM devices WHERE id=$1`, id).
		Scan(&d.ID, &d.LicenseID, &d.Fingerprint, &d.Components, &d.InstallID, &d.Hostname, &d.OS)
	if err != nil {
		return nil
	}
	return &d
}

// SaveTelemetry records one heartbeat sample.
func (s *Store) SaveTelemetry(ctx context.Context, t Telemetry) {
	if s.pg == nil {
		return
	}
	_, err := s.pg.Exec(ctx,
		`INSERT INTO device_telemetry(device_id,ts,latency_ms,mt4_conn,mt5_conn,broker,account_masked,equity,balance,open_positions,floating_pnl,cpu_pct,ram_pct,version,status)
		 VALUES($1,now(),$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		t.DeviceID, t.LatencyMs, t.MT4Conn, t.MT5Conn, t.Broker, t.AccountMasked, t.Equity, t.Balance,
		t.OpenPositions, t.FloatingPnL, t.CPU, t.RAM, t.Version, t.Status)
	if err != nil {
		log.Printf("store: save telemetry: %v", err)
	}
}

// RecentBars returns the latest n bars (newest first).
func (s *Store) RecentBars(ctx context.Context, n int) []backtest.Bar {
	if s.pg == nil {
		s.mu.Lock()
		defer s.mu.Unlock()
		out := make([]backtest.Bar, 0, n)
		for i := len(s.memBars) - 1; i >= 0 && len(out) < n; i-- {
			out = append(out, s.memBars[i])
		}
		return out
	}
	rows, err := s.pg.Query(ctx, `SELECT ts,symbol,open,high,low,close,spread FROM bars ORDER BY ts DESC LIMIT $1`, n)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []backtest.Bar
	for rows.Next() {
		var ts time.Time
		var sym string
		var o, h, l, c, sp float64
		_ = rows.Scan(&ts, &sym, &o, &h, &l, &c, &sp)
		out = append(out, backtest.Bar{Time: ts.Unix(), Open: o, High: h, Low: l, Close: c, Spread: sp})
	}
	return out
}

func addrFromURL(u string) string {
	// accept redis://host:port or host:port
	if len(u) > 8 && u[:8] == "redis://" {
		return u[8:]
	}
	return u
}
