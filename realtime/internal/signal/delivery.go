// Package signal — Signal delivery, sequence tracking, and replay/resume.
// SOW Sections 19, 29, 47: Signal Sequence Resume + Idempotent Execution.
package signal

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/predictatrade/realtime/internal/types"
)

// DeliveryState represents the signal delivery state machine.
type DeliveryState string

const (
	DeliveryGenerated    DeliveryState = "GENERATED"
	DeliveryGated        DeliveryState = "GATED"
	DeliveryQueued       DeliveryState = "QUEUED"
	DeliverySent         DeliveryState = "SENT"
	DeliveryDelivered    DeliveryState = "DELIVERED"
	DeliveryAcknowledged DeliveryState = "ACKNOWLEDGED"
	DeliveryExecuting    DeliveryState = "EXECUTING"
	DeliveryExecuted     DeliveryState = "EXECUTED"
	DeliveryRejected     DeliveryState = "REJECTED"
	DeliveryFailed       DeliveryState = "FAILED"
	DeliveryExpired      DeliveryState = "EXPIRED"
	DeliveryCancelled    DeliveryState = "CANCELLED"
)

// SignalDelivery tracks per-device signal delivery state.
type SignalDelivery struct {
	ID              string          `json:"id"`
	SignalID        string          `json:"signal_id"`
	DeviceID        string          `json:"device_id"`
	LicenseID       string          `json:"license_id"`
	AccountID       string          `json:"account_id"`
	TerminalID      string          `json:"terminal_id"`
	SequenceNumber  int64           `json:"sequence_number"`
	DeliveryState   DeliveryState   `json:"delivery_state"`
	SentAt          *time.Time      `json:"sent_at,omitempty"`
	DeliveredAt     *time.Time      `json:"delivered_at,omitempty"`
	AcknowledgedAt  *time.Time      `json:"acknowledged_at,omitempty"`
	ExecutedAt      *time.Time      `json:"executed_at,omitempty"`
	BrokerTicket    string          `json:"broker_ticket,omitempty"`
	ExecutionResult json.RawMessage `json:"execution_result,omitempty"`
	Slippage        float64         `json:"slippage,omitempty"`
	TotalLatencyMs  int             `json:"total_latency_ms,omitempty"`
	SendAttempts    int             `json:"send_attempts"`
	ReplayCount     int             `json:"replay_count"`
	FailureReason   string          `json:"failure_reason,omitempty"`
	LastError       string          `json:"last_error,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// DeliveryManager manages signal delivery state, sequence tracking, and replay.
type DeliveryManager struct {
	db        *sql.DB
	mu        sync.Mutex
	sequences map[string]int64 // device_id → next sequence
}

// NewDeliveryManager creates a delivery manager with optional DB persistence.
func NewDeliveryManager(db *sql.DB) *DeliveryManager {
	return &DeliveryManager{
		db:        db,
		sequences: make(map[string]int64),
	}
}

// RecordDelivery creates a signal delivery record when a signal is sent to a device.
func (dm *DeliveryManager) RecordDelivery(ctx context.Context, signal *types.Signal, deviceID, licenseID, accountID string) (*SignalDelivery, error) {
	dm.mu.Lock()
	seq := dm.sequences[deviceID]
	dm.sequences[deviceID] = seq + 1
	dm.mu.Unlock()

	delivery := &SignalDelivery{
		SignalID:       signal.ID,
		DeviceID:       deviceID,
		LicenseID:      licenseID,
		AccountID:      accountID,
		SequenceNumber: seq,
		DeliveryState:  DeliverySent,
		SendAttempts:   1,
		// v1.28: timestamps MUST be UTC instants — .UTC() guarantees the
		// RFC3339 payload carries Z (or +00:00) so every consumer (EA
		// freshness gates, dashboards, broker-time conversions) treats them
		// unambiguously. The Postgres write path uses now() (timestamptz) and
		// is unaffected.
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	now := time.Now().UTC()
	delivery.SentAt = &now

	if dm.db == nil {
		return delivery, nil
	}

	// Empty license/account IDs are the normal case for agent-delivered signals
	// (no per-signal license/account binding). Insert them as NULL rather than
	// the empty string, which Postgres rejects as a UUID and would fail every
	// delivery-record write.
	var licID, accID interface{}
	if licenseID == "" {
		licID = nil
	} else {
		licID = licenseID
	}
	if accountID == "" {
		accID = nil
	} else {
		accID = accountID
	}

	_, err := dm.db.ExecContext(ctx, `
		INSERT INTO trading.signal_deliveries
		(signal_id, device_id, license_id, account_id, sequence_number, delivery_state, sent_at, send_attempts, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, 'SENT', now(), 1, now(), now())
		ON CONFLICT (signal_id, device_id) DO UPDATE SET send_attempts = signal_deliveries.send_attempts + 1, updated_at = now()
	`, signal.ID, deviceID, licID, accID, seq)
	if err != nil {
		return nil, fmt.Errorf("record delivery: %w", err)
	}

	// Update sequence tracker
	_, _ = dm.db.ExecContext(ctx, `
		INSERT INTO trading.signal_sequences (device_id, last_sent_sequence, last_heartbeat_at, updated_at)
		VALUES ($1, $2, now(), now())
		ON CONFLICT (device_id) DO UPDATE SET last_sent_sequence = $2, updated_at = now()
	`, deviceID, seq+1)

	return delivery, nil
}

// AcknowledgeSignal processes a signal ACK from a device.
// Idempotent: duplicate ACKs don't cause errors.
func (dm *DeliveryManager) AcknowledgeSignal(ctx context.Context, signalID, deviceID string, brokerTicket string, execResult json.RawMessage) error {
	if dm.db == nil {
		return nil
	}

	_, err := dm.db.ExecContext(ctx, `
		UPDATE trading.signal_deliveries 
		SET delivery_state = 'ACKNOWLEDGED', acknowledged_at = now(), broker_ticket = $3, execution_result = $4, updated_at = now()
		WHERE signal_id = $1 AND device_id = $2 AND delivery_state NOT IN ('EXECUTED', 'ACKNOWLEDGED')
	`, signalID, deviceID, brokerTicket, execResult)
	return err
}

// ResumeFromSequence returns signals that were sent but not acknowledged,
// for replay on reconnect. Only returns signals that are still valid (not expired).
func (dm *DeliveryManager) ResumeFromSequence(ctx context.Context, deviceID string, lastAckedSeq int64) ([]SignalDelivery, error) {
	if dm.db == nil {
		return nil, nil
	}

	rows, err := dm.db.QueryContext(ctx, `
		SELECT sd.signal_id, sd.sequence_number, sd.delivery_state, sd.sent_at,
		       s.direction, s.entry_price, s.stop_loss, s.tp1, s.tp2, s.tp3,
		       s.strategy_id, s.status, s.expires_at
		FROM trading.signal_deliveries sd
		JOIN trading.signals s ON s.id = sd.signal_id
		WHERE sd.device_id = $1 
		  AND sd.sequence_number > $2
		  AND sd.delivery_state IN ('SENT', 'DELIVERED', 'QUEUED')
		  AND s.expires_at > now()
		  AND s.direction IN ('BUY', 'SELL')
		  AND s.status IN ('CONFIRMED', 'ACTIVE')
		ORDER BY sd.sequence_number ASC
		LIMIT 10
	`, deviceID, lastAckedSeq)
	if err != nil {
		return nil, fmt.Errorf("resume query: %w", err)
	}
	defer rows.Close()

	var deliveries []SignalDelivery
	for rows.Next() {
		var d SignalDelivery
		var sentAt sql.NullTime
		if err := rows.Scan(&d.SignalID, &d.SequenceNumber, &d.DeliveryState, &sentAt); err != nil {
			continue
		}
		if sentAt.Valid {
			t := sentAt.Time
			d.SentAt = &t
		}
		d.ReplayCount++
		deliveries = append(deliveries, d)
	}

	// Mark as replayed
	for _, d := range deliveries {
		_, _ = dm.db.ExecContext(ctx, `
			UPDATE trading.signal_deliveries SET replay_count = replay_count + 1, updated_at = now()
			WHERE signal_id = $1 AND device_id = $2
		`, d.SignalID, deviceID)
	}

	return deliveries, nil
}

// MarkExpired marks signals that have passed their expiry as EXPIRED.
func (dm *DeliveryManager) MarkExpired(ctx context.Context) (int64, error) {
	if dm.db == nil {
		return 0, nil
	}
	res, err := dm.db.ExecContext(ctx, `
		UPDATE trading.signal_deliveries SET delivery_state = 'EXPIRED', updated_at = now()
		WHERE delivery_state IN ('SENT', 'DELIVERED', 'QUEUED')
		  AND signal_id IN (SELECT id FROM trading.signals WHERE expires_at < now())
	`)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// IsAlreadyExecuted checks idempotency: has this signal already been executed for this device?
func (dm *DeliveryManager) IsAlreadyExecuted(ctx context.Context, signalID, deviceID string) (bool, error) {
	if dm.db == nil {
		return false, nil
	}
	var count int
	err := dm.db.QueryRowContext(ctx, `
		SELECT count(*) FROM trading.signal_deliveries 
		WHERE signal_id = $1 AND device_id = $2 AND delivery_state IN ('EXECUTED', 'ACKNOWLEDGED')
	`, signalID, deviceID).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
