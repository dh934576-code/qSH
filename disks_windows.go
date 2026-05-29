//go:build windows
package main

import (
	"bytes"
	"fmt"
	"net"
	"os/exec"
	"strings"
	"syscall"
	"unsafe"
)

var (
	modkernel32          = syscall.NewLazyDLL("kernel32.dll")
	procGetDiskFreeSpace = modkernel32.NewProc("GetDiskFreeSpaceExW")
)

type SYSTEM_POWER_STATUS struct {
	ACLineStatus        byte
	BatteryFlag         byte
	BatteryLifePercent  byte
	SystemStatus        byte
	BatteryLifeTime     uint32
	BatteryFullLifeTime uint32
}

func getMainDiskStats() (string, string, string, string) {
	var freeBytes, totalBytes, totalFree uint64

	pathPtr, err := syscall.UTF16PtrFromString("C:\\")
	if err != nil {
		return "C:\\", "0GB", "0GB", "0%"
	}

	r1, _, _ := procGetDiskFreeSpace.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		uintptr(unsafe.Pointer(&freeBytes)),
		uintptr(unsafe.Pointer(&totalBytes)),
		uintptr(unsafe.Pointer(&totalFree)),
	)

	if r1 == 0 {
		return "C:\\", "0GB", "0GB", "0%"
	}

	usedBytes := totalBytes - freeBytes
	usedGB := float64(usedBytes) / (1024 * 1024 * 1024)
	totalGB := float64(totalBytes) / (1024 * 1024 * 1024)
	percent := (float64(usedBytes) / float64(totalBytes)) * 100

	return "C:\\",
		fmt.Sprintf("%.1f GB", usedGB),
		fmt.Sprintf("%.1f GB", totalGB),
		fmt.Sprintf("%.0f%%", percent)
}

func getCPU() string {
	cmd := exec.Command("wmic", "cpu", "get", "name")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err == nil {
		lines := strings.Split(out.String(), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" && line != "Name" {
				return line
			}
		}
	}
	return "Unknown CPU"
}

func getGPU() string {
	cmd := exec.Command("wmic", "path", "win32_VideoController", "get", "name")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err == nil {
		lines := strings.Split(out.String(), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" && line != "Name" {
				return line
			}
		}
	}
	return "Unknown GPU"
}

func getMemory() string {
	cmdTotal := exec.Command("wmic", "OS", "get", "TotalVisibleMemorySize")
	cmdFree := exec.Command("wmic", "OS", "get", "FreePhysicalMemory")
	var outT, outF bytes.Buffer
	cmdTotal.Stdout = &outT
	cmdFree.Stdout = &outF

	if cmdTotal.Run() == nil && cmdFree.Run() == nil {
		linesT := strings.Split(outT.String(), "\n")
		linesF := strings.Split(outF.String(), "\n")
		var total, free uint64
		for _, l := range linesT {
			l = strings.TrimSpace(l)
			if l != "" && l != "TotalVisibleMemorySize" {
				fmt.Sscanf(l, "%d", &total)
			}
		}
		for _, l := range linesF {
			l = strings.TrimSpace(l)
			if l != "" && l != "FreePhysicalMemory" {
				fmt.Sscanf(l, "%d", &free)
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

func getDisplayResolution() string {
	// Безопасно загружаем системную библиотеку user32.dll
	mod := syscall.NewLazyDLL("user32.dll")
	proc := mod.NewProc("GetSystemMetrics")
	
	// 0 — ширина экрана (SM_CXSCREEN), 1 — высота (SM_CYSCREEN)
	width, _, _ := proc.Call(0)
	height, _, _ := proc.Call(1)
	
	// Если WinAPI вернул нули (например, в фоновом режиме)
	if width == 0 || height == 0 {
		return "Unknown Resolution"
	}
	return fmt.Sprintf("%dx%d", width, height)
}

func getBatteryStatus() string {
	// Запрашиваем процент оставшегося заряда через PowerShell
	cmd := exec.Command("powershell", "-Command", "Get-CimInstance -ClassName Win32_Battery | Select-Object -ExpandProperty EstimatedChargeRemaining")
	out, err := cmd.Output()
	
	// Если произошла ошибка или у вас стационарный ПК (массив пустой), возвращаем безопасный текст
	if err != nil || len(out) == 0 {
		return "AC Power (No Battery)"
	}
	
	// Если батарея есть, очищаем текст от лишних пробелов и добавляем значок процента
	return strings.TrimSpace(string(out)) + "%"
}

func getLocale() string {
	// Запрашиваем культуру системы (например, ru-RU или en-US) через PowerShell
	cmd := exec.Command("powershell", "-Command", "[System.Globalization.CultureInfo]::CurrentCulture.Name")
	out, err := cmd.Output()
	if err != nil || len(out) == 0 {
		return "ru-RU (Дефолт)" // Безопасный возврат, если PowerShell заблокирован
	}
	return strings.TrimSpace(string(out))
}