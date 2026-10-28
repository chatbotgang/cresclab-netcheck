//go:build linux

package main

import (
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

func systemCommands() []command {
	return []command{
		{"System", "uname", []string{"-a"}},
		{"Network interfaces: ip addr", "ip", []string{"addr"}},
		{"Routing table: ip route", "ip", []string{"route"}},
		{"DNS configuration: /etc/resolv.conf", "cat", []string{"/etc/resolv.conf"}},
	}
}

// Linux ping (iputils) takes -W in seconds.
func pingCommand(host string) command {
	return command{name: "ping", args: []string{"-c", "4", "-W", "2", host}}
}

func mtuPingCommand(ip string, size int) command {
	return command{name: "ping", args: []string{"-M", "do", "-s", strconv.Itoa(size), "-c", "2", "-W", "2", ip}}
}

// traceCommand falls back to tracepath, which many distributions ship
// without traceroute.
func traceCommand(host string) command {
	if _, err := exec.LookPath("traceroute"); err != nil {
		if _, err := exec.LookPath("tracepath"); err == nil {
			return command{name: "tracepath", args: []string{"-n", "-m", "20", host}}
		}
	}
	return command{name: "traceroute", args: []string{"-n", "-m", "20", "-w", "1", host}}
}

// systemProxy uses the HTTPS_PROXY and NO_PROXY environment variables, which
// is how Linux command-line tools usually receive a proxy.
func systemProxy() (*url.URL, string) {
	req, _ := http.NewRequest(http.MethodGet, "https://maac.cresclab.com/", nil)
	u, err := http.ProxyFromEnvironment(req)
	if err != nil {
		return nil, "Could not read proxy environment variables: " + err.Error()
	}
	if u == nil {
		return nil, "No proxy environment variable set; only the direct connection is tested."
	}
	return u, "Proxy from environment: " + u.Host
}

func desktopDir() string {
	if out, err := exec.Command("xdg-user-dir", "DESKTOP").Output(); err == nil {
		if dir := strings.TrimSpace(string(out)); dir != "" {
			if _, err := os.Stat(dir); err == nil {
				return dir
			}
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	dir := filepath.Join(home, "Desktop")
	if _, err := os.Stat(dir); err != nil {
		return ""
	}
	return dir
}

func revealFile(string) {}
