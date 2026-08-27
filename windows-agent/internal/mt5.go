package agent

import (
	"encoding/json"
)

// MT5Tick represents real tick data read from MT5 terminal.
type MT5Tick struct {
	Type      string    `json:"type"`       // always "TICK"
	Symbol    string    `json:"symbol"`     // "XAUUSD" or broker symbol
	Bid       float64   `json:"bid"`        // real MT5 bid
	Ask       float64   `json:"ask"`        // real MT5 ask
	Volume    int64     `json:"volume"`     // real MT5 tick volume
	Timestamp string    `json:"timestamp"`  // MT5 server time
	Source    string    `json:"source"`     // "MT5" or "MT4"
	Broker    string    `json:"broker"`     // broker name
	Account   string    `json:"account"`    // MT5 account number
	// BrokerOffset is the broker UTC offset in hours, reported live by the
	// Master Node (TimeGMTOffset). Forwarded verbatim so the engine can align
	// candles to broker session TF rather than UTC.
	BrokerOffset int `json:"broker_offset"`
}

// MT5Heartbeat is sent periodically to keep connection alive.
type MT5Heartbeat struct {
	Type      string    `json:"type"`       // "HEARTBEAT"
	AgentID   string    `json:"agentId"`
	Timestamp string `json:"timestamp"`
	MT5Connected bool  `json:"mt5Connected"`
}

// MarshalTick serializes an MT5 tick for sending via WebSocket.
func MarshalTick(t MT5Tick) ([]byte, error) {
	t.Type = "TICK"
	return json.Marshal(t)
}

// MarshalHeartbeat serializes a heartbeat message.
func MarshalHeartbeat(h MT5Heartbeat) ([]byte, error) {
	h.Type = "HEARTBEAT"
	return json.Marshal(h)
}
