package main

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"os/exec"
	"strings"
	"time"
	"unicode/utf8"
)

type command struct {
	title string
	name  string
	args  []string
}

// runCommand runs an external tool with a timeout and returns its decoded
// output. ok is false when the tool is missing, times out, or exits non-zero.
func runCommand(name string, args ...string) string {
	out, _ := runCommandOK(name, args...)
	return out
}

func runCommandOK(name string, args ...string) (string, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	text := decodeConsoleOutput(out)
	if err != nil {
		text += fmt.Sprintf("[%s: %v]\n", name, err)
	}
	return text, err == nil
}

func probeHost(t *text, host string, proxyURL *url.URL) hostResult {
	h := hostResult{host: host}

	t.printf("--- DNS, system resolver\n")
	ctx, cancel := context.WithTimeout(context.Background(), dialTimeout)
	addrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	cancel()
	if err != nil {
		t.printf("ERROR: %v\n", err)
	} else {
		t.printf("%s\n", joinAddrs(addrs))
		h.dnsOK = true
	}
	for _, server := range publicResolvers {
		t.printf("--- DNS, %s\n%s\n", server, lookupVia(server, host))
	}

	ping := pingCommand(host)
	t.printf("--- ping\n%s", runCommand(ping.name, ping.args...))

	// Up to two addresses per IP family, so a network without IPv6 does not
	// hide the IPv4 results and the summary can tell the two apart.
	t.printf("--- TCP connect to port 443\n")
	tcp6NoRoute := 0
	for _, a := range addrs {
		v4 := a.IP.To4() != nil
		if (v4 && h.tcp4Tried == 2) || (!v4 && h.tcp6Tried == 2) {
			continue
		}
		line, err := tcpConnect(a.IP)
		t.printf("%s\n", line)
		switch {
		case v4:
			h.tcp4Tried++
			if err == nil {
				h.tcp4OK++
			}
		default:
			h.tcp6Tried++
			if err == nil {
				h.tcp6OK++
			} else if noRoute(err) {
				tcp6NoRoute++
			}
		}
	}
	h.tcp6NoRoute = h.tcp6Tried > 0 && tcp6NoRoute == h.tcp6Tried
	if h.tcp4Tried+h.tcp6Tried == 0 {
		t.printf("Skipped: no address from the system resolver\n")
	}

	var detail string
	detail, h.tls = tlsHandshake(host)
	t.printf("--- TLS handshake\n%s\n", detail)

	detail, h.https = httpProbe("https://"+host+"/", probeOptions{})
	t.printf("--- HTTPS GET, direct connection\n%s\n", detail)

	if proxyURL != nil {
		detail, h.httpsProxy = httpProbe("https://"+host+"/", probeOptions{proxy: proxyURL})
		t.printf("--- HTTPS GET, through system proxy %s\n%s\n", proxyURL.Host, detail)
	}

	detail, h.http80 = plainHTTP("http://" + host + "/")
	t.printf("--- HTTP port 80, first lines; a firewall block page shows up here\n%s\n", detail)

	return h
}

// noRoute reports whether a dial failed because the network has no route to
// the address, as with IPv6 on an IPv4-only network. Windows reports this as
// an "unreachable network" or "unreachable host".
func noRoute(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "no route") || strings.Contains(msg, "unreachable")
}

func joinAddrs(addrs []net.IPAddr) string {
	s := make([]string, len(addrs))
	for i, a := range addrs {
		s[i] = a.IP.String()
	}
	return strings.Join(s, " ")
}

func lookupVia(server, host string) string {
	res := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
			d := net.Dialer{Timeout: 5 * time.Second}
			return d.DialContext(ctx, network, net.JoinHostPort(server, "53"))
		},
	}
	ctx, cancel := context.WithTimeout(context.Background(), dialTimeout)
	defer cancel()
	addrs, err := res.LookupIPAddr(ctx, host)
	if err != nil {
		return "ERROR: " + err.Error()
	}
	return joinAddrs(addrs)
}

func tcpConnect(ip net.IP) (string, error) {
	addr := net.JoinHostPort(ip.String(), "443")
	start := time.Now()
	conn, err := net.DialTimeout("tcp", addr, dialTimeout)
	elapsed := time.Since(start).Round(time.Millisecond)
	if err != nil {
		return fmt.Sprintf("%s FAILED after %s: %v", addr, elapsed, err), err
	}
	_ = conn.Close()
	return fmt.Sprintf("%s OK in %s", addr, elapsed), nil
}

// tlsHandshake reports the certificate the network actually presents. When a
// company firewall intercepts HTTPS, the issuer shows its own CA instead of
// the public one, so a failed verification is retried without verification to
// reveal that issuer.
func tlsHandshake(host string) (string, string) {
	dialer := &net.Dialer{Timeout: dialTimeout}
	addr := net.JoinHostPort(host, "443")
	start := time.Now()
	conn, err := tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{ServerName: host})
	if err == nil {
		defer func() { _ = conn.Close() }()
		return describeTLS(conn, time.Since(start), "verified"), "ok"
	}
	failure := fmt.Sprintf("FAILED after %s: %v", time.Since(start).Round(time.Millisecond), err)
	var verifyErr *tls.CertificateVerificationError
	if !errors.As(err, &verifyErr) {
		return failure, "FAIL(" + errKind(err) + ")"
	}
	start = time.Now()
	conn, err = tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{ServerName: host, InsecureSkipVerify: true})
	if err != nil {
		return failure, "FAIL(untrusted-cert)"
	}
	defer func() { _ = conn.Close() }()
	return failure + "\n" + describeTLS(conn, time.Since(start), "NOT trusted by this computer"), "FAIL(untrusted-cert)"
}

func describeTLS(conn *tls.Conn, elapsed time.Duration, trust string) string {
	st := conn.ConnectionState()
	s := fmt.Sprintf("OK in %s, %s, %s, remote %s, certificate %s",
		elapsed.Round(time.Millisecond), tls.VersionName(st.Version), tls.CipherSuiteName(st.CipherSuite), conn.RemoteAddr(), trust)
	if len(st.PeerCertificates) > 0 {
		c := st.PeerCertificates[0]
		s += fmt.Sprintf("\n  subject: %s\n  issuer:  %s\n  expires: %s", c.Subject, c.Issuer, c.NotAfter.Format("2006-01-02"))
	}
	return s
}

type probeOptions struct {
	proxy  *url.URL
	dialIP string // connect to this IP instead of resolving the host
}

// httpProbe sends one GET without following redirects and reports how long
// each step took. A "-" step never happened, which shows where it stalled.
func httpProbe(target string, opts probeOptions) (string, string) {
	dialer := &net.Dialer{Timeout: dialTimeout}
	transport := &http.Transport{
		DisableKeepAlives:   true,
		ForceAttemptHTTP2:   true,
		TLSHandshakeTimeout: dialTimeout,
		DialContext:         dialer.DialContext,
	}
	if opts.proxy != nil {
		transport.Proxy = http.ProxyURL(opts.proxy)
	}
	if opts.dialIP != "" {
		transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			_, port, _ := net.SplitHostPort(addr)
			return dialer.DialContext(ctx, network, net.JoinHostPort(opts.dialIP, port))
		}
	}
	client := &http.Client{
		Transport:     transport,
		Timeout:       requestTimeout,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}

	var dnsDone, connDone, tlsDone, firstByte time.Time
	var remote string
	trace := &httptrace.ClientTrace{
		DNSDone:              func(httptrace.DNSDoneInfo) { dnsDone = time.Now() },
		ConnectDone:          func(_, addr string, _ error) { connDone = time.Now(); remote = addr },
		TLSHandshakeDone:     func(tls.ConnectionState, error) { tlsDone = time.Now() },
		GotFirstResponseByte: func() { firstByte = time.Now() },
	}
	req, err := http.NewRequest(http.MethodGet, target, nil)
	if err != nil {
		return "ERROR: " + err.Error(), "FAIL(error)"
	}
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
	req.Header.Set("User-Agent", "cresclab-netcheck/"+version)

	start := time.Now()
	resp, err := client.Do(req)
	timing := fmt.Sprintf("remote=%s dns=%s tcp=%s tls=%s ttfb=%s total=%s",
		remote, since(start, dnsDone), since(start, connDone), since(start, tlsDone), since(start, firstByte), time.Since(start).Round(time.Millisecond))
	if err != nil {
		return "ERROR: " + err.Error() + "\n" + timing, "FAIL(" + errKind(err) + ")"
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
	_ = resp.Body.Close()
	return fmt.Sprintf("HTTP %d %s\n%s", resp.StatusCode, resp.Proto, timing), fmt.Sprint(resp.StatusCode)
}

// plainHTTP shows the start of a plain HTTP response. Firewalls that block a
// site often answer port 80 with their own block page naming the vendor and
// the category.
func plainHTTP(target string) (string, string) {
	client := &http.Client{
		Timeout:       requestTimeout,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	resp, err := client.Get(target)
	if err != nil {
		return "ERROR: " + err.Error(), "FAIL(" + errKind(err) + ")"
	}
	defer func() { _ = resp.Body.Close() }()
	var b strings.Builder
	fmt.Fprintf(&b, "HTTP %d %s\n", resp.StatusCode, resp.Proto)
	for _, h := range []string{"Server", "Location", "Content-Type", "Via", "X-Cache"} {
		if v := resp.Header.Get(h); v != "" {
			fmt.Fprintf(&b, "%s: %s\n", h, v)
		}
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	if s := strings.TrimSpace(string(body)); s != "" {
		fmt.Fprintf(&b, "\n%s\n", clip(s, http80BodyLimit))
	}
	return strings.TrimRight(b.String(), "\n"), fmt.Sprint(resp.StatusCode)
}

// http80BodyLimit keeps a block page readable without printing a whole site.
const http80BodyLimit = 300

// clip shortens s to at most n bytes without splitting a UTF-8 character.
func clip(s string, n int) string {
	if len(s) <= n {
		return s
	}
	for n > 0 && !utf8.RuneStart(s[n]) {
		n--
	}
	return s[:n] + " [truncated]"
}

func cloudflareTraceCheck(t *text) result {
	summary := "Public IP: unknown"
	for _, u := range cloudflareTrace {
		client := &http.Client{Timeout: requestTimeout}
		resp, err := client.Get(u)
		if err != nil {
			t.printf("--- %s\nERROR: %v\n", u, err)
			continue
		}
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		_ = resp.Body.Close()
		t.printf("--- %s\nHTTP %d\n%s\n", u, resp.StatusCode, strings.TrimSpace(string(body)))
		if summary == "Public IP: unknown" {
			fields := traceFields(string(body))
			if fields["ip"] != "" {
				summary = fmt.Sprintf("Public IP: %s, Cloudflare location: %s %s", fields["ip"], fields["colo"], fields["loc"])
			}
		}
	}
	return result{summary: summary}
}

func traceFields(body string) map[string]string {
	fields := map[string]string{}
	for _, line := range strings.Split(body, "\n") {
		if k, v, ok := strings.Cut(strings.TrimSpace(line), "="); ok {
			fields[k] = v
		}
	}
	return fields
}

func sniCheck(t *text) result {
	t.printf("Any HTTP code, including 403, means the network path works. A timeout means it is blocked.\n")
	t.printf("If only maac.cresclab.com fails on the same IP, the network filters by domain name.\n\n")
	var failures []string
	sni := sniResult{}
	for _, ip := range sniIPs {
		for _, name := range sniNames {
			detail, status := httpProbe("https://"+name+"/cdn-cgi/trace", probeOptions{dialIP: ip})
			t.printf("ip=%-15s sni=%-20s %s\n", ip, name, strings.ReplaceAll(detail, "\n", " | "))
			sni.set(ip, name, !failed(status))
			if failed(status) {
				failures = append(failures, name+"@"+ip)
			}
		}
	}
	if len(failures) == 0 {
		return result{summary: "SNI comparison: all reachable", sni: sni}
	}
	return result{summary: "SNI comparison: failed " + strings.Join(failures, ", "), sni: sni}
}

func mtuCheck(t *text) result {
	t.printf("1472 failing alone is normal on PPPoE lines such as HiNet fiber. If 1300 or 1200 fail\n")
	t.printf("while plain ping works, large packets are dropped on the path and TLS can stall.\n")
	t.printf("If no size gets replies, ICMP is blocked and this check says nothing.\n")
	var status []string
	for _, size := range mtuSizes {
		c := mtuPingCommand(mtuTarget, size)
		out, ok := runCommandOK(c.name, c.args...)
		t.printf("--- payload %d bytes\n%s", size, out)
		mark := "no-reply"
		if ok {
			mark = "reply"
		}
		status = append(status, fmt.Sprintf("%d=%s", size, mark))
	}
	return result{summary: "MTU ping: " + strings.Join(status, " ")}
}

func repeatCheck(t *text) result {
	ok := 0
	for i := 1; i <= repeatCount; i++ {
		detail, status := httpProbe(repeatURL, probeOptions{})
		t.printf("#%-2d %s\n", i, strings.ReplaceAll(detail, "\n", " | "))
		if !failed(status) {
			ok++
		}
	}
	return result{summary: fmt.Sprintf("Repeated HTTPS: %d/%d succeeded", ok, repeatCount)}
}

func since(start, event time.Time) string {
	if event.IsZero() {
		return "-"
	}
	return event.Sub(start).Round(time.Millisecond).String()
}

// errKind classifies a network error for the summary line.
func errKind(err error) string {
	var netErr net.Error
	var dnsErr *net.DNSError
	switch {
	case errors.As(err, &dnsErr):
		return "dns"
	case errors.As(err, &netErr) && netErr.Timeout():
		return "timeout"
	case strings.Contains(err.Error(), "connection reset"), strings.Contains(err.Error(), "forcibly closed"):
		return "reset"
	case strings.Contains(err.Error(), "refused"):
		return "refused"
	default:
		return "error"
	}
}
