package img_uploaders

import (
	"context"
	"image"
	"image/color"
	"image/png"
	"log"
	"os"
	"path/filepath"
	"spoilr/backend/img_uploaders"
	"strings"
	"testing"
	"time"

	"github.com/joho/godotenv"
)

func init() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}
}

// createTestImage generates a simple test PNG image
func createTestImage(path string) error {
	// Create a simple 100x100 red square
	img := image.NewRGBA(image.Rect(0, 0, 100, 100))

	// Fill with red color
	for y := range 100 {
		for x := range 100 {
			img.Set(x, y, color.RGBA{255, 0, 0, 255})
		}
	}

	// Create the file
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	// Encode as PNG
	return png.Encode(file, img)
}

func TestHamsterService_UploadImage(t *testing.T) {
	// Skip test if credentials not provided
	email := os.Getenv("HAMSTER_EMAIL")
	password := os.Getenv("HAMSTER_PASSWORD")

	if email == "" || password == "" {
		t.Skip("HAMSTER_EMAIL and HAMSTER_PASSWORD environment variables required")
	}

	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "hamster_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test image
	testImagePath := filepath.Join(tempDir, "test_image.png")
	if err := createTestImage(testImagePath); err != nil {
		t.Fatalf("Failed to create test image: %v", err)
	}

	// Initialize service
	service := img_uploaders.NewHamsterService(email, password)
	if service == nil {
		t.Fatal("Failed to create HamsterService")
	}

	// Test upload with context timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Perform upload
	result, err := service.UploadImage(ctx, testImagePath)
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}

	// Validate result
	if result == nil {
		t.Fatal("Upload result is nil")
	}

	if result.URL == "" {
		t.Error("URL is empty")
	}

	if result.ViewerURL == "" {
		t.Error("ViewerURL is empty")
	}

	if result.ThumbnailURL == "" {
		t.Error("ThumbnailURL is empty")
	}

	if result.BBThumb == "" {
		t.Error("BBThumb is empty")
	}

	if result.BBBig == "" {
		t.Error("BBBig is empty")
	}

	// Validate URLs contain hamster.is domain
	if !strings.Contains(result.URL, "hamster.is") {
		t.Errorf("URL doesn't contain hamster.is: %s", result.URL)
	}

	if !strings.Contains(result.ViewerURL, "hamster.is") {
		t.Errorf("ViewerURL doesn't contain hamster.is: %s", result.ViewerURL)
	}

	if !strings.Contains(result.ThumbnailURL, "hamster.is") {
		t.Errorf("ThumbnailURL doesn't contain hamster.is: %s", result.ThumbnailURL)
	}

	// Log results for manual verification
	t.Logf("Upload successful!")
	t.Logf("URL: %s", result.URL)
	t.Logf("Viewer: %s", result.ViewerURL)
	t.Logf("Thumbnail: %s", result.ThumbnailURL)
	t.Logf("BBThumb: %s", result.BBThumb)
	t.Logf("BBBig: %s", result.BBBig)
}

func TestHamsterService_UploadMultipleFormats(t *testing.T) {
	email := os.Getenv("HAMSTER_EMAIL")
	password := os.Getenv("HAMSTER_PASSWORD")

	if email == "" || password == "" {
		t.Skip("HAMSTER_EMAIL and HAMSTER_PASSWORD environment variables required")
	}

	service := img_uploaders.NewHamsterService(email, password)
	if service == nil {
		t.Fatal("Failed to create HamsterService")
	}

	tempDir, err := os.MkdirTemp("", "hamster_formats_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Test different file extensions
	formats := []string{"test1.png", "test2.jpg", "test3.jpeg"}

	ctx := context.Background()

	for _, filename := range formats {
		t.Run(filename, func(t *testing.T) {
			testPath := filepath.Join(tempDir, filename)
			if err := createTestImage(testPath); err != nil {
				t.Fatalf("Failed to create test image %s: %v", filename, err)
			}

			result, err := service.UploadImage(ctx, testPath)
			if err != nil {
				t.Errorf("Upload failed for %s: %v", filename, err)
				return
			}

			if result.URL == "" {
				t.Errorf("Empty URL for %s", filename)
			}

			t.Logf("%s uploaded successfully: %s", filename, result.URL)
		})
	}
}

func TestHamsterService_InvalidFile(t *testing.T) {
	email := os.Getenv("HAMSTER_EMAIL")
	password := os.Getenv("HAMSTER_PASSWORD")

	if email == "" || password == "" {
		t.Skip("HAMSTER_EMAIL and HAMSTER_PASSWORD environment variables required")
	}

	service := img_uploaders.NewHamsterService(email, password)
	ctx := context.Background()
	service.Login(ctx)

	// Test non-existent file
	_, err := service.UploadImage(ctx, "/non/existent/file.png")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}

	// Test empty file
	tempDir, err := os.MkdirTemp("", "hamster_invalid_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	emptyFile := filepath.Join(tempDir, "empty.png")
	if file, err := os.Create(emptyFile); err == nil {
		file.Close()
		_, err = service.UploadImage(ctx, emptyFile)
		if err == nil {
			t.Error("Expected error for empty file")
		}
	}
}
