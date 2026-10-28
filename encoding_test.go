package main

import "testing"

func TestDecodeCodePageBig5(t *testing.T) {
	// "要求等候逾時。" in Big5, as tracert prints it on Traditional Chinese Windows.
	big5 := []byte{0xad, 0x6e, 0xa8, 0x44, 0xb5, 0xa5, 0xad, 0xd4, 0xb9, 0x4f, 0xae, 0xc9, 0xa1, 0x43}
	if got, want := decodeCodePage(big5, 950), "要求等候逾時。"; got != want {
		t.Errorf("decodeCodePage(big5, 950) = %q, want %q", got, want)
	}
	if got, want := decodeCodePage([]byte("Request timed out."), 950), "Request timed out."; got != want {
		t.Errorf("ASCII input changed: got %q, want %q", got, want)
	}
}
