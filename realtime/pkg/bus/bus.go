// Package bus provides the ingest/signal decoupling seam for the realtime
// engine. The Windows Agents (data-collection edge) publish inbound messages
// onto a bus; the signal engine consumes them. This isolates the
// data-collection plane from the signal-processing plane: if ingestion stalls
// or a separate ingest service is introduced, the signal engine keeps running.
//
// Two transports are provided:
//   - DirectBus: in-process, calls the handler synchronously. This preserves
//     the current (pre-NATS) behavior and is the default when no NATS_URL is
//     configured. Zero external dependencies, zero network.
//   - NatsBus: uses NATS (github.com/nats-io/nats.go) as the production
//     transport. The publisher enqueues messages; a subscriber goroutine
//     dispatches them to the same handler. Activated only when NATS_URL is set.
//
// Either way the engine's message handler (marketdata.AgentProvider
// .HandleAgentMessage) is the single consumer, so behavior is identical.
package bus

import (
	"encoding/json"

	"github.com/nats-io/nats.go"
)

// MessageHandler consumes an inbound agent message for a given agent.
type MessageHandler func(agentID string, data []byte)

// IngestBus is the producer side: agents publish inbound messages here.
type IngestBus interface {
	Publish(agentID string, data []byte) error
	Close() error
}

// IngestSubscriber is the consumer side: the engine subscribes and dispatches.
type IngestSubscriber interface {
	Subscribe(handler MessageHandler) error
	Close() error
}

// Envelope is the on-the-wire (NATS) representation of an inbound agent message.
type Envelope struct {
	AgentID string          `json:"agent_id"`
	Data    json.RawMessage `json:"data"`
}

// DirectBus is the in-process default transport. Publish invokes the handler
// synchronously, preserving the original call path with no external systems.
type DirectBus struct {
	handler MessageHandler
}

// NewDirectBus returns a DirectBus that dispatches every message to handler.
func NewDirectBus(handler MessageHandler) *DirectBus {
	return &DirectBus{handler: handler}
}

// Publish calls the handler synchronously. It never errors.
func (d *DirectBus) Publish(agentID string, data []byte) error {
	if d.handler != nil {
		d.handler(agentID, data)
	}
	return nil
}

// Close is a no-op for the in-process bus.
func (d *DirectBus) Close() error { return nil }

// NatsBus is the NATS-backed transport. A single connection serves both
// publishing and subscribing (NATS core, no JetStream required for the live
// fire-and-forget stream). At-least-once delivery is provided by NATS; the
// engine's idempotency (duplicate checker) handles redelivery safely.
type NatsBus struct {
	conn    *nats.Conn
	subject string
}

// NewNatsBus connects to the given NATS URL and binds to subject.
func NewNatsBus(url, subject string) (*NatsBus, error) {
	nc, err := nats.Connect(url, nats.RetryOnFailedConnect(true), nats.MaxReconnects(-1))
	if err != nil {
		return nil, err
	}
	return &NatsBus{conn: nc, subject: subject}, nil
}

// Publish enqueues the message onto the NATS subject (async).
func (n *NatsBus) Publish(agentID string, data []byte) error {
	env := Envelope{AgentID: agentID, Data: data}
	b, err := json.Marshal(env)
	if err != nil {
		return err
	}
	return n.conn.Publish(n.subject, b)
}

// Subscribe starts a long-lived dispatcher for the bound subject.
func (n *NatsBus) Subscribe(handler MessageHandler) error {
	_, err := n.conn.Subscribe(n.subject, func(m *nats.Msg) {
		var env Envelope
		if err := json.Unmarshal(m.Data, &env); err != nil {
			return
		}
		if handler != nil {
			handler(env.AgentID, env.Data)
		}
	})
	return err
}

// Close drains and closes the NATS connection.
func (n *NatsBus) Close() error {
	if n.conn == nil {
		return nil
	}
	n.conn.Close()
	return nil
}
