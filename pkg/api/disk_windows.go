//go:build windows

package api

import (
	"encoding/json"
	"os/exec"
)

func getDiskInfo() DiskInfo {
	info := DiskInfo{
		MountPoint: "/",
	}

	cmd := exec.Command("powershell", "-Command", "Get-Volume | Select-Object DriveLetter, Size, SizeRemaining | ConvertTo-Json")
	output, err := cmd.Output()
	if err == nil {
		var volumes []map[string]interface{}
		if err := json.Unmarshal(output, &volumes); err == nil && len(volumes) > 0 {
			vol := volumes[0]
			if size, ok := vol["Size"].(float64); ok {
				info.Total = uint64(size)
			}
			if remaining, ok := vol["SizeRemaining"].(float64); ok {
				info.Free = uint64(remaining)
				info.Used = info.Total - info.Free
				if info.Total > 0 {
					info.UsedPct = float64(info.Used) / float64(info.Total) * 100
				}
			}
			if driveLetter, ok := vol["DriveLetter"].(string); ok {
				info.MountPoint = driveLetter + ":"
			}
		}
	}

	return info
}
