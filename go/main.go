// Command cresclab-netcheck collects DNS, TCP, TLS, HTTP, and traceroute results
// for Crescendo Lab services and writes them to a UTF-8 text file that customers
// can send back to support. It only reads information and changes no settings.
package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptrace"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const version = "2026-10-27"

var (
	serviceHosts    = []string{"maac.cresclab.com", "caac.cresclab.com", "api.cresclab.com", "api.caac.cresclab.com", "cdn.caac.cresclab.com"}
	controlHosts    = []string{"www.cloudflare.com", "www.google.com"}
	traceHosts      = []string{"maac.cresclab.com", "api.cresclab.com", "www.cloudflare.com"}
	cloudflareTrace = []string{"https://www.cloudflare.com/cdn-cgi/trace", "https://maac.cresclab.com/cdn-cgi/trace"}
	publicResolvers = []string{"1.1.1.1", "8.8.8.8"}
)

const (
	dialTimeout    = 10 * time.Second
	requestTimeout = 20 * time.Second
	commandTimeout = 3 * time.Minute
)

func main() {
	start := time.Now()
	fmt.Println("Crescendo Lab network check")
	fmt.Println("This takes about 2 minutes. Please keep this window open.")
	fmt.Println()

	// Traceroute is the slowest step, so it runs in the background while the other checks run.
	traces := make([]<-chan string, len(traceHosts))
	for i, host := range traceHosts {
		traces[i] = runAsync(traceCommand(host))
	}

	r := &report{}
	r.printf("Crescendo Lab network check - cresclab-netcheck.exe %s\n", version)
	r.printf("Local time: %s\n", start.Format("2006-01-02 15:04:05 -0700"))
	r.printf("UTC time:   %s\n", start.UTC().Format("2006-01-02 15:04:05"))
	r.printf("OS: %s/%s\n\n", runtime.GOOS, runtime.GOARCH)
	r.printf("This file contains network adapter settings from ipconfig /all, proxy settings,\n")
	r.printf("the public IP address seen by Cloudflare, and connection test results for\n")
	r.printf("Crescendo Lab services. The program changes no settings.\n")

	fmt.Println("Collecting system and network settings ...")
	for _, c := range systemCommands() {
		r.section(c.title)
		r.printf("%s", runCommand(c.name, c.args...))
	}

	proxyURL, proxyNote := systemProxy()
	r.section("Proxy used for the extra HTTPS check")
	r.printf("%s\n", proxyNote)

	r.section("Cloudflare trace: public IP and Cloudflare location")
	for _, u := range cloudflareTrace {
		r.printf("--- %s\n%s\n", u, fetchBody(u))
	}

	hosts := append(append([]string{}, serviceHosts...), controlHosts...)
	for i, host := range hosts {
		fmt.Printf("[%d/%d] Checking %s ...\n", i+1, len(hosts), host)
		r.section(host)
		probeHost(r, host, proxyURL)
	}

	fmt.Println("Waiting for traceroute to finish ...")
	for i, host := range traceHosts {
		r.section("traceroute " + host)
		r.printf("%s", <-traces[i])
	}

	r.section("Finished")
	r.printf("Elapsed: %s\n", time.Since(start).Round(time.Second))

	path, err := writeReport(r.String(), start)
	fmt.Println()
	if err != nil {
		fmt.Println("Could not save the result file:", err)
	} else {
		fmt.Println("Done. Result file:")
		fmt.Println("  " + path)
		fmt.Println("Please send this file back to Crescendo Lab.")
		revealFile(path)
	}
	fmt.Println()
	fmt.Print("Press Enter to close this window ...")
	_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
}

type report struct{ b strings.Builder }

func (r *report) printf(format string, a ...any) { fmt.Fprintf(&r.b, format, a...) }

func (r *report) section(title string) {
	r.printf("\n==================== %s ====================\n", title)
}

func (r *report) String() string { return r.b.String() }

type command struct {
	title string
	name  string
	args  []string
}

func runCommand(name string, args ...string) string {
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	text := decodeConsoleOutput(out)
	if err != nil {
		text += fmt.Sprintf("[command %s failed: %v]\n", name, err)
	}
	return text
}

func runAsync(c command) <-chan string {
	ch := make(chan string, 1)
	go func() { ch <- runCommand(c.name, c.args...) }()
	return ch
}

func probeHost(r *report, host string, proxyURL *url.URL) {
	r.printf("--- DNS, system resolver\n")
	addrs, err := net.DefaultResolver.LookupIPAddr(context.Background(), host)
	if err != nil {
		r.printf("ERROR: %v\n", err)
	} else {
		r.printf("%s\n", joinAddrs(addrs))
	}
	for _, server := range publicResolvers {
		r.printf("--- DNS, %s\n%s\n", server, lookupVia(server, host))
	}

	r.printf("--- ping\n%s", runCommand(pingCommand(host).name, pingCommand(host).args...))

	r.printf("--- TCP connect to port 443\n")
	for i, a := range addrs {
		if i == 4 {
			break
		}
		r.printf("%s\n", tcpConnect(a.IP))
	}

	r.printf("--- TLS handshake\n%s\n", tlsHandshake(host))
	r.printf("--- HTTPS GET, direct connection\n%s\n", httpProbe("https://"+host+"/", nil))
	if proxyURL != nil {
		r.printf("--- HTTPS GET, through system proxy %s\n%s\n", proxyURL.Host, httpProbe("https://"+host+"/", proxyURL))
	}
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

func tcpConnect(ip net.IP) string {
	addr := net.JoinHostPort(ip.String(), "443")
	t := time.Now()
	conn, err := net.DialTimeout("tcp", addr, dialTimeout)
	elapsed := time.Since(t).Round(time.Millisecond)
	if err != nil {
		return fmt.Sprintf("%s FAILED after %s: %v", addr, elapsed, err)
	}
	conn.Close()
	return fmt.Sprintf("%s OK in %s", addr, elapsed)
}

// tlsHandshake reports the certificate the network actually presents. When a
// corporate firewall intercepts HTTPS, the issuer shows its own CA instead of
// the public one, so a failed verification is retried without verification to
// reveal that issuer.
func tlsHandshake(host string) string {
	dialer := &net.Dialer{Timeout: dialTimeout}
	t := time.Now()
	conn, err := tls.DialWithDialer(dialer, "tcp", net.JoinHostPort(host, "443"), &tls.Config{ServerName: host})
	if err == nil {
		defer conn.Close()
		return describeTLS(conn, time.Since(t), "verified")
	}
	result := fmt.Sprintf("FAILED after %s: %v", time.Since(t).Round(time.Millisecond), err)
	var verifyErr *tls.CertificateVerificationError
	if !errors.As(err, &verifyErr) {
		return result
	}
	t = time.Now()
	conn, err = tls.DialWithDialer(dialer, "tcp", net.JoinHostPort(host, "443"), &tls.Config{ServerName: host, InsecureSkipVerify: true})
	if err != nil {
		return result
	}
	defer conn.Close()
	return result + "\n" + describeTLS(conn, time.Since(t), "NOT trusted by this computer")
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

func httpProbe(target string, proxyURL *url.URL) string {
	transport := &http.Transport{
		DisableKeepAlives:   true,
		ForceAttemptHTTP2:   true,
		TLSHandshakeTimeout: dialTimeout,
		DialContext:         (&net.Dialer{Timeout: dialTimeout}).DialContext,
	}
	if proxyURL != nil {
		transport.Proxy = http.ProxyURL(proxyURL)
	}
	client := &http.Client{
		Transport:     transport,
		Timeout:       requestTimeout,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}

	var start, dnsDone, connDone, tlsDone, firstByte time.Time
	var remote string
	trace := &httptrace.ClientTrace{
		DNSDone:              func(httptrace.DNSDoneInfo) { dnsDone = time.Now() },
		ConnectDone:          func(_, addr string, _ error) { connDone = time.Now(); remote = addr },
		TLSHandshakeDone:     func(tls.ConnectionState, error) { tlsDone = time.Now() },
		GotFirstResponseByte: func() { firstByte = time.Now() },
	}
	req, err := http.NewRequest(http.MethodGet, target, nil)
	if err != nil {
		return "ERROR: " + err.Error()
	}
	req = req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
	req.Header.Set("User-Agent", "cresclab-netcheck/"+version)

	start = time.Now()
	resp, err := client.Do(req)
	timing := fmt.Sprintf("remote=%s dns=%s tcp=%s tls=%s ttfb=%s total=%s",
		remote, since(start, dnsDone), since(start, connDone), since(start, tlsDone), since(start, firstByte), time.Since(start).Round(time.Millisecond))
	if err != nil {
		return "ERROR: " + err.Error() + "\n" + timing
	}
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
	resp.Body.Close()
	return fmt.Sprintf("HTTP %d %s\n%s", resp.StatusCode, resp.Proto, timing)
}

// since formats the time from start to an httptrace event, or "-" when the
// event never happened, which shows the step where a connection stalled.
func since(start, event time.Time) string {
	if event.IsZero() {
		return "-"
	}
	return event.Sub(start).Round(time.Millisecond).String()
}

func fetchBody(target string) string {
	client := &http.Client{Timeout: requestTimeout}
	resp, err := client.Get(target)
	if err != nil {
		return "ERROR: " + err.Error()
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return fmt.Sprintf("HTTP %d\n%s", resp.StatusCode, strings.TrimSpace(string(body)))
}

func writeReport(text string, start time.Time) (string, error) {
	if runtime.GOOS == "windows" {
		text = strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\n", "\r\n")
	}
	name := "cresclab-netcheck-" + start.Format("20060102-150405") + ".txt"
	var lastErr error
	for _, dir := range outputDirs() {
		path := filepath.Join(dir, name)
		// The byte order mark lets Notepad on older Windows detect UTF-8.
		if lastErr = os.WriteFile(path, []byte("\xef\xbb\xbf"+text), 0o644); lastErr == nil {
			return path, nil
		}
	}
	return "", lastErr
}

func outputDirs() []string {
	var dirs []string
	if d := desktopDir(); d != "" {
		dirs = append(dirs, d)
	}
	if exe, err := os.Executable(); err == nil {
		dirs = append(dirs, filepath.Dir(exe))
	}
	if wd, err := os.Getwd(); err == nil {
		dirs = append(dirs, wd)
	}
	return append(dirs, os.TempDir())
}
