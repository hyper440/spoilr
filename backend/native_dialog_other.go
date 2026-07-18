//go:build !windows

package backend

import (
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

func runDarwinDialog(script, title, message string) ([]byte, error) {
	command := exec.Command("osascript", "-e", script)
	command.Env = append(
		os.Environ(),
		"SPOILR_DIALOG_TITLE="+title,
		"SPOILR_DIALOG_MESSAGE="+message,
	)
	return command.Output()
}

func runLinuxDialog(kind, title, message string) error {
	if _, err := exec.LookPath("zenity"); err == nil {
		return exec.Command(
			"zenity",
			"--"+kind,
			"--title", title,
			"--text", message,
			"--no-wrap",
		).Run()
	}

	if _, err := exec.LookPath("kdialog"); err == nil {
		kdialogKind := map[string]string{
			"error":    "error",
			"info":     "msgbox",
			"question": "yesno",
		}[kind]
		return exec.Command("kdialog", "--"+kdialogKind, message, "--title", title).Run()
	}

	return exec.ErrNotFound
}

// ShowErrorDialog displays a native error dialog when the platform provides one.
func ShowErrorDialog(title, message string) {
	var err error
	switch runtime.GOOS {
	case "darwin":
		_, err = runDarwinDialog(
			`display alert (system attribute "SPOILR_DIALOG_TITLE") message (system attribute "SPOILR_DIALOG_MESSAGE") as critical`,
			title,
			message,
		)
	case "linux":
		err = runLinuxDialog("error", title, message)
	default:
		err = exec.ErrNotFound
	}

	if err != nil {
		log.Printf("%s: %s", title, message)
	}
}

// ShowInfoDialog displays a native information dialog when the platform provides one.
func ShowInfoDialog(title, message string) {
	var err error
	switch runtime.GOOS {
	case "darwin":
		_, err = runDarwinDialog(
			`display alert (system attribute "SPOILR_DIALOG_TITLE") message (system attribute "SPOILR_DIALOG_MESSAGE") as informational`,
			title,
			message,
		)
	case "linux":
		err = runLinuxDialog("info", title, message)
	default:
		err = exec.ErrNotFound
	}

	if err != nil {
		log.Printf("%s: %s", title, message)
	}
}

// AskYesNoDialog displays a native question dialog when the platform provides one.
func AskYesNoDialog(title, message string) bool {
	switch runtime.GOOS {
	case "darwin":
		output, err := runDarwinDialog(
			`button returned of (display dialog (system attribute "SPOILR_DIALOG_MESSAGE") with title (system attribute "SPOILR_DIALOG_TITLE") buttons {"No", "Yes"} default button "Yes")`,
			title,
			message,
		)
		return err == nil && strings.TrimSpace(string(output)) == "Yes"
	case "linux":
		return runLinuxDialog("question", title, message) == nil
	default:
		log.Printf("%s: %s", title, message)
		return false
	}
}
