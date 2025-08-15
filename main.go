package main

import (
	"embed"
	"errors"
	"log"
	"os"
	"os/exec"
	"spoilr/backend"
	"strings"

	"github.com/sqweek/dialog"
	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed all:frontend/dist
var assets embed.FS

func showErrorDialog(title, message string) {
	log.Printf("FATAL ERROR: %s - %s", title, message)
	dialog.Message("%s", message).Title(title).Error()
	os.Exit(1)
}

func checkExecutable(name string) error {
	_, err := exec.LookPath(name)
	if err != nil {
		return err
	}
	return nil
}

func ensureFFmpeg() error {
	var missing []string

	if err := checkExecutable("ffmpeg"); err != nil {
		missing = append(missing, "ffmpeg")
	}

	if err := checkExecutable("ffprobe"); err != nil {
		missing = append(missing, "ffprobe")
	}

	if len(missing) > 0 {
		var errorMsg strings.Builder
		errorMsg.WriteString("The following FFmpeg components are missing:\n\n")

		for _, component := range missing {
			errorMsg.WriteString("• " + component + "\n")
		}

		errorMsg.WriteString("\nPlease install FFmpeg and ensure these components are available in your system PATH.")

		if len(missing) == 2 {
			errorMsg.WriteString("\n\nNote: Both ffmpeg and ffprobe are required for full functionality.")
		}

		return errors.New(errorMsg.String())
	}

	return nil
}

func main() {
	if err := ensureWebView2(); err != nil {
		showErrorDialog("WebView2 Required", err.Error())
		return
	}

	// Check for ffmpeg and ffprobe
	if err := ensureFFmpeg(); err != nil {
		showErrorDialog("FFmpeg Components Missing", err.Error())
		return
	}

	spoilerService := backend.NewSpoilerService()

	app := application.New(application.Options{
		Name:        "Spoilr",
		Description: "Advanced media analyzer with automatic screenshot generation and FastPic upload",
		Services: []application.Service{
			application.NewService(spoilerService),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})

	spoilerService.SetApp(app)

	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:             "Spoilr",
		EnableDragAndDrop: true,
		DisableResize:     true,
		BackgroundColour:  application.NewRGBA(0, 0, 0, 0),
		BackgroundType:    application.BackgroundTypeTranslucent,
		URL:               "/",
		Width:             1200,
		Height:            800,
		// MinWidth:          1200,
		// MinHeight:         800,
	})

	// Handle drag and drop events
	window.OnWindowEvent(events.Common.WindowFilesDropped, func(event *application.WindowEvent) {
		paths := event.Context().DroppedFiles()
		log.Printf("Files dropped: %v", paths)

		// Just add files to the list without processing
		err := spoilerService.AddMovies(paths)
		if err != nil {
			log.Printf("Error adding movies: %v", err)
		}
	})

	err := app.Run()
	if err != nil {
		log.Fatal(err)
	}
}
