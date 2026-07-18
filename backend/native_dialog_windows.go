//go:build windows

package backend

import (
	"log"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	messageBoxOK            = 0x00000000
	messageBoxYesNo         = 0x00000004
	messageBoxIconError     = 0x00000010
	messageBoxIconQuestion  = 0x00000020
	messageBoxIconInfo      = 0x00000040
	messageBoxSetForeground = 0x00010000
	messageBoxResultYes     = 6
)

var messageBoxProc = windows.NewLazySystemDLL("user32.dll").NewProc("MessageBoxW")

func showWindowsDialog(title, message string, flags uintptr) uintptr {
	titlePtr, err := windows.UTF16PtrFromString(title)
	if err != nil {
		log.Printf("Unable to encode dialog title %q: %v", title, err)
		return 0
	}

	messagePtr, err := windows.UTF16PtrFromString(message)
	if err != nil {
		log.Printf("Unable to encode dialog message: %v", err)
		return 0
	}

	result, _, callErr := messageBoxProc.Call(
		0,
		uintptr(unsafe.Pointer(messagePtr)),
		uintptr(unsafe.Pointer(titlePtr)),
		flags|messageBoxSetForeground,
	)
	if result == 0 {
		log.Printf("Unable to show native dialog: %v", callErr)
	}
	return result
}

// ShowErrorDialog displays a native blocking error dialog.
func ShowErrorDialog(title, message string) {
	showWindowsDialog(title, message, messageBoxOK|messageBoxIconError)
}

// ShowInfoDialog displays a native blocking information dialog.
func ShowInfoDialog(title, message string) {
	showWindowsDialog(title, message, messageBoxOK|messageBoxIconInfo)
}

// AskYesNoDialog displays a native blocking question dialog.
func AskYesNoDialog(title, message string) bool {
	return showWindowsDialog(title, message, messageBoxYesNo|messageBoxIconQuestion) == messageBoxResultYes
}
