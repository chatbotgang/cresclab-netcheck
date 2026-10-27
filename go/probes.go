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

func probeHost(t *text, host string, proxyURL *url.URL) string {
	var status []string

	t.printf("--- DNS, system resolver\n")
	ctx, cancel := context.WithTimeout(context.Background(), dialTimeout)
	addrs, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	cancel()
	if err != nil {
		t.printf("ERROR: %v\n", err)
		status = append(status, "dns=FAIL")
	} else {
		t.printf("%s\n", joinAddrs(addrs))
		status = append(status, "dns=ok")
	}
	for _, server := range publicResolvers {
		t.printf("--- DNS, %s\n%s\n", server, lookupVia(server, host))
	}

	ping := pingCommand(host)
	t.printf("--- ping\n%s", runCommand(ping.name, ping.args...))

	t.printf("--- TCP connect to port 443\n")
	tcpOK, tcpTried := 0, 0
	for _, a := range addrs {
		if tcpTried == 4 {
			break
		}
		line, ok := tcpConnect(a.IP)
		t.printf("%s\n", line)
		tcpTried++
		if ok {
			tcpOK++
		}
	}
	if tcpTried == 0 {
		t.printf("Skipped: no address from the system resolver\n")
	}
	status = append(status, fmt.Sprintf("tcp=%d/%d", tcpOK, tcpTried))

	detail, tlsStatus := tlsHandshake(host)
	t.printf("--- TLS handshake\n%s\n", detail)
	status = append(status, "tls="+tlsStatus)

	detail, httpsStatus := httpProbe("https://"+host+"/", probeOptions{})
	t.printf("--- HTTPS GET, direct connection\n%s\n", detail)
	status = append(status, "https="+httpsStatus)

	if proxyURL != nil {
		detail, proxyStatus := httpProbe("https://"+host+"/", probeOptions{proxy: proxyURL})
		t.printf("--- HTTPS GET, through system proxy %s\n%s\n", proxyURL.Host, detail)
		status = append(status, "https-via-proxy="+proxyStatus)
	}

	detail, plainStatus := plainHTTP("http://" + host + "/")
	t.printf("--- HTTP port 80, first lines; a firewall block page shows up here\n%s\n", detail)
	status = append(status, "http80="+plainStatus)

	return host + ": " + strings.Join(status, " ")
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

func tcpConnect(ip net.IP) (string, bool) {
	addr := net.JoinHostPort(ip.String(), "443")
	start := time.Now()
	conn, err := net.DialTimeout("tcp", addr, dialTimeout)
	elapsed := time.Since(start).Round(time.Millisecond)
	if err != nil {
		return fmt.Sprintf("%s FAILED after %s: %v", addr, elapsed, err), false
	}
	conn.Close()
	return fmt.Sprintf("%s OK in %s", addr, elapsed), true
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
		defer conn.Close()
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
	defer conn.Close()
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
	resp.Body.Close()
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
	defer resp.Body.Close()
	var b strings.Builder
	fmt.Fprintf(&b, "HTTP %d %s\n", resp.StatusCode, resp.Proto)
	for _, h := range []string{"Server", "Location", "Content-Type", "Via", "X-Cache"} {
		if v := resp.Header.Get(h); v != "" {
			fmt.Fprintf(&b, "%s: %s\n", h, v)
		}
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1500))
	if s := strings.TrimSpace(string(body)); s != "" {
		fmt.Fprintf(&b, "\n%s\n", s)
	}
	return strings.TrimRight(b.String(), "\n"), fmt.Sprint(resp.StatusCode)
}

func cloudflareTraceCheck(t *text) string {
	summary := "Public IP: unknown"
	for _, u := range cloudflareTrace {
		client := &http.Client{Timeout: requestTimeout}
		resp, err := client.Get(u)
		if err != nil {
			t.printf("--- %s\nERROR: %v\n", u, err)
			continue
		}
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		resp.Body.Close()
		t.printf("--- %s\nHTTP %d\n%s\n", u, resp.StatusCode, strings.TrimSpace(string(body)))
		if summary == "Public IP: unknown" {
			fields := traceFields(string(body))
			if fields["ip"] != "" {
				summary = fmt.Sprintf("Public IP: %s, Cloudflare location: %s %s", fields["ip"], fields["colo"], fields["loc"])
			}
		}
	}
	return summary
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

func sniCheck(t *text) string {
	t.printf("Any HTTP code, including 403, means the network path works. A timeout means it is blocked.\n")
	t.printf("If only maac.cresclab.com fails on the same IP, the network filters by domain name.\n\n")
	var failed []string
	for _, ip := range sniIPs {
		for _, name := range sniNames {
			detail, status := httpProbe("https://"+name+"/cdn-cgi/trace", probeOptions{dialIP: ip})
			t.printf("ip=%-15s sni=%-20s %s\n", ip, name, strings.ReplaceAll(detail, "\n", " | "))
			if strings.HasPrefix(status, "FAIL") {
				failed = append(failed, name+"@"+ip)
			}
		}
	}
	if len(failed) == 0 {
		return "SNI comparison: all reachable"
	}
	return "SNI comparison: failed " + strings.Join(failed, ", ")
}

func mtuCheck(t *text) string {
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
	return "MTU ping: " + strings.Join(status, " ")
}

func repeatCheck(t *text) string {
	ok := 0
	for i := 1; i <= repeatCount; i++ {
		detail, status := httpProbe(repeatURL, probeOptions{})
		t.printf("#%-2d %s\n", i, strings.ReplaceAll(detail, "\n", " | "))
		if !strings.HasPrefix(status, "FAIL") {
			ok++
		}
	}
	return fmt.Sprintf("Repeated HTTPS: %d/%d succeeded", ok, repeatCount)
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
