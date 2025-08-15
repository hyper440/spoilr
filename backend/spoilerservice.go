package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v3/pkg/application"
)

type SpoilerService struct {
	app                 *application.App
	movies              []Movie
	settings            AppSettings
	template            string
	processing          bool
	cancelCtx           context.Context
	cancelFn            context.CancelFunc
	screenshotSemaphore chan struct{} // Limits concurrent screenshot generation
	uploadSemaphore     chan struct{} // Limits concurrent uploads
	configManager       *ConfigService
}

func NewSpoilerService() *SpoilerService {
	configManager := NewConfigService()
	config := configManager.GetConfig()

	service := &SpoilerService{
		movies: make([]Movie, 0),
		settings: AppSettings{
			ScreenshotCount:          config.ScreenshotCount,
			FastpicSID:               config.FastpicSID,
			ScreenshotQuality:        config.ScreenshotQuality,
			MaxConcurrentScreenshots: config.MaxConcurrentScreenshots,
			MaxConcurrentUploads:     config.MaxConcurrentUploads,
		},
		template:      config.Template,
		processing:    false,
		configManager: configManager,
	}

	if service.template == "" {
		service.template = service.GetDefaultTemplate()
	}

	service.initSemaphores()
	return service
}

func (s *SpoilerService) initSemaphores() {
	s.screenshotSemaphore = make(chan struct{}, s.settings.MaxConcurrentScreenshots)
	s.uploadSemaphore = make(chan struct{}, s.settings.MaxConcurrentUploads)
}

func (s *SpoilerService) SetApp(app *application.App) {
	s.app = app
}

func (s *SpoilerService) GetState() AppState {
	return AppState{
		Processing: s.processing,
		Movies:     s.movies,
	}
}

func (s *SpoilerService) emitState() {
	if s.app != nil {
		state := s.GetState()
		s.app.Event.Emit("state", state)
	}
}

func (s *SpoilerService) GetDefaultTemplate() string {
	return `[spoiler="%FILE_NAME%"]
File: %FILE_NAME%
Size: %FILE_SIZE%
Duration: %DURATION%
Resolution: %WIDTH%x%HEIGHT%
Video: %VIDEO_CODEC% @ %VIDEO_BIT_RATE%
Audio: %AUDIO_CODEC% @ %AUDIO_BIT_RATE%
Overall Bitrate: %BIT_RATE%

%THUMBNAIL%

%SCREENSHOTS%
[/spoiler]`
}

func (s *SpoilerService) updateMovieByID(id string, updateFn func(*Movie)) bool {
	for i := range s.movies {
		if s.movies[i].ID == id {
			updateFn(&s.movies[i])
			return true
		}
	}
	return false
}

func (s *SpoilerService) getMovieByID(id string) (Movie, bool) {
	for _, movie := range s.movies {
		if movie.ID == id {
			return movie, true
		}
	}
	return Movie{}, false
}

func (s *SpoilerService) extractMediaInfo(movie *Movie, mediaInfo MediaInfo) {
	// Extract basic info
	if duration, ok := mediaInfo.General["duration"]; ok {
		if dur, err := strconv.ParseFloat(duration, 64); err == nil {
			movie.Duration = formatDuration(time.Duration(dur * float64(time.Second)))
		}
	}

	if width, ok := mediaInfo.Video["width"]; ok {
		movie.Width = width
	}

	if height, ok := mediaInfo.Video["height"]; ok {
		movie.Height = height
	}

	// Extract bitrates
	if bitRate, ok := mediaInfo.Video["bit_rate"]; ok && bitRate != "" {
		movie.VideoBitRate = formatBitRate(bitRate)
	} else if overallBitRateStr, ok := mediaInfo.General["bit_rate"]; ok && overallBitRateStr != "" {
		if overall, err := strconv.ParseFloat(overallBitRateStr, 64); err == nil {
			estimatedVideoBitRate := overall * 0.8
			movie.VideoBitRate = formatBitRate(fmt.Sprintf("%.0f", estimatedVideoBitRate))
		}
	}

	if bitRate, ok := mediaInfo.Audio["bit_rate"]; ok && bitRate != "" {
		movie.AudioBitRate = formatBitRate(bitRate)
	} else if overallBitRateStr, ok := mediaInfo.General["bit_rate"]; ok && overallBitRateStr != "" {
		if overall, err := strconv.ParseFloat(overallBitRateStr, 64); err == nil {
			estimatedAudioBitRate := overall * 0.1
			movie.AudioBitRate = formatBitRate(fmt.Sprintf("%.0f", estimatedAudioBitRate))
		}
	}

	if codec, ok := mediaInfo.Video["codec_name"]; ok {
		movie.VideoCodec = codec
	}

	if codec, ok := mediaInfo.Audio["codec_name"]; ok {
		movie.AudioCodec = codec
	}

	if overallBitRate, ok := mediaInfo.General["bit_rate"]; ok {
		movie.BitRate = formatBitRate(overallBitRate)
	}

	// Store all parameters
	for key, value := range mediaInfo.General {
		movie.Params[fmt.Sprintf("%%General@%s%%", key)] = value
	}

	for key, value := range mediaInfo.Video {
		movie.Params[fmt.Sprintf("%%Video@%s%%", key)] = value
	}

	for key, value := range mediaInfo.Audio {
		movie.Params[fmt.Sprintf("%%Audio@%s%%", key)] = value
	}
}

func (s *SpoilerService) AddMovies(filePaths []string) error {
	expandedPaths, err := s.GetExpandedFilePaths(filePaths)
	if err != nil {
		return err
	}

	// First pass: add all movies with basic info and emit state
	var movieIDs []string
	for _, path := range expandedPaths {
		fileInfo, err := os.Stat(path)
		if err != nil {
			continue
		}

		movie := Movie{
			ID:              uuid.New().String(),
			FileName:        filepath.Base(path),
			FilePath:        path,
			FileSize:        formatFileSize(fileInfo.Size()),
			FileSizeBytes:   fileInfo.Size(),
			Params:          make(map[string]string),
			ScreenshotURLs:  make([]string, 0),
			ProcessingState: StateAnalyzingMedia,
		}

		s.movies = append(s.movies, movie)
		movieIDs = append(movieIDs, movie.ID)
	}
	s.emitState()

	// Second pass: process media info concurrently with state updates
	var wg sync.WaitGroup

	for _, movieID := range movieIDs {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()

			// Get movie by ID safely
			movie, exists := s.getMovieByID(id)
			if !exists || movie.ProcessingState != StateAnalyzingMedia {
				return
			}

			mediaInfo, err := s.getMediaInfoWithFFProbe(movie.FilePath)

			// Update movie using ID reference
			if err != nil {
				log.Printf("Failed to analyze media %s: %v", movie.FileName, err)
				s.updateMovieByID(id, func(m *Movie) {
					m.ProcessingState = StateError
					m.ProcessingError = err.Error()
				})
			} else {
				s.updateMovieByID(id, func(m *Movie) {
					s.extractMediaInfo(m, mediaInfo)
					m.ProcessingState = StatePending
				})
			}
			s.emitState() // Emit state after each file is processed
		}(movieID)
	}

	wg.Wait()
	return nil
}

func (s *SpoilerService) RemoveMovie(id string) {
	for i, movie := range s.movies {
		if movie.ID == id {
			s.movies = append(s.movies[:i], s.movies[i+1:]...)
			break
		}
	}
	s.emitState()
}

func (s *SpoilerService) ClearMovies() {
	s.movies = make([]Movie, 0)
	s.emitState()
}

func (s *SpoilerService) StartProcessing() error {
	if s.processing {
		return fmt.Errorf("processing already in progress")
	}

	pendingMovies := s.getPendingMovies()
	if len(pendingMovies) == 0 {
		return fmt.Errorf("no pending movies to process")
	}

	s.processing = true
	s.cancelCtx, s.cancelFn = context.WithCancel(context.Background())
	s.emitState()

	go func() {
		defer func() {
			s.processing = false
			// Reset any movies that are still in processing states back to pending
			for i := range s.movies {
				if s.movies[i].ProcessingState != StateCompleted && s.movies[i].ProcessingState != StateError {
					s.movies[i].ProcessingState = StatePending
					s.movies[i].ProcessingError = ""
				}
			}
			s.emitState()
			log.Println("Processing completed")
		}()

		err := s.processAllMoviesConcurrently()
		if err != nil {
			log.Printf("Processing error: %v", err)
		}
	}()

	return nil
}

func (s *SpoilerService) ReorderMovies(newOrder []string) error {
	movieMap := make(map[string]Movie)
	for _, movie := range s.movies {
		movieMap[movie.ID] = movie
	}

	for _, id := range newOrder {
		if _, exists := movieMap[id]; !exists {
			return fmt.Errorf("movie with ID %s not found", id)
		}
	}

	var reorderedMovies []Movie
	for _, id := range newOrder {
		reorderedMovies = append(reorderedMovies, movieMap[id])
	}

	s.movies = reorderedMovies
	s.emitState()
	return nil
}

func (s *SpoilerService) CancelProcessing() {
	if s.cancelFn != nil {
		s.cancelFn()
	}
	s.processing = false
	s.emitState()
}

func (s *SpoilerService) getPendingMovies() []Movie {
	var pending []Movie
	for _, movie := range s.movies {
		if movie.ProcessingState == StatePending {
			pending = append(pending, movie)
		}
	}
	return pending
}

// Improved concurrent processing
func (s *SpoilerService) processAllMoviesConcurrently() error {
	pendingMovies := s.getPendingMovies()
	if len(pendingMovies) == 0 {
		return nil
	}

	// Create temp directory for all media
	tempDir, err := os.MkdirTemp("", "media_processing_*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Get upload ID once for all movies
	fastpicService := NewFastpicService(s.settings.FastpicSID)
	err = fastpicService.getFastpicUploadID(s.cancelCtx)
	if err != nil {
		return fmt.Errorf("failed to get fastpic upload ID: %v", err)
	}

	log.Printf("Starting concurrent media processing for %d movies (screenshot limit: %d, upload limit: %d)",
		len(pendingMovies), s.settings.MaxConcurrentScreenshots, s.settings.MaxConcurrentUploads)

	// Process each movie concurrently
	var wg sync.WaitGroup
	for _, movie := range pendingMovies {
		wg.Add(1)
		go func(movie Movie) {
			defer wg.Done()
			s.processMovieWithLimits(movie, tempDir, fastpicService)
		}(movie)
	}

	wg.Wait()
	return nil
}

func (s *SpoilerService) processMovieWithLimits(movie Movie, tempDir string, fastpicService *FastpicService) {
	// Update status to waiting for screenshot slot
	s.updateMovieByID(movie.ID, func(m *Movie) {
		m.ProcessingState = StateWaitingForScreenshotSlot
	})
	s.emitState()

	// Create movie-specific temp directory
	movieTempDir := filepath.Join(tempDir, movie.ID)
	if err := os.MkdirAll(movieTempDir, 0755); err != nil {
		s.updateMovieByID(movie.ID, func(m *Movie) {
			m.ProcessingState = StateError
			m.ProcessingError = fmt.Sprintf("Failed to create temp directory: %v", err)
		})
		s.emitState()
		return
	}

	// Get video duration
	duration, err := s.getVideoDuration(movie.FilePath)
	if err != nil {
		s.updateMovieByID(movie.ID, func(m *Movie) {
			m.ProcessingState = StateError
			m.ProcessingError = fmt.Sprintf("Failed to get duration: %v", err)
		})
		s.emitState()
		return
	}

	// Generate media with proper concurrency limits
	thumbnailPath, screenshotPaths, err := s.generateMediaConcurrently(movie, movieTempDir, duration)
	if err != nil {
		s.updateMovieByID(movie.ID, func(m *Movie) {
			m.ProcessingState = StateError
			m.ProcessingError = fmt.Sprintf("Media generation failed: %v", err)
		})
		s.emitState()
		return
	}

	// Check if we have anything to upload
	if thumbnailPath == "" && len(screenshotPaths) == 0 {
		s.updateMovieByID(movie.ID, func(m *Movie) {
			m.ProcessingState = StateError
			m.ProcessingError = "No media generated"
		})
		s.emitState()
		return
	}

	// Update status to waiting for upload slot
	s.updateMovieByID(movie.ID, func(m *Movie) {
		m.ProcessingState = StateWaitingForUploadSlot
	})
	s.emitState()

	// Upload media with concurrency limits
	thumbnailURL, thumbnailBigURL, screenshotURLs, screenshotBigURLs, albumURL, err := s.uploadMediaConcurrently(movie, thumbnailPath, screenshotPaths, fastpicService)
	if err != nil {
		s.updateMovieByID(movie.ID, func(m *Movie) {
			m.ProcessingState = StateError
			m.ProcessingError = fmt.Sprintf("Upload failed: %v", err)
		})
		s.emitState()
		return
	}

	// Update movie with results
	s.updateMovieByID(movie.ID, func(m *Movie) {
		m.ThumbnailURL = thumbnailURL
		m.ThumbnailBigURL = thumbnailBigURL
		m.ScreenshotURLs = screenshotURLs
		m.ScreenshotBigURLs = screenshotBigURLs
		m.ScreenshotAlbum = albumURL
		m.ProcessingState = StateCompleted
	})
	s.emitState()

	log.Printf("Successfully processed movie: %s", movie.FileName)
}

// Generate thumbnail and screenshots with proper concurrency control
func (s *SpoilerService) generateMediaConcurrently(movie Movie, tempDir string, duration float64) (string, []string, error) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var generationStarted bool // Track if any generation has actually started

	var thumbnailPath string
	var screenshotPaths []string
	var screenshotErrors []error
	var thumbnailErr error

	// Generate thumbnail (if needed)
	if strings.Contains(s.template, "%THUMBNAIL%") || strings.Contains(s.template, "%THUMBNAIL_BIG%") {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// Acquire screenshot semaphore for thumbnail generation
			select {
			case s.screenshotSemaphore <- struct{}{}:
				defer func() { <-s.screenshotSemaphore }()

				// Update to generating state when first generation actually starts
				mu.Lock()
				if !generationStarted {
					generationStarted = true
					s.updateMovieByID(movie.ID, func(m *Movie) {
						m.ProcessingState = StateGeneratingScreenshots
					})
					s.emitState()
				}
				mu.Unlock()

				path, err := s.generateMovieThumbnail(movie.FilePath, tempDir)
				thumbnailPath = path
				thumbnailErr = err

			case <-s.cancelCtx.Done():
				thumbnailErr = s.cancelCtx.Err()
			}
		}()
	}

	// Generate screenshots concurrently (each screenshot gets its own goroutine)
	if s.settings.ScreenshotCount > 0 {
		screenshotPaths = make([]string, s.settings.ScreenshotCount)
		screenshotErrors = make([]error, s.settings.ScreenshotCount)

		interval := duration / float64(s.settings.ScreenshotCount+1)

		for i := 0; i < s.settings.ScreenshotCount; i++ {
			wg.Add(1)
			go func(index int) {
				defer wg.Done()

				// Acquire screenshot semaphore
				select {
				case s.screenshotSemaphore <- struct{}{}:
					defer func() { <-s.screenshotSemaphore }()

					// Update to generating state when first generation actually starts
					mu.Lock()
					if !generationStarted {
						generationStarted = true
						s.updateMovieByID(movie.ID, func(m *Movie) {
							m.ProcessingState = StateGeneratingScreenshots
						})
						s.emitState()
					}
					mu.Unlock()

					timestamp := interval * float64(index+1)
					outputPath := filepath.Join(tempDir, fmt.Sprintf("screenshot_%d.jpg", index+1))

					err := s.generateScreenshot(movie.FilePath, outputPath, timestamp)
					if err == nil {
						screenshotPaths[index] = outputPath
					}
					screenshotErrors[index] = err

				case <-s.cancelCtx.Done():
					screenshotErrors[index] = s.cancelCtx.Err()
				}
			}(i)
		}
	}

	wg.Wait()

	// Check for cancellation
	if s.cancelCtx.Err() != nil {
		return "", nil, s.cancelCtx.Err()
	}

	// Handle thumbnail error
	if thumbnailErr != nil {
		log.Printf("Failed to generate thumbnail for %s: %v", movie.FileName, thumbnailErr)
	}

	// Filter out failed screenshots
	var validScreenshots []string
	for i, path := range screenshotPaths {
		if path != "" {
			validScreenshots = append(validScreenshots, path)
		} else if i < len(screenshotErrors) && screenshotErrors[i] != nil {
			log.Printf("Failed to generate screenshot %d for %s: %v", i+1, movie.FileName, screenshotErrors[i])
		}
	}

	return thumbnailPath, validScreenshots, nil
}

// Upload media with proper concurrency control
func (s *SpoilerService) uploadMediaConcurrently(movie Movie, thumbnailPath string, screenshotPaths []string, fastpicService *FastpicService) (string, string, []string, []string, string, error) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var uploadStarted bool // Track if any upload has actually started

	var thumbnailURL, thumbnailBigURL, albumURL string
	screenshotURLs := make([]string, len(screenshotPaths))
	screenshotBigURLs := make([]string, len(screenshotPaths))

	baseFileName := strings.TrimSuffix(filepath.Base(movie.FilePath), filepath.Ext(movie.FilePath))

	// Upload thumbnail
	if thumbnailPath != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// Acquire upload semaphore
			select {
			case s.uploadSemaphore <- struct{}{}:
				defer func() { <-s.uploadSemaphore }()

				// Update to uploading state when first upload actually starts
				mu.Lock()
				if !uploadStarted {
					uploadStarted = true
					s.updateMovieByID(movie.ID, func(m *Movie) {
						m.ProcessingState = StateUploadingScreenshots
					})
					s.emitState()
				}
				mu.Unlock()

				fileName := fmt.Sprintf("%s_thumbnail.jpg", baseFileName)
				result, err := fastpicService.uploadToFastpic(s.cancelCtx, thumbnailPath, fileName)
				if err != nil {
					log.Printf("Failed to upload thumbnail for %s: %v", movie.FileName, err)
					return
				}

				mu.Lock()
				thumbnailURL = result.BBThumb
				thumbnailBigURL = result.BBBig
				if albumURL == "" {
					albumURL = result.AlbumLink
				}
				mu.Unlock()

			case <-s.cancelCtx.Done():
				return
			}
		}()
	}

	// Upload screenshots concurrently
	for i, screenshotPath := range screenshotPaths {
		wg.Add(1)
		go func(index int, path string) {
			defer wg.Done()

			// Acquire upload semaphore
			select {
			case s.uploadSemaphore <- struct{}{}:
				defer func() { <-s.uploadSemaphore }()

				// Update to uploading state when first upload actually starts
				mu.Lock()
				if !uploadStarted {
					uploadStarted = true
					s.updateMovieByID(movie.ID, func(m *Movie) {
						m.ProcessingState = StateUploadingScreenshots
					})
					s.emitState()
				}
				mu.Unlock()

				fileName := fmt.Sprintf("%s_screenshot_%d.jpg", baseFileName, index+1)
				result, err := fastpicService.uploadToFastpic(s.cancelCtx, path, fileName)
				if err != nil {
					log.Printf("Failed to upload screenshot %d for %s: %v", index+1, movie.FileName, err)
					return
				}

				mu.Lock()
				screenshotURLs[index] = result.BBThumb
				screenshotBigURLs[index] = result.BBBig
				if albumURL == "" {
					albumURL = result.AlbumLink
				}
				mu.Unlock()

			case <-s.cancelCtx.Done():
				return
			}
		}(i, screenshotPath)
	}

	wg.Wait()

	// Check for cancellation
	if s.cancelCtx.Err() != nil {
		return "", "", nil, nil, "", s.cancelCtx.Err()
	}

	// Filter out failed uploads
	var validScreenshotURLs, validScreenshotBigURLs []string
	for i := range screenshotURLs {
		if screenshotURLs[i] != "" {
			validScreenshotURLs = append(validScreenshotURLs, screenshotURLs[i])
			validScreenshotBigURLs = append(validScreenshotBigURLs, screenshotBigURLs[i])
		}
	}

	return thumbnailURL, thumbnailBigURL, validScreenshotURLs, validScreenshotBigURLs, albumURL, nil
}

func (s *SpoilerService) generateMovieThumbnail(videoPath, tempDir string) (string, error) {
	// Check if mtn is available before trying to use it
	if _, err := exec.LookPath("mtn"); err != nil {
		// Emit event that mtn is missing (only once per processing session)
		if s.app != nil {
			s.app.Event.Emit("mtn-missing", map[string]string{
				"message": "MTN (Movie Thumbnailer) is not installed or not found in PATH. Thumbnail generation will be skipped.",
			})
		}
		log.Printf("MTN not found, skipping thumbnail generation for %s", filepath.Base(videoPath))
		return "", nil // Return empty string to skip thumbnail
	}

	cmd := exec.CommandContext(s.cancelCtx, "mtn",
		"-b", "2", // border width
		"-w", "1200", // width
		"-c", "4", // columns
		"-r", "4", // rows
		"-g", "0", // gap between thumbnails
		"-k", "1C1C1C", // background color
		"-L", "4:2", // font size info:timestamp
		"-F", "F0FFFF:10", // font color:shadow
		"-O", tempDir, // output directory
		"-q",      // quiet
		videoPath, // input path (no -P flag needed)
	)

	err := cmd.Run()
	if err != nil {
		if s.cancelCtx.Err() != nil {
			return "", fmt.Errorf("thumbnail generation cancelled: %v", s.cancelCtx.Err())
		}
		return "", fmt.Errorf("mtn command failed: %v", err)
	}

	// mtn creates files based on video filename
	videoBasename := strings.TrimSuffix(filepath.Base(videoPath), filepath.Ext(videoPath))

	// List all files in temp directory to debug
	files, err := os.ReadDir(tempDir)
	if err != nil {
		return "", fmt.Errorf("failed to read temp directory: %v", err)
	}

	// Look for any .jpg files that match the pattern
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(strings.ToLower(file.Name()), ".jpg") {
			if strings.HasPrefix(file.Name(), videoBasename) {
				return filepath.Join(tempDir, file.Name()), nil
			}
		}
	}

	// If no exact match, take the first .jpg file (mtn should only create one)
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(strings.ToLower(file.Name()), ".jpg") {
			return filepath.Join(tempDir, file.Name()), nil
		}
	}

	return "", fmt.Errorf("thumbnail file not found after generation - no .jpg files in %s", tempDir)
}

func (s *SpoilerService) getVideoDuration(filePath string) (float64, error) {
	cmd := exec.CommandContext(s.cancelCtx, "ffprobe",
		"-v", "quiet",
		"-show_entries", "format=duration",
		"-of", "csv=p=0",
		filePath,
	)

	output, err := cmd.Output()
	if err != nil {
		if s.cancelCtx.Err() != nil {
			return 0, fmt.Errorf("duration check cancelled: %v", s.cancelCtx.Err())
		}
		return 0, fmt.Errorf("ffprobe command failed: %v", err)
	}

	duration, err := strconv.ParseFloat(strings.TrimSpace(string(output)), 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse duration: %v", err)
	}

	return duration, nil
}

func (s *SpoilerService) generateScreenshot(videoPath, outputPath string, timestamp float64) error {
	cmd := exec.CommandContext(s.cancelCtx, "ffmpeg",
		"-ss", fmt.Sprintf("%.2f", timestamp),
		"-i", videoPath,
		"-vframes", "1",
		"-q:v", fmt.Sprintf("%d", s.settings.ScreenshotQuality),
		"-y",
		outputPath,
	)

	err := cmd.Run()
	if err != nil {
		if s.cancelCtx.Err() != nil {
			return fmt.Errorf("screenshot generation cancelled: %v", s.cancelCtx.Err())
		}
		return fmt.Errorf("ffmpeg command failed: %v", err)
	}

	return nil
}

func (s *SpoilerService) getMediaInfoWithFFProbe(filePath string) (MediaInfo, error) {
	cmd := exec.Command("ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		filePath,
	)

	output, err := cmd.Output()
	if err != nil {
		return MediaInfo{}, fmt.Errorf("ffprobe failed: %v", err)
	}

	var result struct {
		Format struct {
			Duration string            `json:"duration"`
			Size     string            `json:"size"`
			BitRate  string            `json:"bit_rate"`
			Tags     map[string]string `json:"tags"`
		} `json:"format"`
		Streams []struct {
			CodecType string            `json:"codec_type"`
			CodecName string            `json:"codec_name"`
			Width     int               `json:"width"`
			Height    int               `json:"height"`
			Duration  string            `json:"duration"`
			BitRate   string            `json:"bit_rate"`
			Tags      map[string]string `json:"tags"`
		} `json:"streams"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return MediaInfo{}, fmt.Errorf("failed to parse ffprobe output: %v", err)
	}

	mediaInfo := MediaInfo{
		General: make(map[string]string),
		Video:   make(map[string]string),
		Audio:   make(map[string]string),
	}

	// General info
	mediaInfo.General["duration"] = result.Format.Duration
	mediaInfo.General["size"] = result.Format.Size
	mediaInfo.General["bit_rate"] = result.Format.BitRate

	// Process streams
	for _, stream := range result.Streams {
		switch stream.CodecType {
		case "video":
			mediaInfo.Video["codec_name"] = stream.CodecName
			if stream.Width > 0 {
				mediaInfo.Video["width"] = strconv.Itoa(stream.Width)
			}
			if stream.Height > 0 {
				mediaInfo.Video["height"] = strconv.Itoa(stream.Height)
			}
			if stream.Duration != "" {
				mediaInfo.Video["duration"] = stream.Duration
			}
			if stream.BitRate != "" {
				mediaInfo.Video["bit_rate"] = stream.BitRate
			}
			if stream.BitRate == "" && stream.Tags != nil {
				if br, ok := stream.Tags["BPS"]; ok {
					mediaInfo.Video["bit_rate"] = br
				}
			}
		case "audio":
			mediaInfo.Audio["codec_name"] = stream.CodecName
			if stream.Duration != "" {
				mediaInfo.Audio["duration"] = stream.Duration
			}
			if stream.BitRate != "" {
				mediaInfo.Audio["bit_rate"] = stream.BitRate
			}
			if stream.BitRate == "" && stream.Tags != nil {
				if br, ok := stream.Tags["BPS"]; ok {
					mediaInfo.Audio["bit_rate"] = br
				}
			}
		}
	}

	return mediaInfo, nil
}

func (s *SpoilerService) isVideoFile(filePath string) (bool, error) {
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=codec_type",
		"-of", "csv=p=0",
		filePath,
	)

	output, err := cmd.Output()
	if err != nil {
		return false, nil // Not a video file or ffprobe failed
	}

	return strings.TrimSpace(string(output)) == "video", nil
}

func (s *SpoilerService) GetExpandedFilePaths(paths []string) ([]string, error) {
	var files []string

	for _, path := range paths {
		info, err := os.Stat(path)
		if err != nil {
			continue
		}

		if info.IsDir() {
			err := filepath.WalkDir(path, func(filePath string, d fs.DirEntry, err error) error {
				if err != nil {
					return nil
				}

				if !d.IsDir() {
					// Check if file is a video using ffprobe
					isVideo, err := s.isVideoFile(filePath)
					if err == nil && isVideo {
						files = append(files, filePath)
					}
				}
				return nil
			})
			if err != nil {
				return nil, err
			}
		} else {
			// Check if file is a video using ffprobe
			isVideo, err := s.isVideoFile(path)
			if err == nil && isVideo {
				files = append(files, path)
			}
		}
	}

	sort.Strings(files)
	return files, nil
}

func (s *SpoilerService) GenerateResultForMovie(movieID string) string {
	var movie *Movie
	for _, m := range s.movies {
		if m.ID == movieID {
			movie = &m
			break
		}
	}

	if movie == nil || movie.FileName == "" {
		return ""
	}

	return s.generateMovieSpoiler(*movie)
}

func (s *SpoilerService) GenerateResult() string {
	if len(s.movies) == 0 {
		return ""
	}

	var result strings.Builder

	for _, movie := range s.movies {
		if movie.FileName == "" || movie.ProcessingState != StateCompleted {
			continue
		}

		result.WriteString(s.generateMovieSpoiler(movie))
		result.WriteString("\n")
	}

	return result.String()
}

func (s *SpoilerService) generateMovieSpoiler(movie Movie) string {
	tmp := s.template

	// Replace basic placeholders
	tmp = strings.ReplaceAll(tmp, "%FILE_NAME%", movie.FileName)
	tmp = strings.ReplaceAll(tmp, "%FILE_SIZE%", movie.FileSize)
	tmp = strings.ReplaceAll(tmp, "%DURATION%", movie.Duration)
	tmp = strings.ReplaceAll(tmp, "%WIDTH%", movie.Width)
	tmp = strings.ReplaceAll(tmp, "%HEIGHT%", movie.Height)
	tmp = strings.ReplaceAll(tmp, "%BIT_RATE%", movie.BitRate)
	tmp = strings.ReplaceAll(tmp, "%VIDEO_BIT_RATE%", movie.VideoBitRate)
	tmp = strings.ReplaceAll(tmp, "%AUDIO_BIT_RATE%", movie.AudioBitRate)
	tmp = strings.ReplaceAll(tmp, "%VIDEO_CODEC%", movie.VideoCodec)
	tmp = strings.ReplaceAll(tmp, "%AUDIO_CODEC%", movie.AudioCodec)

	// Handle thumbnail (BBThumb)
	if movie.ThumbnailURL != "" {
		tmp = strings.ReplaceAll(tmp, "%THUMBNAIL%", movie.ThumbnailURL)
	} else {
		tmp = strings.ReplaceAll(tmp, "%THUMBNAIL%", "")
	}

	// Handle thumbnail big (BBBig)
	if movie.ThumbnailBigURL != "" {
		tmp = strings.ReplaceAll(tmp, "%THUMBNAIL_BIG%", movie.ThumbnailBigURL)
	} else {
		tmp = strings.ReplaceAll(tmp, "%THUMBNAIL_BIG%", "")
	}

	// Handle screenshots (BBThumb) with newline separator
	if len(movie.ScreenshotURLs) > 0 {
		screenshotsStr := strings.Join(movie.ScreenshotURLs, "\n")
		tmp = strings.ReplaceAll(tmp, "%SCREENSHOTS%", screenshotsStr)

		// Handle screenshots with space separator
		screenshotsSpaced := strings.Join(movie.ScreenshotURLs, " ")
		tmp = strings.ReplaceAll(tmp, "%SCREENSHOTS_SPACED%", screenshotsSpaced)
	} else {
		tmp = strings.ReplaceAll(tmp, "%SCREENSHOTS%", "")
		tmp = strings.ReplaceAll(tmp, "%SCREENSHOTS_SPACED%", "")
	}

	// Handle screenshots big (BBBig) with newline separator
	if len(movie.ScreenshotBigURLs) > 0 {
		screenshotsBigStr := strings.Join(movie.ScreenshotBigURLs, "\n")
		tmp = strings.ReplaceAll(tmp, "%SCREENSHOTS_BIG%", screenshotsBigStr)

		// Handle screenshots big with space separator
		screenshotsBigSpaced := strings.Join(movie.ScreenshotBigURLs, " ")
		tmp = strings.ReplaceAll(tmp, "%SCREENSHOTS_BIG_SPACED%", screenshotsBigSpaced)
	} else {
		tmp = strings.ReplaceAll(tmp, "%SCREENSHOTS_BIG%", "")
		tmp = strings.ReplaceAll(tmp, "%SCREENSHOTS_BIG_SPACED%", "")
	}

	// Replace parameter placeholders
	paramPattern := regexp.MustCompile(`%[^%]+%`)
	tmp = paramPattern.ReplaceAllStringFunc(tmp, func(param string) string {
		if value, exists := movie.Params[param]; exists && value != "" {
			return value
		}
		return "−"
	})

	return tmp
}

// Settings management
func (s *SpoilerService) GetSettings() AppSettings {
	return s.settings
}

func (s *SpoilerService) UpdateSettings(settings AppSettings) {
	s.settings = settings

	// Save to config
	config := SpoilerConfig{
		ScreenshotCount:          settings.ScreenshotCount,
		FastpicSID:               settings.FastpicSID,
		ScreenshotQuality:        settings.ScreenshotQuality,
		MaxConcurrentScreenshots: settings.MaxConcurrentScreenshots,
		MaxConcurrentUploads:     settings.MaxConcurrentUploads,
		Template:                 s.template,
	}

	if err := s.configManager.UpdateConfig(config); err != nil {
		log.Printf("Failed to save settings: %v", err)
	}

	s.initSemaphores() // Reinitialize semaphores with new limits
}

// Template management
func (s *SpoilerService) GetTemplate() string {
	return s.template
}

func (s *SpoilerService) SetTemplate(template string) {
	s.template = template

	// Update config with new template
	config := s.configManager.GetConfig()
	config.Template = template
	if err := s.configManager.UpdateConfig(config); err != nil {
		log.Printf("Failed to save template: %v", err)
	}
}

// Helper functions
func formatDuration(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60

	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}

func formatBitRate(bitRateStr string) string {
	if bitRateStr == "" {
		return ""
	}

	bitRate, err := strconv.ParseFloat(bitRateStr, 64)
	if err != nil {
		return bitRateStr
	}

	kbps := bitRate / 1000
	if kbps >= 1000 {
		return fmt.Sprintf("%.1f Mbps", kbps/1000)
	}
	return fmt.Sprintf("%.0f kbps", kbps)
}

func formatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
