package main

import "testing"

// dataFeedOutage must be true only when an agent is connected AND the snapshot
// feed is critically stale/missing. No agents (disconnected) is NOT an outage
// (there is no data source to be silent); a connected agent with fresh data is
// NOT an outage; a lone tick without snapshots must not suppress the outage.
func TestDataFeedOutage(t *testing.T) {
	cases := []struct {
		name     string
		critical bool
		agents   int
		want     bool
	}{
		{"no agents, critical", true, 0, false},
		{"no agents, fresh", false, 0, false},
		{"agents, critical", true, 3, true},
		{"agents, fresh", false, 3, false},
		{"single data agent, critical", true, 1, true},
		{"single data agent, fresh", false, 1, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := dataFeedOutage(c.critical, c.agents); got != c.want {
				t.Fatalf("dataFeedOutage(%v,%d) = %v, want %v", c.critical, c.agents, got, c.want)
			}
		})
	}
}
