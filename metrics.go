package main

import (
	"net/http"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

func MetricsHandler(w http.ResponseWriter, r *http.Request) {
	// CPU percent requires a sampling interval; keep it short for “realtime”.
	cpuPct, _ := cpu.Percent(350*time.Millisecond, false) // overall
	cpuPctVal := 0.0
	if len(cpuPct) > 0 {
		cpuPctVal = cpuPct[0]
	}

	vm, _ := mem.VirtualMemory()
	la, _ := load.Avg()
	ioCounters, _ := net.IOCounters(false)
	di, _ := disk.Usage("/") // root filesystem
	hi, _ := host.Info()

	netRx := uint64(0)
	netTx := uint64(0)
	if len(ioCounters) > 0 {
		netRx = ioCounters[0].BytesRecv
		netTx = ioCounters[0].BytesSent
	}

	resp := map[string]any{
		"timestamp_utc": time.Now().UTC().Format(time.RFC3339),

		"host": map[string]any{
			"hostname":  hi.Hostname,
			"uptime_s":  hi.Uptime,
			"boot_time": hi.BootTime,
		},

		"cpu": map[string]any{
			"percent": cpuPctVal,
			"cores":   hi.Procs, // process count from host.Info; still useful
		},

		"memory": map[string]any{
			"total":        vm.Total,
			"available":    vm.Available,
			"used":         vm.Used,
			"used_percent": vm.UsedPercent,
		},

		"load": map[string]any{
			"load1":  la.Load1,
			"load5":  la.Load5,
			"load15": la.Load15,
		},

		"disk_root": map[string]any{
			"path":         "/",
			"total":        di.Total,
			"used":         di.Used,
			"free":         di.Free,
			"used_percent": di.UsedPercent,
		},

		"network": map[string]any{
			"bytes_recv": netRx,
			"bytes_sent": netTx,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	_ = writeJSON(w, resp)
}
