package client

import "testing"

func TestFormatBitRate(t *testing.T) {
	tests := []struct {
		name           string
		bytesPerSecond uint64
		want           string
	}{
		{name: "zero", bytesPerSecond: 0, want: "0 bps"},
		{name: "bits per second", bytesPerSecond: 100, want: "800 bps"},
		{name: "kilobits per second", bytesPerSecond: 1_000, want: "8.00 Kbps"},
		{name: "megabits per second", bytesPerSecond: 1_000_000, want: "8.00 Mbps"},
		{name: "gigabits per second", bytesPerSecond: 1_000_000_000, want: "8.00 Gbps"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := formatBitRate(test.bytesPerSecond); got != test.want {
				t.Fatalf("formatBitRate(%d) = %q, want %q", test.bytesPerSecond, got, test.want)
			}
		})
	}
}

func TestCounterDelta(t *testing.T) {
	tests := []struct {
		name              string
		current, previous uint64
		want              uint64
	}{
		{name: "increase", current: 125, previous: 100, want: 25},
		{name: "unchanged", current: 100, previous: 100, want: 0},
		{name: "counter reset", current: 10, previous: 100, want: 0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := counterDelta(test.current, test.previous); got != test.want {
				t.Fatalf("counterDelta(%d, %d) = %d, want %d", test.current, test.previous, got, test.want)
			}
		})
	}
}
