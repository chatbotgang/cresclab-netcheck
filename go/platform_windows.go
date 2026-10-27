//go:build windows

package main

import (
	"bufio"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

const internetSettingsKey = `HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings`

var kernel32 = syscall.NewLazyDLL("kernel32.dll")

func consoleCodePage() uint32 {
	if cp, _, _ := kernel32.NewProc("GetConsoleOutputCP").Call(); cp != 0 {
		return uint32(cp)
	}
	cp, _, _ := kernel32.NewProc("GetOEMCP").Call()
	return uint32(cp)
}

func systemCommands() []command {
	return []command{
		{"System", "cmd", []string{"/c", "ver"}},
		{"Network adapters: ipconfig /all", "ipconfig", []string{"/all"}},
		{"WinHTTP proxy", "netsh", []string{"winhttp", "show", "proxy"}},
		{"Browser proxy settings", "reg", []string{"query", internetSettingsKey}},
	}
}

func pingCommand(host string) command {
	return command{name: "ping", args: []string{"-n", "4", "-w", "2000", host}}
}

func mtuPingCommand(ip string, size int) command {
	return command{name: "ping", args: []string{"-f", "-l", strconv.Itoa(size), "-n", "2", "-w", "2000", ip}}
}

func traceCommand(host string) command {
	return command{name: "tracert", args: []string{"-d", "-h", "20", "-w", "1000", host}}
}

// systemProxy reads the per-user proxy that browsers use. A PAC script set by
// AutoConfigURL is reported but not evaluated.
func systemProxy() (*url.URL, string) {
	values := map[string]string{}
	out, _ := exec.Command("reg", "query", internetSettingsKey).Output()
	sc := bufio.NewScanner(strings.NewReader(decodeConsoleOutput(out)))
	for sc.Scan() {
		f := strings.Fields(sc.Text())
		if len(f) >= 3 && strings.HasPrefix(f[1], "REG_") {
			values[f[0]] = strings.Join(f[2:], " ")
		}
	}
	note := ""
	if pac := values["AutoConfigURL"]; pac != "" {
		note = "PAC script configured but not evaluated: " + pac + "\n"
	}
	if values["ProxyEnable"] != "0x1" {
		return nil, note + "No manual proxy enabled; only the direct connection is tested."
	}
	u := parseProxyServer(values["ProxyServer"])
	if u == nil {
		return nil, note + "Manual proxy enabled but ProxyServer could not be parsed: " + values["ProxyServer"]
	}
	return u, note + "Manual proxy enabled: " + u.Host
}

func desktopDir() string {
	out, err := exec.Command("powershell", "-NoProfile", "-Command", "[Environment]::GetFolderPath('Desktop')").Output()
	if dir := strings.TrimSpace(string(out)); err == nil && dir != "" {
		if _, err := os.Stat(dir); err == nil {
			return dir
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, "Desktop")
}

func revealFile(path string) {
	_ = exec.Command("explorer", "/select,", path).Start()
}
