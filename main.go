package main

import (
	"changeme/backend"
	"context"
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

// Wails uses Go's `embed` package to embed the frontend files into the binary.
// Any files in the frontend/dist folder will be embedded into the binary and
// made available to the frontend.
// See https://pkg.go.dev/embed for more information.

//go:embed all:frontend/dist
var assets embed.FS

// main function serves as the application's entry point. It initializes the application, creates a window,
// and starts the application.
func main() {
	// Create the spoiler service
	spoilerService := backend.NewSpoilerService()

	// Create a new Wails application by providing the necessary options.
	// Variables 'Name' and 'Description' are for application metadata.
	// 'Assets' configures the asset server with the 'FS' variable pointing to the frontend files.
	// 'Services' is a list of Go struct instances. The frontend has access to the methods of these instances.
	// 'Mac' options tailor the application when running on macOS.
	app := application.New(application.Options{
		Name:        "Spoiler List Generator",
		Description: "A media file analyzer that generates spoiler lists for torrents",
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

	// Set the app instance in the service so it can emit events
	spoilerService.SetApp(app)

	// Create a new window with the necessary options.
	// 'Title' is the title of the window.
	// 'Mac' options tailor the window when running on macOS.
	// 'BackgroundColour' is the background colour of the window.
	// 'URL' is the URL that will be loaded into the webview.
	window := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:             "Spoiler List Generator",
		EnableDragAndDrop: true,
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		BackgroundColour: application.NewRGB(27, 38, 54),
		URL:              "/",
		Width:            1200,
		Height:           800,
	})

	// Handle drag and drop events
	window.OnWindowEvent(events.Common.WindowFilesDropped, func(event *application.WindowEvent) {
		paths := event.Context().DroppedFiles()
		log.Printf("Files dropped: %v", paths)

		// Process dropped files in a goroutine to avoid blocking UI
		go func() {
			ctx := context.Background()

			// Expand paths to handle directories
			expandedPaths, err := spoilerService.GetExpandedFilePaths(paths)
			if err != nil {
				log.Printf("Error expanding file paths: %v", err)
				spoilerService.EmitProgress(backend.ProcessProgress{
					Current:   0,
					Total:     0,
					Message:   "Error processing files",
					Error:     err.Error(),
					Completed: true,
				})
				return
			}

			if len(expandedPaths) == 0 {
				spoilerService.EmitProgress(backend.ProcessProgress{
					Current:   0,
					Total:     0,
					Message:   "No video files found",
					Completed: true,
				})
				return
			}

			// Process the files
			err = spoilerService.ProcessFiles(ctx, expandedPaths)
			if err != nil {
				log.Printf("Error processing files: %v", err)
			}
		}()
	})

	// Run the application. This blocks until the application has been exited.
	err := app.Run()

	// If an error occurred while running the application, log it and exit.
	if err != nil {
		log.Fatal(err)
	}
}
