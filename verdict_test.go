package main

import (
	"errors"
	"strings"
	"testing"
)

func okHost(host string) hostResult {
	return hostResult{host: host, dnsOK: true, tcp4OK: 2, tcp4Tried: 2, tls: "ok", https: "200", http80: "301"}
}

func TestHostSummarySplitsIPFamilies(t *testing.T) {
	h := okHost("www.cloudflare.com")
	if got, want := h.summary(), "www.cloudflare.com: dns=ok tcp4=2/2 tls=ok https=200 http80=301"; got != want {
		t.Errorf("IPv4 only: got %q, want %q", got, want)
	}
	h.tcp6Tried, h.tcp6NoRoute = 2, true
	if got, want := h.summary(), "www.cloudflare.com: dns=ok tcp4=2/2 tcp6=0/2(no-route) tls=ok https=200 http80=301"; got != want {
		t.Errorf("no IPv6 route: got %q, want %q", got, want)
	}
}

func TestNoRoute(t *testing.T) {
	for _, msg := range []string{
		"dial tcp [2606:4700::6810:7b60]:443: connect: no route to host",
		"dial tcp [2606:4700::6810:7b60]:443: connect: network is unreachable",
		"dial tcp [2606:4700::6810:7b60]:443: connectex: A socket operation was attempted to an unreachable network.",
	} {
		if !noRoute(errors.New(msg)) {
			t.Errorf("noRoute(%q) = false, want true", msg)
		}
	}
	if noRoute(errors.New("dial tcp 104.18.4.188:443: i/o timeout")) {
		t.Error("noRoute(timeout) = true, want false")
	}
}

func TestClip(t *testing.T) {
	if got := clip("short", 10); got != "short" {
		t.Errorf("clip(short) = %q", got)
	}
	// "封鎖" is 6 bytes; cutting at 4 must not split the second character.
	if got, want := clip("封鎖頁面", 4), "封 [truncated]"; got != want {
		t.Errorf("clip(UTF-8) = %q, want %q", got, want)
	}
}

func TestVerdict(t *testing.T) {
	services := []hostResult{okHost("maac.cresclab.com"), okHost("caac.cresclab.com")}
	controls := []hostResult{okHost("www.cloudflare.com"), okHost("www.google.com")}

	timedOutTLS := okHost("maac.cresclab.com")
	timedOutTLS.tls, timedOutTLS.https = "FAIL(timeout)", "FAIL(timeout)"

	tcpBlocked := okHost("maac.cresclab.com")
	tcpBlocked.tcp4OK, tcpBlocked.tls, tcpBlocked.https = 0, "FAIL(error)", "FAIL(error)"

	viaProxy := timedOutTLS
	viaProxy.httpsProxy = "200"

	down := func(h hostResult) hostResult {
		h.dnsOK, h.tcp4OK, h.tls, h.https = false, 0, "FAIL(dns)", "FAIL(dns)"
		return h
	}

	domainFiltered := sniResult{}
	ipBlocked := sniResult{}
	for _, ip := range crescendoEdgeIPs {
		domainFiltered.set(ip, crescendoSNIName, false)
		domainFiltered.set(ip, controlSNIName, true)
		ipBlocked.set(ip, crescendoSNIName, false)
		ipBlocked.set(ip, controlSNIName, false)
	}
	ipBlocked.set(controlEdgeIP, controlSNIName, true)

	tests := []struct {
		name     string
		services []hostResult
		controls []hostResult
		sni      sniResult
		want     []string
	}{
		{"all ok", services, controls, nil, []string{"Verdict: OK"}},
		{"reachable through proxy", []hostResult{viaProxy}, controls, nil, []string{"Verdict: OK"}},
		{"TLS timeout", []hostResult{timedOutTLS, services[1]}, controls, nil,
			[]string{"Verdict: PROBLEM", "maac.cresclab.com failed at TLS (timeout)"}},
		{"domain filter", []hostResult{timedOutTLS}, controls, domainFiltered,
			[]string{"maac.cresclab.com failed at TLS (timeout)", "filters by domain name"}},
		{"IP blocked", []hostResult{tcpBlocked}, controls, ipBlocked,
			[]string{"maac.cresclab.com failed at TCP connect", "IP addresses or their route are blocked"}},
		{"unfinished", []hostResult{{host: "maac.cresclab.com", unfinished: true}}, controls, nil,
			[]string{"maac.cresclab.com failed at did not finish"}},
		{"no internet", []hostResult{down(services[0])}, []hostResult{down(controls[0]), down(controls[1])}, nil,
			[]string{"control hosts failed too"}},
	}
	for _, tt := range tests {
		got := verdict(tt.services, tt.controls, tt.sni)
		for _, w := range tt.want {
			if !strings.Contains(got, w) {
				t.Errorf("%s: verdict = %q, want it to contain %q", tt.name, got, w)
			}
		}
	}
}
