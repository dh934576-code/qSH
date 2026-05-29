//go:build solaris || illumos

package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
)

// В Solaris структура syscall отличается, проще отдать дефолт, если это просто шелл
func getMainDiskStats() (string, string, string, string) {
	return "/", "0GB", "0GB", "0%"
}

func getCPU() string {
	// Пробуем получить тип процессора
	cmd := exec.Command("uname", "-p")
	out, err := cmd.Output()
	if err != nil {
		return "Solaris CPU"
	}
	arch := strings.TrimSpace(string(out))

	// Пробуем получить модель через psrinfo (если есть права)
	cmd2 := exec.Command("psrinfo", "-v")
	out2, err2 := cmd2.Output()
	if err2 == nil {
		lines := strings.Split(string(out2), "\n")
		for _, line := range lines {
			if strings.Contains(line, "The") && strings.Contains(line, "processor") {
				// Пример: The x86 processor operates at 2400 MHz
				return strings.TrimSpace(line)
			}
		}
	}
	return arch + " Processor"
}

func getMemory() string {
	cmd := exec.Command("prtconf")
	out, err := cmd.Output()
	if err != nil {
		return "Unknown RAM"
	}
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if strings.Contains(line, "Memory size:") {
			// Убираем текст "Memory size:" и оставляем только цифры с MB
			res := strings.Replace(line, "Memory size:", "", 1)
			return strings.TrimSpace(res)
		}
	}
	return "Unknown RAM"
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
		return "C (POSIX)"
	}
	return locale
}

func getGPU() string {
	return "Solaris Framebuffer / Vesa"
}

func getBatteryStatus() string {
	return "No Battery (Server/AC Power)"
}

func getDisplayResolution() string {
	return "Headless (Console Mode)"
}