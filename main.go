package main

import (
	"changeme/backend"
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
