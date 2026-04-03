package app

import "testing"

func TestSttInactivityTimeoutSecondsFromCommitTimeoutMS(t *testing.T) {
	tests := []struct {
		name    string
		inMS    int
		wantSec int
	}{
		{name: "off when non positive", inMS: 0, wantSec: 0},
		{name: "off when negative", inMS: -100, wantSec: 0},
		{name: "round up to one second", inMS: 1, wantSec: 1},
		{name: "round up partial second", inMS: 1500, wantSec: 2},
		{name: "exact second", inMS: 5000, wantSec: 5},
		{name: "180s config maps directly", inMS: 180_000, wantSec: 180},
		{name: "clamp to max", inMS: 999_999, wantSec: 180},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := sttInactivityTimeoutSecondsFromCommitTimeoutMS(tc.inMS)
			if got != tc.wantSec {
				t.Fatalf("got %d, want %d", got, tc.wantSec)
			}
		})
	}
}
