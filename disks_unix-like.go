//go:build linux || darwin

package main

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"syscall"
)

func getMainDiskStats() (string, string, string, string) {
	var stat syscall.Statfs_t
	err := syscall.Statfs("/", &stat)
	if err != nil {
		return "/", "0GB", "0GB", "0%"
	}

	totalBytes := uint64(stat.Blocks) * uint64(stat.Bsize)
	freeBytes := uint64(stat.Bavail) * uint64(stat.Bsize)
	usedBytes := totalBytes - freeBytes

	usedGB := float64(usedBytes) / (1024 * 1024 * 1024)
	totalGB := float64(totalBytes) / (1024 * 1024 * 1024)
	percent := (float64(usedBytes) / float64(totalBytes)) * 100

	return "/",
		fmt.Sprintf("%.1f GB", usedGB),
		fmt.Sprintf("%.1f GB", totalGB),
		fmt.Sprintf("%.0f%%", percent)
}

func getDisplayResolution() string {
	// Пытаемся получить разрешение через xrandr (для Linux с X11)
	cmd := exec.Command("sh", "-c", "xrandr | grep '*' | awk '{print $1}'")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err == nil && out.Len() > 0 {
		return strings.TrimSpace(out.String())
	}
	return "Unknown"
}

func getCPU() string {
	// Читаем имя процессора из /proc/cpuinfo
	data, err := os.ReadFile("/proc/cpuinfo")
	if err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if strings.Contains(line, "model name") {
				parts := strings.Split(line, ":")
				if len(parts) > 1 {
					return strings.TrimSpace(parts[1])
				}
			}
		}
	}
	return "Unknown CPU"
}

func getGPU() string {
	// Получаем видеокарту через lspci
	cmd := exec.Command("sh", "-c", "lspci | grep -i vga | cut -d ':' -f3")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err == nil && out.Len() > 0 {
		return strings.TrimSpace(out.String())
	}
	return "Unknown GPU"
}

func getMemory() string {
	// Читаем данные памяти из /proc/meminfo
	data, err := os.ReadFile("/proc/meminfo")
	if err == nil {
		lines := strings.Split(string(data), "\n")
		var total, free uint64
		for _, line := range lines {
			if strings.HasPrefix(line, "MemTotal:") {
				fmt.Sscanf(line, "MemTotal: %d", &total)
			}
			if strings.HasPrefix(line, "MemAvailable:") || strings.HasPrefix(line, "MemFree:") {
				if free == 0 {
					fmt.Sscanf(line, "MemAvailable: %d", &free)
				}
			}
		}
		if total > 0 {
			used := total - free
			return fmt.Sprintf("%.2f GB / %.2f GB (%.0f%%)", float64(used)/1024/1024, float64(total)/1024/1024, (float64(used)/float64(total))*100)
		}
	}
	return "Unknown RAM"
}

func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}
	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return "127.0.0.1"
}

func getBatteryStatus() string {
	// Читаем заряд батареи из sysfs Linux
	data, err := os.ReadFile("/sys/class/power_supply/BAT0/capacity")
	if err == nil {
		return strings.TrimSpace(string(data)) + "%"
	}
	return "No Battery"
}

func getLocale() string {
	return os.Getenv("LANG")
}