package main

import (
	"changeme/backend"
	"context"
	"embed"
	"log"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	spoilerService := backend.NewSpoilerService()

	app := application.New(application.Options{
		Name:        "Spoiler List Generator",
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
		Title:             "Spoiler List Generator",
		EnableDragAndDrop: true,
		Mac: application.MacWindow{
			InvisibleTitleBarHeight: 50,
			Backdrop:                application.MacBackdropTranslucent,
			TitleBar:                application.MacTitleBarHiddenInset,
		},
		BackgroundColour: application.NewRGBA(0, 0, 0, 0),
		BackgroundType:   application.BackgroundTypeTranslucent,
		URL:              "/",
		Width:            1400,
		Height:           900,
		MinWidth:         1200,
		MinHeight:        800,
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

			log.Printf("Processing %d video files", len(expandedPaths))

			// Add files to the list first (this makes them show up immediately)
			spoilerService.AddPendingFiles(expandedPaths)

			// Then process the files
			err = spoilerService.ProcessFiles(ctx, expandedPaths)
			if err != nil {
				log.Printf("Error processing files: %v", err)
			}
		}()
	})

	err := app.Run()
	if err != nil {
		log.Fatal(err)
	}
}
