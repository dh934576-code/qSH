//go:build freebsd || openbsd || netbsd || dragonfly

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
	// В BSD системный вызов принимает путь
	err := syscall.Statfs("/", &stat)
	if err != nil {
		return "/", "0GB", "0GB", "0%"
	}

	// В некоторых BSD типы блоков могут быть uint32 или uint64, приводим явно
	totalBytes := uint64(stat.Blocks) * uint64(stat.Bsize)
	freeBytes := uint64(stat.Bavail) * uint64(stat.Bsize)
	usedBytes := totalBytes - freeBytes

	return "/", 
		fmt.Sprintf("%.1f GB", float64(usedBytes)/(1024*1024*1024)), 
		fmt.Sprintf("%.1f GB", float64(totalBytes)/(1024*1024*1024)), 
		fmt.Sprintf("%.0f%%", (float64(usedBytes)/float64(totalBytes))*100)
}

func getCPU() string {
	cpu, err := syscall.Sysctl("hw.model")
	if err != nil {
		return "Unknown BSD CPU"
	}
	return cpu
}

func getMemory() string {
	// hw.physmem возвращает общий объем ОЗУ в байтах
	mem, err := syscall.SysctlUint64("hw.physmem")
	if err != nil {
		return "Unknown RAM"
	}
	// Переводим байты в Гигабайты
	gb := float64(mem) / (1024 * 1024 * 1024)
	return fmt.Sprintf("%.1f GB", gb)
}

func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "Unknown IP"
	}
	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return "No Connection"
}

func getLocale() string {
	locale := os.Getenv("LANG")
	if locale == "" {
		locale = os.Getenv("LC_ALL")
	}
	if locale == "" {
		return "en_US.UTF-8 (Default)"
	}
	return locale
}

func getBatteryStatus() string {
	// Запрашиваем оставшийся процент заряда в BSD
	life, err := syscall.SysctlUint32("hw.acpi.battery.life")
	if err != nil {
		// Если не сработало, возможно это macOS (Darwin). Проверяем сисктл для Mac:
		lifeMac, errMac := syscall.SysctlUint32("hw.sensors.acpibat0.raw0") 
		if errMac != nil {
			return "No Battery (AC Power)"
		}
		return fmt.Sprintf("%d%%", lifeMac)
	}
	
	// Проверяем состояние питания (заряжается или разряжается)
	state, _ := syscall.SysctlUint32("hw.acpi.battery.state")
	if state == 2 {
		return fmt.Sprintf("%d%% (Charging)", life)
	}
	return fmt.Sprintf("%d%%", life)
}

func getDisplayResolution() string {
	// Проверяем, есть ли в системе утилита xrandr
	cmd := exec.Command("xrandr")
	var out bytes.Buffer
	cmd.Stdout = &out
	
	err := cmd.Run()
	if err != nil {
		return "Headless (No Display)"
	}
	
	// Парсим строку со словом 'current'
	lines := strings.Split(out.String(), "\n")
	for _, line := range lines {
		if strings.Contains(line, "current") {
			// Пример строки: Screen 0: minimum 320 x 200, current 1920 x 1080...
			parts := strings.Split(line, "current")
			if len(parts) > 1 {
				res := strings.Split(parts[1], ",")[0]
				return strings.TrimSpace(res) // Вернет "1920 x 1080"
			}
		}
	}
	return "Unknown Resolution"
}

func getGPU() string {
	// Вызываем pciconf, который показывает список всех PCI устройств
	// Флаг -lv выводит подробную информацию с именами устройств
	cmd := exec.Command("pciconf", "-lv")
	var out bytes.Buffer
	cmd.Stdout = &out
	
	err := cmd.Run()
	if err != nil {
		// Если pciconf недоступен, пробуем прочитать dmesg на предмет графики (vgapci)
		cmdDmesg := exec.Command("dmesg")
		var outDmesg bytes.Buffer
		cmdDmesg.Stdout = &outDmesg
		if cmdDmesg.Run() == nil {
			lines := strings.Split(outDmesg.String(), "\n")
			for _, line := range lines {
				if strings.Contains(line, "vgapci") && strings.Contains(line, ": <") {
					// dmesg обычно пишет что-то вроде: vgapci0: <NVIDIA GeForce...>
					parts := strings.Split(line, "<")
					if len(parts) > 1 {
						return strings.Trim(parts[1], ">")
					}
				}
			}
		}
		return "Unknown GPU"
	}

	// Если pciconf сработал, парсим его вывод в поисках VGA-контроллера
	lines := strings.Split(out.String(), "\n")
	for i, line := range lines {
		if strings.Contains(line, "class=0x030000") || strings.Contains(line, "vgapci") {
			// Имя устройства обычно пишется на следующей или текущей строке в поле 'device'
			for j := i; j < i+4 && j < len(lines); j++ {
				if strings.Contains(lines[j], "device") {
					parts := strings.Split(lines[j], "=")
					if len(parts) > 1 {
						return strings.Trim(parts[1])
					}
				}
			}
		}
	}
	return "Standard VGA Graphics"
}