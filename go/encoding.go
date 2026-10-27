package main

import (
	"net/url"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/encoding/traditionalchinese"
)

// codePageEncodings maps Windows console code pages to decoders. Windows tools
// such as tracert write localized text in the console code page, for example
// Big5 (950) on Traditional Chinese Windows.
var codePageEncodings = map[uint32]encoding.Encoding{
	437: charmap.CodePage437,
	850: charmap.CodePage850,
	932: japanese.ShiftJIS,
	936: simplifiedchinese.GBK,
	949: korean.EUCKR,
	950: traditionalchinese.Big5,
}

func decodeCodePage(b []byte, codePage uint32) string {
	if utf8.Valid(b) {
		return string(b)
	}
	enc, ok := codePageEncodings[codePage]
	if !ok {
		return string(b)
	}
	s, err := enc.NewDecoder().Bytes(b)
	if err != nil {
		return string(b)
	}
	return string(s)
}

func decodeConsoleOutput(b []byte) string { return decodeCodePage(b, consoleCodePage()) }

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
	if !strings.Contains(server, "://") {
		server = "http://" + server
	}
	u, err := url.Parse(server)
	if err != nil || u.Host == "" {
		return nil
	}
	return u
}
