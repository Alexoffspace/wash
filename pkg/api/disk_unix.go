//go:build !windows

package api

import (
	"syscall"
)

// getDiskInfo returns disk information
func getDiskInfo() DiskInfo {
	info := DiskInfo{
		MountPoint: "/",
	}

	var stat syscall.Statfs_t
	if err := syscall.Statfs("/", &stat); err == nil {
		info.Total = stat.Blocks * uint64(stat.Bsize)
		info.Free = stat.Bfree * uint64(stat.Bsize)
		info.Used = info.Total - info.Free

		if info.Total > 0 {
			info.UsedPct = float64(info.Used) / float64(info.Total) * 100
		}
	}

	return info
}
