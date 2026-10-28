// Command cresclab-netcheck collects DNS, TCP, TLS, HTTP, MTU, and traceroute
// results for Crescendo Lab services and writes them to a UTF-8 text file that
// customers can send back to support. It only reads information and changes no
// settings.
//
// Checks run concurrently, each with its own timeout. A check that fails,
// hangs, or panics is recorded in the report and never stops the others, and
// the report is written to disk as each check finishes, so closing the window
// early still leaves partial results.
package main

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"runtime"
	"strings"
	"time"
)

// version is set at release time with -ldflags "-X main.version=...".
var version = "dev"

var (
	serviceHosts    = []string{"maac.cresclab.com", "caac.cresclab.com", "api.cresclab.com", "api.caac.cresclab.com", "cdn.caac.cresclab.com"}
	controlHosts    = []string{"www.cloudflare.com", "www.google.com"}
	traceHosts      = []string{"maac.cresclab.com", "api.cresclab.com", "www.cloudflare.com"}
	cloudflareTrace = []string{"https://www.cloudflare.com/cdn-cgi/trace", "https://maac.cresclab.com/cdn-cgi/trace"}
	publicResolvers = []string{"1.1.1.1", "8.8.8.8"}

	// Fixed Cloudflare edge IPs as of 2026-10-27, so the SNI and MTU checks still
	// run when DNS is blocked. 104.18.x serves Crescendo Lab; 104.16.123.96
	// normally serves www.cloudflare.com.
	sniIPs    = []string{"104.18.4.188", "104.18.5.188", "104.16.123.96"}
	sniNames  = []string{"maac.cresclab.com", "www.cloudflare.com"}
	mtuTarget = "104.18.4.188"
	mtuSizes  = []int{1472, 1400, 1300, 1200}

	repeatURL   = "https://maac.cresclab.com/cdn-cgi/trace"
	repeatCount = 10
)

const (
	dialTimeout    = 10 * time.Second
	requestTimeout = 15 * time.Second
	commandTimeout = 3 * time.Minute
	overallTimeout = 6 * time.Minute
)

// check is one independent diagnostic. run writes details to t and returns a
// one-line summary for the end of the report, or "" for none.
type check struct {
	title string
	run   func(t *text) string
}

type result struct{ body, summary string }

func main() {
	start := time.Now()
	fmt.Println("Crescendo Lab network check")
	fmt.Println("This takes about 2 minutes. Please keep this window open.")
	fmt.Println("Some checks may show errors or timeouts; that is expected and is part of the result.")
	fmt.Println()

	r := newReport("cresclab-netcheck-"+start.Format("20060102-150405")+".txt", outputDirs())
	r.printf("Crescendo Lab network check - cresclab-netcheck %s\n", version)
	r.printf("Local time: %s\n", start.Format("2006-01-02 15:04:05 -0700"))
	r.printf("UTC time:   %s\n", start.UTC().Format("2006-01-02 15:04:05"))
	r.printf("OS: %s/%s\n\n", runtime.GOOS, runtime.GOARCH)
	r.printf("This file contains network adapter settings, DNS and proxy settings,\n")
	r.printf("the public IP address seen by Cloudflare, and connection test results for\n")
	r.printf("Crescendo Lab services. The program changes no settings.\n")
	r.flush()

	// Host checks reuse the browser proxy, so it is detected before the others start.
	var proxyURL *url.URL
	proxyCheck := check{"Proxy used for the extra HTTPS check", func(t *text) string {
		var note string
		proxyURL, note = systemProxy()
		t.printf("%s\n", note)
		return ""
	}}
	checks := []check{proxyCheck}
	pending := []<-chan result{done(runCheck(proxyCheck))}

	var later []check
	for _, c := range systemCommands() {
		later = append(later, commandCheck(c))
	}
	later = append(later, check{"Cloudflare trace: public IP and Cloudflare location", cloudflareTraceCheck})
	for _, host := range append(append([]string{}, serviceHosts...), controlHosts...) {
		later = append(later, check{host, func(t *text) string { return probeHost(t, host, proxyURL) }})
	}
	later = append(later,
		check{"SNI comparison: same IP, different domain names", sniCheck},
		check{"MTU: ping " + mtuTarget + " with Don't Fragment", mtuCheck},
		check{"Repeated HTTPS requests: " + repeatURL, repeatCheck},
	)
	for _, host := range traceHosts {
		later = append(later, commandCheck(command{"traceroute " + host, traceCommand(host).name, traceCommand(host).args}))
	}
	for _, c := range later {
		checks = append(checks, c)
		pending = append(pending, startCheck(c))
	}

	// Results are written in a fixed order as soon as each one is ready.
	deadline := time.After(overallTimeout)
	expired := false
	var summaries []string
	for i, c := range checks {
		var res result
		ready := false
		if expired {
			// After the deadline, keep results that are already done but wait for nothing.
			select {
			case res = <-pending[i]:
				ready = true
			default:
			}
		} else {
			select {
			case res = <-pending[i]:
				ready = true
			case <-deadline:
				expired = true
			}
		}
		if !ready {
			res = result{"[did not finish within " + overallTimeout.String() + "]\n", c.title + ": did not finish"}
		}
		fmt.Printf("[%d/%d] %s\n", i+1, len(checks), c.title)
		r.section(c.title)
		r.printf("%s", res.body)
		r.flush()
		if res.summary != "" {
			summaries = append(summaries, res.summary)
		}
	}

	r.section("Summary")
	r.printf("http80 fails on api and cdn hosts by design; they do not serve port 80.\n\n")
	for _, s := range summaries {
		r.printf("%s\n", s)
	}
	r.printf("\nElapsed: %s\n", time.Since(start).Round(time.Second))
	r.flush()
	path, err := r.close()

	fmt.Println()
	fmt.Println("Summary:")
	for _, s := range summaries {
		fmt.Println("  " + s)
	}
	fmt.Println()
	if err != nil {
		fmt.Println("Could not save the result file:", err)
		fmt.Println("Please take a screenshot of this window and send it to Crescendo Lab.")
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

// runCheck runs c and turns a panic into a recorded failure.
func runCheck(c check) (res result) {
	t := &text{}
	defer func() {
		if p := recover(); p != nil {
			t.printf("[this check stopped unexpectedly: %v]\n", p)
			res = result{t.String(), c.title + ": check stopped unexpectedly"}
		}
	}()
	summary := c.run(t)
	return result{t.String(), summary}
}

func startCheck(c check) <-chan result {
	ch := make(chan result, 1)
	go func() { ch <- runCheck(c) }()
	return ch
}

func done(res result) <-chan result {
	ch := make(chan result, 1)
	ch <- res
	return ch
}

func commandCheck(c command) check {
	return check{c.title, func(t *text) string {
		t.printf("%s", runCommand(c.name, c.args...))
		return ""
	}}
}

// text is a string builder with printf.
type text struct{ strings.Builder }

func (t *text) printf(format string, a ...any) { fmt.Fprintf(&t.Builder, format, a...) }
