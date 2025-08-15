//go:build windows

package main

import (
	_ "embed"
	"errors"
	"slg/backend"
)

//go:embed build/windows/nsis/MicrosoftEdgeWebview2Setup.exe
var webview2Installer []byte

func ensureWebView2() error {
	handler := backend.NewWebView2Handler(webview2Installer)
	_, err := handler.EnsureWebView2Available()
	if err != nil {
		return errors.New("WebView2 runtime is not available: " + err.Error())
	}
	return nil
}
