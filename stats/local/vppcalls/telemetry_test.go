package vppcalls

import (
	"fmt"
	"strings"
	"testing"

	govppapi "go.fd.io/govpp/api"
)

type errorStatsProvider struct {
	errors []govppapi.ErrorCounter
	nodes  []govppapi.NodeCounters
}

func (*errorStatsProvider) GetSystemStats(*govppapi.SystemStats) error { return nil }
func (p *errorStatsProvider) GetNodeStats(stats *govppapi.NodeStats) error {
	stats.Nodes = p.nodes
	return nil
}
func (*errorStatsProvider) GetInterfaceStats(*govppapi.InterfaceStats) error { return nil }
func (p *errorStatsProvider) GetErrorStats(stats *govppapi.ErrorStats) error {
	stats.Errors = p.errors
	return nil
}
func (*errorStatsProvider) GetBufferStats(*govppapi.BufferStats) error { return nil }
func (*errorStatsProvider) GetMemoryStats(*govppapi.MemoryStats) error { return nil }

func TestParseNodeCountersLargeOutput(t *testing.T) {
	const entries = 2000
	var output strings.Builder
	output.WriteString("\n Count Node Reason Severity\n")
	for i := 0; i < entries; i++ {
		fmt.Fprintf(&output, " %d node-%d packet length > MTU (path: ip4/udp); dropped error\n", i+1, i)
	}

	counters, err := parseNodeCounters(output.String())
	if err != nil {
		t.Fatalf("parseNodeCounters() error = %v", err)
	}
	if len(counters) != entries {
		t.Fatalf("parseNodeCounters() returned %d counters, want %d", len(counters), entries)
	}

	last := counters[len(counters)-1]
	if last.Node != "node-1999" || last.Reason != "packet length > MTU (path: ip4/udp); dropped" || last.Severity != "error" {
		t.Fatalf("last counter parsed incorrectly: %+v", last)
	}
}

func TestParseNodeCountersOldFormat(t *testing.T) {
	output := "Count Node Reason\n 12 ethernet-input malformed packet: size/offset mismatch\n"
	counters, err := parseNodeCounters(output)
	if err != nil {
		t.Fatalf("parseNodeCounters() error = %v", err)
	}
	if len(counters) != 1 {
		t.Fatalf("parseNodeCounters() returned %d counters, want 1", len(counters))
	}
	if got := counters[0]; got.Reason != "malformed packet: size/offset mismatch" || got.Severity != "unknown" {
		t.Fatalf("counter parsed incorrectly: %+v", got)
	}
}

func TestParseNodeCountersRejectsEmptyResponse(t *testing.T) {
	if _, err := parseNodeCounters("\n\t\n"); err == nil {
		t.Fatal("parseNodeCounters() accepted an empty response")
	}
}

func TestGetNodeCountersFromStats(t *testing.T) {
	handler := &TelemetryHandler{sp: &errorStatsProvider{errors: []govppapi.ErrorCounter{{
		CounterName: "ip4-input/header checksum failure",
		Values:      []uint64{2, 3},
	}}}}

	counters, err := handler.getNodeCountersFromStats()
	if err != nil {
		t.Fatalf("getNodeCountersFromStats() error = %v", err)
	}
	if len(counters) != 1 {
		t.Fatalf("getNodeCountersFromStats() returned %d counters, want 1", len(counters))
	}
	if got := counters[0]; got.Count != 5 || got.Node != "ip4-input" || got.Reason != "header checksum failure" || got.Severity != "unknown" {
		t.Fatalf("fallback counter converted incorrectly: %+v", got)
	}
}

func TestParseRuntimeInfoIncludesInterfaceNodes(t *testing.T) {
	output := `Time 1.0, 10 sec internal node vector rate 2.0 loops/sec 3.0
  vector rates in 4.0, out 5.0, drop 6.0, punt 7.0
             Name                 State         Calls          Vectors        Suspends         Clocks       Vectors/Call
local0-output                     active             1                2               0          1.00            2.00
GigabitEthernet0/8/0-output       active             3                6               0          2.00            2.00
GigabitEthernet0/9/0-output       active             4                8               0          3.00            2.00
`

	runtimeInfo, err := parseRuntimeInfo(output)
	if err != nil {
		t.Fatalf("parseRuntimeInfo() error = %v", err)
	}
	if len(runtimeInfo.Threads) != 1 || len(runtimeInfo.Threads[0].Items) != 3 {
		t.Fatalf("parseRuntimeInfo() returned incomplete data: %+v", runtimeInfo.Threads)
	}
	if got := runtimeInfo.Threads[0].Items[2].Name; got != "GigabitEthernet0/9/0-output" {
		t.Fatalf("last runtime node = %q, want interface node", got)
	}
}

func TestGetRuntimeInfoFromStats(t *testing.T) {
	handler := &TelemetryHandler{sp: &errorStatsProvider{nodes: []govppapi.NodeCounters{{
		NodeIndex: 9,
		NodeName:  "GigabitEthernet0/8/0-output",
		Calls:     4,
		Vectors:   10,
		Suspends:  2,
		Clocks:    12,
	}}}}

	runtimeInfo, err := handler.getRuntimeInfoFromStats()
	if err != nil {
		t.Fatalf("getRuntimeInfoFromStats() error = %v", err)
	}
	item := runtimeInfo.Threads[0].Items[0]
	if item.Name != "GigabitEthernet0/8/0-output" || item.VectorsPerCall != 2.5 || item.State != "unknown" {
		t.Fatalf("fallback runtime node converted incorrectly: %+v", item)
	}
}
