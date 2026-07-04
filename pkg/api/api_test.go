package api

import (
	"testing"
)

func TestGetCPUInfo(t *testing.T) {
	info := getCPUInfo()
	if info.Cores <= 0 {
		t.Errorf("CPU cores should be > 0, got %d", info.Cores)
	}
}

func TestGetRealCPUUsage(t *testing.T) {
	pct := getRealCPUUsage()
	if pct < -1 || pct > 100 {
		t.Errorf("CPU usage_pct out of range: %f", pct)
	}
}

func TestGetCPUInfoLoadAverages(t *testing.T) {
	info := getCPUInfo()
	if info.Load1 == "" {
		t.Error("Load1 should not be empty")
	}
	if info.Load5 == "" {
		t.Error("Load5 should not be empty")
	}
	if info.Load15 == "" {
		t.Error("Load15 should not be empty")
	}
}

func TestGetMemoryInfo(t *testing.T) {
	info := getMemoryInfo()
	if info.Total > 0 {
		if info.UsedPct < 0 || info.UsedPct > 100 {
			t.Errorf("Memory used_pct out of range: %f", info.UsedPct)
		}
		if info.Used > info.Total {
			t.Errorf("Memory used (%d) > total (%d)", info.Used, info.Total)
		}
	}
	if info.Total == 0 && info.Used > 0 {
		t.Error("Memory used > 0 but total is 0")
	}
}

func TestGetSystemUptime(t *testing.T) {
	uptime := getSystemUptime()
	if uptime == "" {
		t.Error("Uptime should not be empty")
	}
	if uptime == "unknown" {
		t.Log("Uptime is unknown (may be OK on some platforms)")
	}
}
