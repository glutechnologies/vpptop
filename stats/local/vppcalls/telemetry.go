/*
 * Copyright (c) 2020 Cisco and/or its affiliates.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at:
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package vppcalls

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/glutechnologies/vpptop/stats/api"
	govppapi "go.fd.io/govpp/api"
	"go.fd.io/govpp/binapi/vlib"
)

// TelemetryVppAPI defines telemetry-specific methods
type TelemetryVppAPI interface {
	GetInterfaceStats(context.Context) (*govppapi.InterfaceStats, error)
	GetNodeCounters(context.Context) (*api.NodeCounterInfo, error)
	GetRuntimeInfo(context.Context) (*api.RuntimeInfo, error)
	GetThreads(context.Context) ([]api.ThreadData, error)
}

// TelemetryHandler implements TelemetryVppAPI
type TelemetryHandler struct {
	sp govppapi.StatsProvider
	ch govppapi.Channel
}

// NewTelemetryHandler returns a new instance of the TelemetryVppAPI
func NewTelemetryHandler(ch govppapi.Channel, sp govppapi.StatsProvider) TelemetryVppAPI {
	return &TelemetryHandler{
		ch: ch,
		sp: sp,
	}
}

// Regular expressions used to parse telemetry output
var (
	// 'show runtime'
	runtimeRe = regexp.MustCompile(`(?:Thread ([0-9]+) (\S+)[^\n]*\n)?Time ([0-9\.e-]+), ([0-9]+) sec internal node vector rate ([0-9\.e-]+) loops/sec ([0-9\.e-]+)\s+` +
		`vector rates in ([0-9\.e-]+), out ([0-9\.e-]+), drop ([0-9\.e-]+), punt ([0-9\.e-]+)\n` +
		`\s+Name\s+State\s+Calls\s+Vectors\s+Suspends\s+Clocks\s+Vectors/Call\s+` +
		`((?:\S+\s+\w+(?:[ -]\w+)*\s+\d+\s+\d+\s+\d+\s+[0-9\.e-]+\s+[0-9\.e-]+\s+)+)`)
	// 'show runtime' items
	runtimeItemsRe         = regexp.MustCompile(`(\S+)\s+(\w+(?:[ -]\w+)*)\s+(\d+)\s+(\d+)\s+(\d+)\s+([0-9.e-]+)\s+([0-9.e-]+)\s+`)
	errorNameLikeMemifRe   = regexp.MustCompile(`^[A-Za-z0-9-]+([0-9]+/[0-9]+|pg/stream)`)
	errorNameLikeGigabitRe = regexp.MustCompile(`^[A-Za-z0-9]+[0-9a-f]+(/[0-9a-f]+){2}`)
)

func splitErrorName(name string) (node, reason string) {
	parts := strings.Split(name, "/")
	switch len(parts) {
	case 1:
		return parts[0], ""
	case 2:
		return parts[0], parts[1]
	case 3:
		if strings.Contains(parts[1], " ") {
			return parts[0], strings.Join(parts[1:], "/")
		}
		if errorNameLikeMemifRe.MatchString(name) {
			return strings.Join(parts[:2], "/"), parts[2]
		}
	default:
		if strings.Contains(parts[2], " ") {
			return strings.Join(parts[:2], "/"), strings.Join(parts[2:], "/")
		}
		if errorNameLikeGigabitRe.MatchString(name) {
			return strings.Join(parts[:3], "/"), strings.Join(parts[3:], "/")
		}
	}
	return strings.Join(parts[:len(parts)-1], "/"), parts[len(parts)-1]
}

func (h *TelemetryHandler) GetInterfaceStats(context.Context) (*govppapi.InterfaceStats, error) {
	ifStats := &govppapi.InterfaceStats{}
	err := h.sp.GetInterfaceStats(ifStats)
	if err != nil {
		return nil, err
	}
	return ifStats, nil
}

func (h *TelemetryHandler) GetNodeCounters(ctx context.Context) (*api.NodeCounterInfo, error) {
	data := new(vlib.CliInbandReply)
	err := h.ch.SendRequest(&vlib.CliInband{Cmd: "show node counters"}).ReceiveReply(data)
	if err == nil {
		if counters, parseErr := parseNodeCounters(data.Reply); parseErr == nil {
			return &api.NodeCounterInfo{Counters: counters}, nil
		} else {
			err = parseErr
		}
	} else {
		err = fmt.Errorf("VPP CLI command \"show node counters\" failed: %w", err)
	}

	// cli_inband returns the complete command output in one binary API message.
	// Large error lists can exceed that path's capacity, so use the stats segment
	// as a fallback. Severity is not exposed there, but the counters remain useful.
	counters, statsErr := h.getNodeCountersFromStats()
	if statsErr != nil {
		return nil, fmt.Errorf("%v; stats API fallback failed: %w", err, statsErr)
	}
	return &api.NodeCounterInfo{Counters: counters}, nil
}

func parseNodeCounters(reply string) ([]api.NodeCounter, error) {
	var counters []api.NodeCounter
	headerSeen := false
	hasSeverity := false

	for _, line := range strings.Split(reply, "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if !headerSeen {
			fields := strings.Fields(line)
			if (len(fields) == 3 || len(fields) == 4) && fields[0] == "Count" {
				headerSeen = true
				hasSeverity = len(fields) == 4
				continue
			}
			return nil, fmt.Errorf("invalid header for `show node counters` received: %q", line)
		}

		fields := strings.Fields(line)
		minimumFields := 3
		if hasSeverity {
			minimumFields = 4
		}
		if len(fields) < minimumFields {
			return nil, fmt.Errorf("`show node counters` parsing failed line: %q", line)
		}

		severity := "unknown"
		reasonEnd := len(fields)
		if hasSeverity {
			severity = fields[len(fields)-1]
			reasonEnd--
		}
		counters = append(counters, api.NodeCounter{
			Count:    uint64(strToFloat64(fields[0])),
			Node:     fields[1],
			Reason:   strings.Join(fields[2:reasonEnd], " "),
			Severity: severity,
		})
	}
	if !headerSeen {
		return nil, fmt.Errorf("invalid empty response for `show node counters`")
	}
	return counters, nil
}

func (h *TelemetryHandler) getNodeCountersFromStats() ([]api.NodeCounter, error) {
	errorStats := new(govppapi.ErrorStats)
	if err := h.sp.GetErrorStats(errorStats); err != nil {
		return nil, err
	}

	counters := make([]api.NodeCounter, 0, len(errorStats.Errors))
	for _, counter := range errorStats.Errors {
		node, reason := splitErrorName(counter.CounterName)
		var count uint64
		for _, workerCount := range counter.Values {
			count += workerCount
		}
		counters = append(counters, api.NodeCounter{
			Count:    count,
			Node:     node,
			Reason:   reason,
			Severity: "unknown",
		})
	}
	return counters, nil
}

func (h *TelemetryHandler) GetRuntimeInfo(ctx context.Context) (*api.RuntimeInfo, error) {
	// Runtime output retains the thread associated with each node. The stats
	// segment aggregates workers and therefore cannot populate the Worker column.
	cliResp := new(vlib.CliInbandReply)
	err := h.ch.SendRequest(&vlib.CliInband{Cmd: "show runtime"}).ReceiveReply(cliResp)
	if err == nil {
		if cliRuntimeInfo, parseErr := parseRuntimeInfo(cliResp.Reply); parseErr == nil {
			return cliRuntimeInfo, nil
		} else {
			err = parseErr
		}
	} else {
		err = fmt.Errorf("VPP CLI command \"show runtime\" failed: %w", err)
	}

	// Fall back to aggregated stats if the CLI request or parser is unavailable.
	runtimeInfo, statsErr := h.getRuntimeInfoFromStats()
	if statsErr != nil {
		return nil, fmt.Errorf("%v; stats API fallback failed: %w", err, statsErr)
	}
	return runtimeInfo, nil
}

func parseRuntimeInfo(reply string) (*api.RuntimeInfo, error) {
	threadMatches := runtimeRe.FindAllStringSubmatch(reply, -1)
	if len(threadMatches) == 0 && reply != "" {
		return nil, fmt.Errorf("invalid command: %q, thread matches: %d", reply, len(threadMatches))
	}

	var threads []api.RuntimeThread
	for _, matches := range threadMatches {
		fields := matches[1:]
		if len(fields) != 11 {
			return nil, fmt.Errorf("invalid runtime data for thread (len=%v): %q", len(fields), matches[0])
		}
		thread := api.RuntimeThread{
			ID:                 uint(strToFloat64(fields[0])),
			Name:               fields[1],
			Time:               strToFloat64(fields[2]),
			AvgVectorsPerNode:  strToFloat64(fields[3]),
			LastMainLoops:      uint64(strToFloat64(fields[4])),
			VectorsPerMainLoop: strToFloat64(fields[5]),
			VectorRatesIn:      strToFloat64(fields[6]),
			VectorRatesOut:     strToFloat64(fields[7]),
			VectorRatesDrop:    strToFloat64(fields[8]),
			VectorRatesPunt:    strToFloat64(fields[9]),
		}

		itemMatches := runtimeItemsRe.FindAllStringSubmatch(fields[10], -1)
		for _, matches := range itemMatches {
			fields := matches[1:]
			if len(fields) != 7 {
				return nil, fmt.Errorf("invalid runtime data for thread item: %q", matches[0])
			}
			thread.Items = append(thread.Items, api.RuntimeItem{
				Name:           fields[0],
				State:          strings.Replace(fields[1], " ", "-", -1),
				Calls:          uint64(strToFloat64(fields[2])),
				Vectors:        uint64(strToFloat64(fields[3])),
				Suspends:       uint64(strToFloat64(fields[4])),
				Clocks:         strToFloat64(fields[5]),
				VectorsPerCall: strToFloat64(fields[6]),
			})
		}

		threads = append(threads, thread)
	}

	return &api.RuntimeInfo{
		Threads: threads,
	}, nil
}

func (h *TelemetryHandler) getRuntimeInfoFromStats() (*api.RuntimeInfo, error) {
	nodeStats := new(govppapi.NodeStats)
	if err := h.sp.GetNodeStats(nodeStats); err != nil {
		return nil, err
	}
	if len(nodeStats.Nodes) == 0 {
		return nil, fmt.Errorf("node stats are unavailable")
	}

	thread := api.RuntimeThread{Name: "ALL"}
	thread.Items = make([]api.RuntimeItem, 0, len(nodeStats.Nodes))
	for _, node := range nodeStats.Nodes {
		vectorsPerCall := 0.0
		if node.Calls != 0 {
			vectorsPerCall = float64(node.Vectors) / float64(node.Calls)
		}
		thread.Items = append(thread.Items, api.RuntimeItem{
			Index:          uint(node.NodeIndex),
			Name:           node.NodeName,
			State:          "unknown",
			Calls:          node.Calls,
			Vectors:        node.Vectors,
			Suspends:       node.Suspends,
			Clocks:         float64(node.Clocks),
			VectorsPerCall: vectorsPerCall,
		})
	}

	return &api.RuntimeInfo{Threads: []api.RuntimeThread{thread}}, nil
}

func (h *TelemetryHandler) GetThreads(ctx context.Context) ([]api.ThreadData, error) {
	threads := new(vlib.ShowThreadsReply)
	if err := h.ch.SendRequest(new(vlib.ShowThreads)).ReceiveReply(threads); err != nil {
		return nil, fmt.Errorf("show threads error: %v", err)
	}

	result := make([]api.ThreadData, len(threads.ThreadData))
	for i := range threads.ThreadData {
		result[i].ID = threads.ThreadData[i].ID
		result[i].Name = threads.ThreadData[i].Name
		result[i].Type = threads.ThreadData[i].Type
		result[i].PID = threads.ThreadData[i].PID
		result[i].Core = threads.ThreadData[i].Core
		result[i].CPUID = threads.ThreadData[i].CPUID
		result[i].CPUSocket = threads.ThreadData[i].CPUSocket
	}

	return result, nil
}

func strToFloat64(s string) float64 {
	// Replace 'k' (thousands) with 'e3' to make it parsable with strconv
	s = strings.Replace(s, "k", "e3", 1)
	s = strings.Replace(s, "K", "e3", 1)
	s = strings.Replace(s, "m", "e6", 1)
	s = strings.Replace(s, "M", "e6", 1)
	s = strings.Replace(s, "g", "e9", 1)
	s = strings.Replace(s, "G", "e9", 1)

	num, err := strconv.ParseFloat(s, 10)
	if err != nil {
		return 0
	}
	return num
}
