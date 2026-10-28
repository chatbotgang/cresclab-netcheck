//go:build darwin

package main

import (
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
)

func systemCommands() []command {
	return []command{
		{"System", "sw_vers", nil},
		{"Network interfaces: ifconfig", "ifconfig", nil},
		{"DNS configuration: scutil --dns", "scutil", []string{"--dns"}},
		{"Proxy settings: scutil --proxy", "scutil", []string{"--proxy"}},
		{"Routing table", "netstat", []string{"-rn", "-f", "inet"}},
	}
}

// macOS ping takes -W in milliseconds.
func pingCommand(host string) command {
	return command{name: "ping", args: []string{"-c", "4", "-W", "2000", host}}
}

func mtuPingCommand(ip string, size int) command {
	return command{name: "ping", args: []string{"-D", "-s", strconv.Itoa(size), "-c", "2", "-W", "2000", ip}}
}

func traceCommand(host string) command {
	return command{name: "traceroute", args: []string{"-n", "-m", "20", "-w", "1", host}}
}

func systemProxy() (*url.URL, string) {
	out, err := exec.Command("scutil", "--proxy").Output()
	if err != nil {
		return nil, "Could not read proxy settings: " + err.Error()
	}
	return parseScutilProxy(string(out))
}

func desktopDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, "Desktop")
}

func revealFile(path string) {
	_ = exec.Command("open", "-R", path).Start()
}
