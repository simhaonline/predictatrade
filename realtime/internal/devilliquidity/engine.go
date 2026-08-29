package devilliquidity

import (
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// rollingBuf is a fixed-capacity ring buffer for median-based statistics.
type rollingBuf struct {
	vals []float64
	pos  int
	full bool
}

func newRollingBuf(cap int) *rollingBuf {
	if cap < 1 {
		cap = 1
	}
	return &rollingBuf{vals: make([]float64, cap)}
}

func (r *rollingBuf) add(v float64) {
	if v <= 0 {
		return
	}
	r.vals[r.pos] = v
	r.pos++
	if r.pos >= len(r.vals) {
		r.pos = 0
		r.full = true
	}
}

func (r *rollingBuf) median() float64 {
	if len(r.vals) == 0 {
		return 0
	}
	n := len(r.vals)
	if !r.full {
		n = r.pos
	}
	if n == 0 {
		return 0
	}
	cp := make([]float64, n)
	copy(cp, r.vals[:n])
	// simple insertion sort median
	for i := 1; i < n; i++ {
		v := cp[i]
		j := i - 1
		for j >= 0 && cp[j] > v {
			cp[j+1] = cp[j]
			j--
		}
		cp[j+1] = v
	}
	if n%2 == 1 {
		return cp[n/2]
	}
	return (cp[n/2-1] + cp[n/2]) / 2
}

// Engine tracks Devil's Marks per symbol/timeframe.
type Engine struct {
	mu      sync.RWMutex
	cfg     Config
	dbURL   string
	enabled bool

	marks map[string]*DevilMark // by mark id

	// per (symbol|tf) rolling context
	bodies    map[string]*rollingBuf
	vols      map[string]*rollingBuf
	atrEMA    map[string]float64
	prevClose map[string]float64

	// liveness / observability stats
	candlesProcessed int64
	marksCreated     int64
	lastCandleTime   time.Time
	symbolsSeen      map[string]int

	// event sink
	onEvent func(DevilEvent)

	store *Store
}

// AttachStore enables best-effort persistence and routes events to the store.
func (e *Engine) AttachStore(s *Store) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.store = s
	e.onEvent = func(ev DevilEvent) {
		if s != nil {
			_ = s.InsertEvent(ev)
		}
	}
}

func (e *Engine) persistMark(m *DevilMark) {
	if e.store != nil {
		_ = e.store.UpsertMark(m)
	}
}

// NewEngine constructs the Devil Liquidity engine. A blank dbURL disables
// persistence (still keeps in-memory state, useful for tests).
func NewEngine(dbURL string, cfg Config) *Engine {
	if cfg.ConfigVersion == "" {
		cfg = DefaultConfig()
	}
	e := &Engine{
		cfg:       cfg,
		dbURL:     dbURL,
		enabled:   cfg.Enabled,
		marks:     make(map[string]*DevilMark),
		bodies:    make(map[string]*rollingBuf),
		vols:      make(map[string]*rollingBuf),
		atrEMA:    make(map[string]float64),
		prevClose: make(map[string]float64),
	}
	return e
}

// SetEventSink registers a callback for emitted lifecycle events.
func (e *Engine) SetEventSink(fn func(DevilEvent)) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.onEvent = fn
}

// SetEnabled toggles the engine at runtime.
func (e *Engine) SetEnabled(v bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.enabled = v
}

func (e *Engine) key(symbol string, tf string) string { return symbol + "|" + tf }

// Ingest is the non-blocking entry called from the main candle loop.
func (e *Engine) Ingest(candle *CandleInput) {
	if candle == nil || !candle.IsClosed {
		return
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				// never crash the main loop
			}
		}()
		if err := e.ProcessCandle(candle); err != nil {
			return
		}
	}()
}

// CandleInput is the minimal candle view the engine consumes.
type CandleInput struct {
	Symbol     string
	Timeframe  string
	Time       time.Time
	Open       decimal.Decimal
	High       decimal.Decimal
	Low        decimal.Decimal
	Close      decimal.Decimal
	Volume     int64
	IsClosed   bool
	Spread     float64
	Digits     int
	FeedSource string
	Broker     string
	ServerID   string
}

func (e *Engine) getBuf(m map[string]*rollingBuf, k string, cap int) *rollingBuf {
	if b, ok := m[k]; ok {
		return b
	}
	b := newRollingBuf(cap)
	m[k] = b
	return b
}

// ProcessCandle evaluates a single completed candle for detection and advances
// all active marks. It is safe to call from tests directly (synchronous).
func (e *Engine) ProcessCandle(c *CandleInput) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.enabled {
		return nil
	}
	e.candlesProcessed++
	e.lastCandleTime = c.Time
	if e.symbolsSeen == nil {
		e.symbolsSeen = make(map[string]int)
	}
	e.symbolsSeen[c.Symbol]++
	k := e.key(c.Symbol, c.Timeframe)
	o, h, l, cl := decF(c.Open), decF(c.High), decF(c.Low), decF(c.Close)
	rng := h - l
	if rng <= 0 {
		return nil
	}
	body := abs(cl - o)
	bodyRatio := body / rng
	upperWick := h - max(o, cl)
	lowerWick := min(o, cl) - l
	upperWickRatio := upperWick / rng
	lowerWickRatio := lowerWick / rng

	// Update rolling stats + ATR EMA (proxy when external ATR unavailable).
	bodies := e.getBuf(e.bodies, k, 50)
	vols := e.getBuf(e.vols, k, 50)
	bodies.add(body)
	vols.add(float64(c.Volume))
	medianBody := bodies.median()
	medianVol := vols.median()
	var atr float64
	if p, ok := e.prevClose[k]; ok {
		tr := max3(h-l, abs(h-p), abs(l-p))
		if prev, ok2 := e.atrEMA[k]; ok2 && prev > 0 {
			atr = prev + (tr-prev)*(2.0/15.0)
		} else {
			atr = tr
		}
	} else {
		atr = rng // seed
	}
	if atr <= 0 {
		atr = rng
	}
	e.atrEMA[k] = atr
	e.prevClose[k] = cl

	volumeRatio := 1.0
	if medianVol > 0 {
		volumeRatio = float64(c.Volume) / medianVol
	}
	bodyExp := 1.0
	if medianBody > 0 {
		bodyExp = body / medianBody
	}
	rangeATR := rng / atr

	// Detect potential new marks on this closed candle.
	before := make(map[string]bool, len(e.marks))
	for id := range e.marks {
		before[id] = true
	}
	e.detect(c, o, h, l, cl, rng, body, bodyRatio, upperWick, lowerWick,
		upperWickRatio, lowerWickRatio, atr, rangeATR, bodyExp,
		volumeRatio, c.Volume)

	// Advance active marks (skip marks created on THIS candle so a mark is
	// never touched/swept by its own origin candle — non-repaint rule).
	for _, m := range e.marks {
		if m.Symbol != c.Symbol || m.Timeframe != c.Timeframe {
			continue
		}
		if isTerminal(m.State) {
			continue
		}
		if !before[m.ID] {
			continue
		}
		e.advance(m, c, o, h, l, cl, atr)
	}
	return nil
}

func (e *Engine) detect(c *CandleInput, o, h, l, cl, rng, body, bodyRatio,
	upperWick, lowerWick, upperWickRatio, lowerWickRatio, atr, rangeATR,
	bodyExp, volumeRatio float64, vol int64) {

	cfg := e.cfg
	flatTol := max(float64(cfg.MinimumTickTol)*cfg.TickSize, atr*cfg.FlatWickATRTol)

	displacementOK := bodyRatio >= cfg.MinBodyRatio &&
		rangeATR >= cfg.MinRangeATR &&
		bodyExp >= cfg.MinBodyExpansion

	// Duplicate-mark guard: skip detection when a young, non-terminal mark for
	// this (symbol, timeframe) already exists. Without this, the candle right
	// after a displacement (e.g. the reversal leg) can itself qualify as a new
	// displacement — the median body shifted after the first mark — and a second
	// mark is registered with a different level but the same intent, so the
	// level gets double-charged. Range-based level comparison is not enough:
	// ATR doubles after the first displacement, so level distance must be
	// normalized by the mark's DETECTION ATR, and a recency guard covers the
	// reversal-candle case where the new "level" is a full displacement away.
	for _, m := range e.marks {
		if m.Symbol != c.Symbol || m.Timeframe != c.Timeframe {
			continue
		}
		if isTerminal(m.State) {
			continue
		}
		// Level match normalized by the ATR at the existing mark's detection.
		atrRef := m.DetectedATR
		if atrRef <= 0 {
			atrRef = m.ATR
		}
		if atrRef > 0 && m.MarkPrice > 0 &&
			abs(m.MarkPrice-cl) <= cfg.TouchToleranceATR*atrRef {
			return
		}
		// Recency: a displacement within RECLAIM_MAX_BARS of the existing mark
		// is continuation of the same move, not a new mark.
		interval := types.Timeframe(c.Timeframe).Duration()
		if interval <= 0 {
			interval = 5 * time.Minute // conservative default (M5)
		}
		age := c.Time.Sub(m.DetectedAt)
		if age >= 0 && age <= time.Duration(cfg.ReclaimMaxBars)*interval {
			return
		}
	}

	// Bullish: close>open, essentially no lower wick, close near high.
	if cl > o &&
		lowerWick <= flatTol &&
		lowerWickRatio <= cfg.FlatWickRatio &&
		displacementOK &&
		(h-cl)/rng <= cfg.CloseExtremeRatio {
		e.createMark(c, DirBullish, o, h, l, cl, rng, body, bodyRatio, upperWick,
			lowerWick, upperWickRatio, lowerWickRatio, atr, rangeATR, bodyExp,
			volumeRatio, vol)
		return
	}
	// Bearish: close<open, essentially no upper wick, close near low.
	if cl < o &&
		upperWick <= flatTol &&
		upperWickRatio <= cfg.FlatWickRatio &&
		displacementOK &&
		(cl-l)/rng <= cfg.CloseExtremeRatio {
		e.createMark(c, DirBearish, o, h, l, cl, rng, body, bodyRatio, upperWick,
			lowerWick, upperWickRatio, lowerWickRatio, atr, rangeATR, bodyExp,
			volumeRatio, vol)
	}
}

func (e *Engine) createMark(c *CandleInput, dir MarkDirection, o, h, l, cl, rng,
	body, bodyRatio, upperWick, lowerWick, upperWickRatio, lowerWickRatio, atr,
	rangeATR, bodyExp, volumeRatio float64, vol int64) {

	markPrice := o // open ≈ low (bull) / open ≈ high (bear) origin
	sc := e.scoreMark(dir, lowerWickRatio, upperWickRatio, rangeATR, bodyExp,
		bodyRatio, volumeRatio, false, false, false, false)
	if sc.Total < e.cfg.MinMarkQuality {
		return
	}
	now := time.Now()
	m := &DevilMark{
		ID:        uuid.NewString(),
		Symbol:    c.Symbol,
		Timeframe: c.Timeframe,
		Direction: dir,
		MarkPrice: markPrice,
		Open:      o, High: h, Low: l, Close: cl,
		Range:          rng,
		Body:           body,
		BodyRatio:      bodyRatio,
		UpperWick:      upperWick,
		LowerWick:      lowerWick,
		UpperWickRatio: upperWickRatio,
		LowerWickRatio: lowerWickRatio,
		ATR:            atr,
		DetectedATR:    atr,
		RangeATRRatio:  rangeATR,
		BodyExpansion:  bodyExp,
		Volume:         vol,
		VolumeRatio:    volumeRatio,
		Spread:         c.Spread,
		Digits:         c.Digits,
		TickSize:       e.cfg.TickSize,
		MarkQuality:    sc.Total,
		PriorityScore:  sc.Total,
		State:          StateActive,
		FeedSource:     c.FeedSource,
		Broker:         c.Broker,
		ServerID:       c.ServerID,
		ConfigVersion:  e.cfg.ConfigVersion,
		DetectedAt:     c.Time,
		UpdatedAt:      now,
	}
	e.marks[m.ID] = m
	e.marksCreated++
	e.persistMark(m)
	e.emit(DevilEvent{
		MarkID: m.ID, Symbol: m.Symbol, Timeframe: m.Timeframe,
		EventType: "DETECTED", StateFrom: StateDetected, StateTo: StateActive,
		Price: cl, MarkPrice: markPrice, ATR: atr, Spread: c.Spread,
		QualityScore: sc.Total, Metadata: map[string]interface{}{
			"direction": dir, "mark_quality": sc.Total,
		},
	})
}

// scoreMark computes the 0-100 mark quality score (Section 14).
func (e *Engine) scoreMark(dir MarkDirection, lowerWickRatio, upperWickRatio,
	rangeATR, bodyExp, bodyRatio, volumeRatio float64,
	fvg, bos, mss, choch bool) ScoreComponents {

	cfg := e.cfg
	flat := 1.0
	if dir == DirBullish {
		flat = clamp01(1.0 - lowerWickRatio/cfg.FlatWickRatio)
	} else {
		flat = clamp01(1.0 - upperWickRatio/cfg.FlatWickRatio)
	}
	disp := clamp01(rangeATR / cfg.MinRangeATR)
	bodyDom := clamp01(bodyRatio)
	vol := clamp01(volumeRatio / 2.0)
	structure := 0.0
	if bos {
		structure = 1.0
	}
	fvgS := 0.0
	if fvg {
		fvgS = 1.0
	}
	htf := 0.0
	if mss {
		htf = 1.0
	}
	sess := 0.5
	reg := 0.5

	sc := ScoreComponents{
		FlatEdge:     flat,
		Displacement: disp,
		BodyDom:      bodyDom,
		Volume:       vol,
		Structure:    structure,
		FVG:          fvgS,
		HTF:          htf,
		Session:      sess,
		Regime:       reg,
	}
	// weighted sum, normalized by used weights (max 100).
	weights := map[string]float64{
		"flat": 15, "disp": 20, "body": 10, "vol": cfg.VolumeWeight,
		"structure": 15, "fvg": 10, "htf": 10, "sess": 5, "reg": 5,
	}
	vals := map[string]float64{
		"flat": sc.FlatEdge, "disp": sc.Displacement, "body": sc.BodyDom,
		"vol": sc.Volume, "structure": sc.Structure, "fvg": sc.FVG,
		"htf": sc.HTF, "sess": sc.Session, "reg": sc.Regime,
	}
	var wsum, vsum float64
	for k, w := range weights {
		// only count components that were actually observed (struct/fvg/htf optional)
		if (k == "structure" && !bos) || (k == "fvg" && !fvg) || (k == "htf" && !mss) {
			continue
		}
		wsum += w
		vsum += w * vals[k]
	}
	if wsum > 0 {
		sc.Total = vsum / wsum * 100.0
	}
	return sc
}

func (e *Engine) advance(m *DevilMark, c *CandleInput, o, h, l, cl, atr float64) {
	cfg := e.cfg
	m.BarsSinceDetect++
	m.ATR = atr
	// Stable reference ATR captured at detection — prevents a single volatile
	// candle from distorting sweep/invalidation depth thresholds.
	atrRef := m.DetectedATR
	if atrRef <= 0 {
		atrRef = atr
	}
	dist := abs(cl-m.MarkPrice) / atrRef
	m.DistanceATR = dist

	// expiry
	if m.BarsSinceDetect > cfg.MarkExpiryBars {
		e.transition(m, StateExpired, cl, "EXPIRY")
		m.ExpiredAt = &[]time.Time{time.Now()}[0]
		e.persistMark(m)
		return
	}

	// touch
	if m.State != StateTouched && m.State != StateSwept && m.State != StateReclaiming &&
		m.State != StateReversalConfirmed && m.State != StateSignalEligible {
		touchTol := cfg.TouchToleranceATR * atrRef
		if abs(h-m.MarkPrice) <= touchTol || abs(l-m.MarkPrice) <= touchTol {
			m.FirstTouchAt = &[]time.Time{c.Time}[0]
			e.transition(m, StateTouched, cl, "TOUCH")
		}
	}

	// approach (only before touched)
	if m.State == StateActive {
		if dist <= cfg.ApproachDistanceATR {
			m.FirstApproachAt = &[]time.Time{c.Time}[0]
			e.transition(m, StateApproaching, cl, "APPROACH")
		}
	}

	// sweep + reclaim + reversal
	if m.Direction == DirBullish {
		// support: price should push BELOW mark then reclaim above
		if (m.State == StateTouched || m.State == StateApproaching) && l < m.MarkPrice-cfg.MinSweepDepthATR*atrRef {
			if m.FirstSweepAt == nil {
				m.FirstSweepAt = &[]time.Time{c.Time}[0]
				m.SweepLow = l
				m.SweepDepthATR = (m.MarkPrice - l) / atr
				e.transition(m, StateSwept, cl, "SWEEP")
			}
		}
		if m.State == StateSwept {
			if cl > m.MarkPrice {
				m.ReclaimAt = &[]time.Time{c.Time}[0]
				m.ReclaimStrength = (cl - m.MarkPrice) / atr
				e.transition(m, StateReclaiming, cl, "RECLAIM")
			} else if m.BarsSinceDetect > cfg.ReclaimMaxBars {
				e.transition(m, StateFailed, cl, "FAILED")
				m.ResolvedAt = &[]time.Time{time.Now()}[0]
			} else if cl < m.MarkPrice-cfg.MaxSweepDepthATR*atrRef {
				e.transition(m, StateInvalidated, cl, "INVALIDATION")
				m.InvalidatedAt = &[]time.Time{time.Now()}[0]
			}
		}
		if m.State == StateReclaiming {
			e.confirmReversal(m, c, o, h, l, cl, atr, true)
		}
	} else {
		// bearish: resistance sweep ABOVE mark then reject below
		if (m.State == StateTouched || m.State == StateApproaching) && h > m.MarkPrice+cfg.MinSweepDepthATR*atrRef {
			if m.FirstSweepAt == nil {
				m.FirstSweepAt = &[]time.Time{c.Time}[0]
				m.SweepHigh = h
				m.SweepDepthATR = (h - m.MarkPrice) / atr
				e.transition(m, StateSwept, cl, "SWEEP")
			}
		}
		if m.State == StateSwept {
			if cl < m.MarkPrice {
				m.ReclaimAt = &[]time.Time{c.Time}[0]
				m.ReclaimStrength = (m.MarkPrice - cl) / atr
				e.transition(m, StateReclaiming, cl, "REJECTION")
			} else if m.BarsSinceDetect > cfg.ReclaimMaxBars {
				e.transition(m, StateFailed, cl, "FAILED")
				m.ResolvedAt = &[]time.Time{time.Now()}[0]
			} else if cl > m.MarkPrice+cfg.MaxSweepDepthATR*atrRef {
				e.transition(m, StateInvalidated, cl, "INVALIDATION")
				m.InvalidatedAt = &[]time.Time{time.Now()}[0]
			}
		}
		if m.State == StateReclaiming {
			e.confirmReversal(m, c, o, h, l, cl, atr, false)
		}
	}
	e.persistMark(m)
}

func (e *Engine) confirmReversal(m *DevilMark, c *CandleInput, o, h, l, cl, atr float64, bullish bool) {
	cfg := e.cfg
	// reversal confirmation: a candle closing in the expected direction with
	// sufficient body dominance (Section 24). Optionally strengthens with VWAP/MSS.
	reversalBody := abs(cl-o) / max(h-l, 1e-9)
	if (bullish && cl > m.MarkPrice && cl > o && reversalBody >= cfg.ReversalBodyRatio) ||
		(!bullish && cl < m.MarkPrice && cl < o && reversalBody >= cfg.ReversalBodyRatio) {
		m.ReversalConfirmedAt = &[]time.Time{c.Time}[0]
		m.ReversalScore = clamp01(reversalBody) * 100.0
		m.CombinedScore = (m.MarkQuality*0.4 + m.ReversalScore*0.6)
		e.transition(m, StateReversalConfirmed, cl, "REVERSAL_CONFIRMATION")
		if m.CombinedScore >= e.cfg.MinSignalScore {
			e.transition(m, StateSignalEligible, cl, "SIGNAL_ELIGIBLE")
		}
		m.ResolvedAt = &[]time.Time{time.Now()}[0]
	}
}

func (e *Engine) transition(m *DevilMark, to MarkState, price float64, etype string) {
	from := m.State
	if from == to {
		return
	}
	m.State = to
	m.UpdatedAt = time.Now()
	e.emit(DevilEvent{
		MarkID: m.ID, Symbol: m.Symbol, Timeframe: m.Timeframe,
		EventType: etype, StateFrom: from, StateTo: to,
		Price: price, MarkPrice: m.MarkPrice, DistanceATR: m.DistanceATR,
		ATR: m.ATR, Spread: m.Spread, QualityScore: m.MarkQuality,
		ReversalScore: m.ReversalScore, Metadata: map[string]interface{}{
			"direction": m.Direction, "combined_score": m.CombinedScore,
		},
	})
}

func (e *Engine) emit(ev DevilEvent) {
	if e.onEvent != nil {
		e.onEvent(ev)
	}
}

// EngineStats reports liveness/observability counters.
type EngineStats struct {
	CandlesProcessed int64     `json:"candles_processed"`
	MarksCreated     int64     `json:"marks_created"`
	Active           int       `json:"active_marks"`
	LastCandleTime   time.Time `json:"last_candle_time"`
	SymbolsSeen      []string  `json:"symbols_seen"`
}

// Stats returns a snapshot of engine activity.
func (e *Engine) Stats() EngineStats {
	e.mu.RLock()
	defer e.mu.RUnlock()
	syms := make([]string, 0, len(e.symbolsSeen))
	for s := range e.symbolsSeen {
		syms = append(syms, s)
	}
	return EngineStats{
		CandlesProcessed: e.candlesProcessed,
		MarksCreated:     e.marksCreated,
		Active:           len(e.marks),
		LastCandleTime:   e.lastCandleTime,
		SymbolsSeen:      syms,
	}
}

// ActiveMarks returns non-terminal marks (for API/UI).
func (e *Engine) ActiveMarks() []*DevilMark {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := make([]*DevilMark, 0, len(e.marks))
	for _, m := range e.marks {
		if !isTerminal(m.State) {
			cp := *m
			out = append(out, &cp)
		}
	}
	return out
}

// AllMarks returns every tracked mark.
func (e *Engine) AllMarks() []*DevilMark {
	e.mu.RLock()
	defer e.mu.RUnlock()
	out := make([]*DevilMark, 0, len(e.marks))
	for _, m := range e.marks {
		cp := *m
		out = append(out, &cp)
	}
	return out
}

func isTerminal(s MarkState) bool {
	switch s {
	case StateInvalidated, StateExpired, StateFailed, StateMitigated:
		return true
	}
	return false
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
func max3(a, b, c float64) float64 {
	return max(max(a, b), c)
}
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
