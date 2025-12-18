package main

import (
	"bufio"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

func SystemHandler(w http.ResponseWriter, r *http.Request) {
	cpuModel := readCPUModel()
	kernel := readKernelVersion()
	uptimeSec, uptimeHuman := readUptime()

	upgCount, upgErr := debianUpgradableCount()
	alerts, alertsErr := systemAlerts()

	resp := map[string]any{
		"cpu_model":      cpuModel,
		"kernel_version": kernel,
		"os":             runtime.GOOS,
		"arch":           runtime.GOARCH,
		"uptime_seconds": uptimeSec,
		"uptime_human":   uptimeHuman,

		"packages": map[string]any{
			"debian_based":           true,
			"upgradable_count":       upgCount,
			"upgradable_check_error": errString(upgErr),
		},

		"alerts": map[string]any{
			"source": "dmesg(err,crit,alert,emerg)",
			"items":  alerts,
			"error":  errString(alertsErr),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	_ = writeJSON(w, resp)
}

func readCPUModel() string {
	f, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return "unknown"
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "model name") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return "unknown"
}

func readKernelVersion() string {
	out, err := exec.Command("uname", "-r").Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

func readUptime() (seconds int64, human string) {
	b, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0, "unknown"
	}
	fields := strings.Fields(string(b))
	if len(fields) == 0 {
		return 0, "unknown"
	}

	// parse like "12345.67"
	sec := int64(0)
	dot := strings.IndexByte(fields[0], '.')
	if dot >= 0 {
		fields[0] = fields[0][:dot]
	}
	for _, ch := range fields[0] {
		if ch < '0' || ch > '9' {
			break
		}
		sec = sec*10 + int64(ch-'0')
	}

	return sec, (time.Duration(sec) * time.Second).String()
}

func debianUpgradableCount() (int, error) {
	// Fast-ish and works on Debian/Ubuntu:
	// apt list --upgradable (first line is "Listing...").
	cmd := exec.Command("bash", "-lc", "apt list --upgradable 2>/dev/null | tail -n +2 | wc -l")
	out, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	s := strings.TrimSpace(string(out))
	n := 0
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			continue
		}
		n = n*10 + int(ch-'0')
	}
	return n, nil
}

func systemAlerts() ([]string, error) {
	// Pull last 20 high-severity kernel messages (non-root usually works).
	cmd := exec.Command("bash", "-lc", "dmesg --level=err,crit,alert,emerg 2>/dev/null | tail -n 20")
	out, err := cmd.Output()
	if err != nil {
		// if dmesg restricted, return empty + error
		return []string{}, err
	}

	lines := []string{}
	sc := bufio.NewScanner(strings.NewReader(string(out)))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines, nil
}

func errString(err error) any {
	if err == nil {
		return nil
	}
	return err.Error()
}
