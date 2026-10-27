// Package status collects a point-in-time snapshot of the server state.
package status

import (
	"os"
	"runtime"
	"time"

	"v2home/internal/buildinfo"
)

// RuntimeStats is the subset of runtime/metrics exposed to the UI.
type RuntimeStats struct {
	Goroutines     int    `json:"goroutines"`
	NumCPU         int    `json:"numCpu"`
	HeapAllocBytes uint64 `json:"heapAllocBytes"`
	HeapSysBytes   uint64 `json:"heapSysBytes"`
	GCCount        uint32 `json:"gcCount"`
}

// TunnelSummary is a placeholder for the future tunnel engine.
type TunnelSummary struct {
	Total   int `json:"total"`
	Running int `json:"running"`
}

// Snapshot is the JSON payload returned by /api/status and /ws/status.
type Snapshot struct {
	App           string        `json:"app"`
	Version       string        `json:"version"`
	Commit        string        `json:"commit"`
	BuildTime     string        `json:"buildTime"`
	GoVersion     string        `json:"goVersion"`
	OS            string        `json:"os"`
	Arch          string        `json:"arch"`
	Hostname      string        `json:"hostname"`
	StartedAt     time.Time     `json:"startedAt"`
	UptimeSeconds int64         `json:"uptimeSeconds"`
	Now           time.Time     `json:"now"`
	Runtime       RuntimeStats  `json:"runtime"`
	Tunnels       TunnelSummary `json:"tunnels"`
}

// Collector produces snapshots relative to the process start time.
type Collector struct {
	startedAt time.Time
	hostname  string
}

// New creates a Collector anchored at the current time.
func New() *Collector {
	hostname, err := os.Hostname()
	if err != nil || hostname == "" {
		hostname = "unknown"
	}
	return &Collector{
		startedAt: time.Now().UTC(),
		hostname:  hostname,
	}
}

// StartedAt returns the process start time.
func (c *Collector) StartedAt() time.Time {
	return c.startedAt
}

// Snapshot reads the current runtime state.
func (c *Collector) Snapshot() Snapshot {
	now := time.Now().UTC()

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	return Snapshot{
		App:           "v2home",
		Version:       buildinfo.Version,
		Commit:        buildinfo.Commit,
		BuildTime:     buildinfo.BuildTime,
		GoVersion:     runtime.Version(),
		OS:            runtime.GOOS,
		Arch:          runtime.GOARCH,
		Hostname:      c.hostname,
		StartedAt:     c.startedAt,
		UptimeSeconds: int64(now.Sub(c.startedAt).Seconds()),
		Now:           now,
		Runtime: RuntimeStats{
			Goroutines:     runtime.NumGoroutine(),
			NumCPU:         runtime.NumCPU(),
			HeapAllocBytes: mem.HeapAlloc,
			HeapSysBytes:   mem.HeapSys,
			GCCount:        mem.NumGC,
		},
		// Placeholder until the tunnel engine lands.
		Tunnels: TunnelSummary{Total: 0, Running: 0},
	}
}
