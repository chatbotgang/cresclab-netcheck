package main

import (
	"bufio"
	"net/url"
	"strconv"
	"strings"
)

// parseProxyServer converts the Windows ProxyServer registry value, either
// "host:port" or "http=host:port;https=host:port", into the proxy for HTTPS.
func parseProxyServer(value string) *url.URL {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	server := value
	if strings.Contains(value, "=") {
		byScheme := map[string]string{}
		for _, part := range strings.Split(value, ";") {
			if scheme, addr, ok := strings.Cut(part, "="); ok {
				byScheme[strings.ToLower(strings.TrimSpace(scheme))] = strings.TrimSpace(addr)
			}
		}
		server = byScheme["https"]
		if server == "" {
			server = byScheme["http"]
		}
		if server == "" {
			return nil
		}
	}
	return proxyURL(server)
}

// parseScutilProxy reads the HTTPS proxy from macOS `scutil --proxy` output.
// A PAC script is reported in the note but not evaluated.
func parseScutilProxy(out string) (*url.URL, string) {
	values := map[string]string{}
	sc := bufio.NewScanner(strings.NewReader(out))
	for sc.Scan() {
		if k, v, ok := strings.Cut(sc.Text(), " : "); ok {
			values[strings.TrimSpace(k)] = strings.TrimSpace(v)
		}
	}
	note := ""
	if values["ProxyAutoConfigEnable"] == "1" {
		note = "PAC script configured but not evaluated: " + values["ProxyAutoConfigURLString"] + "\n"
	}
	if values["HTTPSEnable"] != "1" || values["HTTPSProxy"] == "" {
		return nil, note + "No manual HTTPS proxy enabled; only the direct connection is tested."
	}
	server := values["HTTPSProxy"]
	if port, err := strconv.Atoi(values["HTTPSPort"]); err == nil && port > 0 {
		server += ":" + strconv.Itoa(port)
	}
	u := proxyURL(server)
	if u == nil {
		return nil, note + "Manual HTTPS proxy enabled but could not be parsed: " + server
	}
	return u, note + "Manual HTTPS proxy enabled: " + u.Host
}

func proxyURL(server string) *url.URL {
	if !strings.Contains(server, "://") {
		server = "http://" + server
	}
	u, err := url.Parse(server)
	if err != nil || u.Host == "" {
		return nil
	}
	return u
}
