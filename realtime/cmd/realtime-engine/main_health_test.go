package main

import "testing"

// dataFeedOutage must be true only when an EA-direct device has been active
// AND the snapshot feed is critically stale/missing. No active device is NOT
// an outage (there is no data source to be silent); an active device with
// fresh data is NOT an outage; a lone tick without snapshots must not
// suppress the outage.
func TestDataFeedOutage(t *testing.T) {
	cases := []struct {
		name         string
		critical     bool
		deviceActive bool
		want         bool
	}{
		{"no device active, critical", true, false, false},
		{"no device active, fresh", false, false, false},
		{"device active, critical", true, true, true},
		{"device active, fresh", false, true, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := dataFeedOutage(c.critical, c.deviceActive); got != c.want {
				t.Fatalf("dataFeedOutage(%v,%v) = %v, want %v", c.critical, c.deviceActive, got, c.want)
			}
		})
	}
}
