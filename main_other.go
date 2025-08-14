//go:build !windows

package main

func ensureWebView2() error {
	// No-op on macOS and Linux
	return nil
}
