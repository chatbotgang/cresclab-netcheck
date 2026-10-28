//go:build darwin || linux

package main

// Unix tools write UTF-8, so their output needs no code page conversion.
func consoleCodePage() uint32 { return 65001 }
