//go:build !windows

// This file lets the program run on macOS and Linux for development.
package main

import (
	"net/url"
	"os"
	"path/filepath"
)

func consoleCodePage() uint32 { return 65001 }

func systemCommands() []command {
	return []command{
		{"System", "uname", []string{"-a"}},
		{"Network adapters", "ifconfig", nil},
	}
}

func pingCommand(host string) command {
	return command{name: "ping", args: []string{"-c", "4", "-W", "2000", host}}
}

func traceCommand(host string) command {
	return command{name: "traceroute", args: []string{"-n", "-m", "20", "-w", "1", host}}
}

func systemProxy() (*url.URL, string) {
	return nil, "System proxy detection is only implemented on Windows."
}

func desktopDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, "Desktop")
}

func revealFile(string) {}
