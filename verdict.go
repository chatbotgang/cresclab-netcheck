package main

import (
	"fmt"
	"strings"
)

// verdictPending is written at the top of the report and replaced with the
// verdict when every check has finished. If the program is closed early, the
// placeholder explains why the verdict is missing.
const verdictPending = "Verdict: not available yet; the check did not finish. See the sections below."

// hostResult is the outcome of probeHost for one host.
type hostResult struct {
	host        string
	unfinished  bool
	dnsOK       bool
	tcp4OK      int
	tcp4Tried   int
	tcp6OK      int
	tcp6Tried   int
	tcp6NoRoute bool   // every IPv6 attempt failed because the network has no IPv6 route
	tls         string // "ok" or "FAIL(kind)"
	https       string // HTTP status code or "FAIL(kind)"
	httpsProxy  string // like https, through the system proxy; "" when there is none
	http80      string // like https, for plain HTTP
}

func failed(status string) bool { return strings.HasPrefix(status, "FAIL") }

// failKind returns the part in parentheses of a "FAIL(kind)" status.
func failKind(status string) string {
	return strings.TrimSuffix(strings.TrimPrefix(status, "FAIL("), ")")
}

func (h hostResult) summary() string {
	if h.unfinished {
		return h.host + ": did not finish"
	}
	dns := "ok"
	if !h.dnsOK {
		dns = "FAIL"
	}
	parts := []string{"dns=" + dns, fmt.Sprintf("tcp4=%d/%d", h.tcp4OK, h.tcp4Tried)}
	if h.tcp6Tried > 0 {
		tcp6 := fmt.Sprintf("tcp6=%d/%d", h.tcp6OK, h.tcp6Tried)
		if h.tcp6NoRoute {
			tcp6 += "(no-route)"
		}
		parts = append(parts, tcp6)
	}
	parts = append(parts, "tls="+h.tls, "https="+h.https)
	if h.httpsProxy != "" {
		parts = append(parts, "https-via-proxy="+h.httpsProxy)
	}
	parts = append(parts, "http80="+h.http80)
	return h.host + ": " + strings.Join(parts, " ")
}

// failure names the first step that failed, or "" when the host answered over
// HTTPS directly or through the system proxy. Any HTTP status counts as an
// answer.
func (h hostResult) failure() string {
	switch {
	case h.unfinished:
		return "did not finish"
	case !failed(h.https), h.httpsProxy != "" && !failed(h.httpsProxy):
		return ""
	case !h.dnsOK:
		return "DNS"
	case h.tcp4OK+h.tcp6OK == 0:
		return "TCP connect"
	case failed(h.tls):
		return "TLS (" + failKind(h.tls) + ")"
	default:
		return "HTTPS (" + failKind(h.https) + ")"
	}
}

// sniResult records which name answered on which Cloudflare edge IP.
type sniResult map[string]map[string]bool

func (s sniResult) set(ip, name string, ok bool) {
	if s[ip] == nil {
		s[ip] = map[string]bool{}
	}
	s[ip][name] = ok
}

// hint explains what the SNI comparison says about the failure, or "" when it
// does not discriminate.
func (s sniResult) hint() string {
	allBlocked := len(crescendoEdgeIPs) > 0
	for _, ip := range crescendoEdgeIPs {
		if s[ip][controlSNIName] && !s[ip][crescendoSNIName] {
			return "The SNI comparison shows that the network filters by domain name; ask the company IT team to allow *.cresclab.com."
		}
		if s[ip][controlSNIName] || s[ip][crescendoSNIName] {
			allBlocked = false
		}
	}
	if allBlocked && s[controlEdgeIP][controlSNIName] {
		return "The SNI comparison shows that the Crescendo Lab Cloudflare IP addresses or their route are blocked."
	}
	return ""
}

// verdict sums up the service and control host results in one line.
func verdict(services, controls []hostResult, sni sniResult) string {
	var failures []string
	for _, h := range services {
		if f := h.failure(); f != "" {
			failures = append(failures, h.host+" failed at "+f)
		}
	}
	if len(failures) == 0 {
		return "Verdict: OK - every Crescendo Lab host responded over HTTPS."
	}
	controlsFailed := len(controls) > 0
	for _, h := range controls {
		if h.failure() == "" {
			controlsFailed = false
		}
	}
	if controlsFailed {
		return "Verdict: PROBLEM - the control hosts failed too, so this computer has no working internet access or needs a proxy that is not configured."
	}
	v := "Verdict: PROBLEM - " + strings.Join(failures, "; ") + "."
	if hint := sni.hint(); hint != "" {
		v += " " + hint
	}
	return v
}
