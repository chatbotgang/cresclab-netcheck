package main

import (
	"strings"
	"testing"
)

func TestParseProxyServer(t *testing.T) {
	tests := []struct{ in, want string }{
		{"proxy.corp:8080", "proxy.corp:8080"},
		{"http=web.corp:80;https=secure.corp:443", "secure.corp:443"},
		{"http=web.corp:80;ftp=ftp.corp:21", "web.corp:80"},
		{"ftp=ftp.corp:21", ""},
		{"", ""},
	}
	for _, tt := range tests {
		u := parseProxyServer(tt.in)
		got := ""
		if u != nil {
			got = u.Host
		}
		if got != tt.want {
			t.Errorf("parseProxyServer(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestParseScutilProxy(t *testing.T) {
	out := `<dictionary> {
  ExceptionsList : <array> {
    0 : *.local
  }
  HTTPSEnable : 1
  HTTPSPort : 3128
  HTTPSProxy : proxy.corp
  ProxyAutoConfigEnable : 1
  ProxyAutoConfigURLString : http://wpad.corp/proxy.pac
}`
	u, note := parseScutilProxy(out)
	if u == nil || u.Host != "proxy.corp:3128" {
		t.Fatalf("proxy = %v, want proxy.corp:3128", u)
	}
	if want := "PAC script configured but not evaluated: http://wpad.corp/proxy.pac"; !strings.Contains(note, want) {
		t.Errorf("note = %q, want it to mention %q", note, want)
	}

	if u, _ := parseScutilProxy("<dictionary> {\n  HTTPSEnable : 0\n}"); u != nil {
		t.Errorf("disabled proxy = %v, want nil", u)
	}
}
