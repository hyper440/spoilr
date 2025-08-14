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
	"time"

	"github.com/google/uuid"
	"github.com/wailsapp/wails/v3/pkg/application"
)

type SpoilerService struct {
	app        *application.App
	movies     []Movie
	settings   AppSettings
	template   string
	processing bool
	cancelCtx  context.Context
	cancelFn   context.CancelFunc
}

func NewSpoilerService() *SpoilerService {
	service := &SpoilerService{
		movies: make([]Movie, 0),
		settings: AppSettings{
			HideEmpty:         true,
			UIFontSize:        12,
			ListFontSize:      10,
			TextFontSize:      12,
			ScreenshotCount:   6,
			FastpicSID:        "",
			ScreenshotQuality: 2,
		},
		processing: false,
	}
	service.template = service.GetDefaultTemplate()
	return service
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

%SCREENSHOTS%
[/spoiler]`
}

// Helper function to find and update movie by ID
func (s *SpoilerService) updateMovieByID(id string, updateFn func(*Movie)) bool {
	for i := range s.movies {
		if s.movies[i].ID == id {
			updateFn(&s.movies[i])
			return true
		}
	}
	return false
}

// Helper function to get movie by ID (returns copy)
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

	for _, path := range expandedPaths {
		// Get basic file info
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

		// Immediately analyze media info
		mediaInfo, err := s.getMediaInfoWithFFProbe(path)
		if err != nil {
			log.Printf("Failed to analyze media %s: %v", filepath.Base(path), err)
			movie.ProcessingState = StateError
			movie.ProcessingError = err.Error()
		} else {
			// Extract and set media info
			s.extractMediaInfo(&movie, mediaInfo)
			movie.ProcessingState = StatePending // Ready for screenshot processing
		}

		s.movies = append(s.movies, movie)
	}

	s.emitState()
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
		// Use a defer to ensure processing is always set to false when done
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
			log.Println("Processing completed, set processing to false")
		}()

		// Process movies by continuously checking for pending ones
		// This handles reordering/removal during processing
		for {
			select {
			case <-s.cancelCtx.Done():
				log.Println("Processing cancelled")
				return
			default:
			}

			// Get current pending movies (refreshed each iteration)
			currentPending := s.getPendingMovies()
			if len(currentPending) == 0 {
				log.Println("No more pending movies, processing complete")
				return
			}

			// Process the first pending movie
			movie := currentPending[0]
			err := s.processMovie(movie.ID)
			if err != nil {
				log.Printf("Error processing movie %s: %v", movie.FileName, err)
				// Continue with next movie even if one fails
				continue
			}
		}
	}()

	return nil
}

// ReorderMovies updates the order of movies in the backend to match the provided order
func (s *SpoilerService) ReorderMovies(newOrder []string) error {
	// Create a map of ID to movie for quick lookup
	movieMap := make(map[string]Movie)
	for _, movie := range s.movies {
		movieMap[movie.ID] = movie
	}

	// Verify all IDs in new order exist
	for _, id := range newOrder {
		if _, exists := movieMap[id]; !exists {
			return fmt.Errorf("movie with ID %s not found", id)
		}
	}

	// Rebuild the movies slice in the new order
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

func (s *SpoilerService) processMovie(movieID string) error {
	// Get movie by ID to check if it still exists and get current state
	movie, exists := s.getMovieByID(movieID)
	if !exists {
		return fmt.Errorf("movie not found")
	}

	// Check for cancellation before starting
	select {
	case <-s.cancelCtx.Done():
		s.updateMovieByID(movieID, func(m *Movie) {
			m.ProcessingState = StatePending
			m.ProcessingError = ""
		})
		s.emitState()
		return fmt.Errorf("processing cancelled before starting")
	default:
	}

	// Skip if media analysis failed
	if movie.ProcessingState == StateError {
		return fmt.Errorf("movie has analysis error: %s", movie.ProcessingError)
	}

	// Generate screenshots if configured
	if s.settings.ScreenshotCount > 0 {
		// Check for cancellation before screenshots
		select {
		case <-s.cancelCtx.Done():
			s.updateMovieByID(movieID, func(m *Movie) {
				m.ProcessingState = StatePending
				m.ProcessingError = ""
			})
			s.emitState()
			return fmt.Errorf("processing cancelled before screenshots")
		default:
		}

		// Update state to generating screenshots
		s.updateMovieByID(movieID, func(m *Movie) {
			m.ProcessingState = StateGeneratingScreenshots
		})
		s.emitState()

		screenshotURLs, albumURL, err := s.generateAndUploadScreenshots(movieID, movie.FilePath)
		if err != nil {
			// Check if it was cancelled
			if s.cancelCtx.Err() != nil {
				s.updateMovieByID(movieID, func(m *Movie) {
					m.ProcessingState = StatePending
					m.ProcessingError = ""
				})
				s.emitState()
				return fmt.Errorf("screenshot processing cancelled")
			}
			log.Printf("Screenshot generation/upload failed for %s: %v", movie.FilePath, err)
			// Don't fail the entire processing, just log the error
		} else {
			s.updateMovieByID(movieID, func(m *Movie) {
				m.ScreenshotURLs = screenshotURLs
				m.ScreenshotAlbum = albumURL
			})
		}
	}

	// Final check for cancellation
	select {
	case <-s.cancelCtx.Done():
		s.updateMovieByID(movieID, func(m *Movie) {
			m.ProcessingState = StatePending
			m.ProcessingError = ""
		})
		s.emitState()
		return fmt.Errorf("processing cancelled before completion")
	default:
	}

	s.updateMovieByID(movieID, func(m *Movie) {
		m.ProcessingState = StateCompleted
		m.ProcessingError = ""
	})
	s.emitState()

	return nil
}

func (s *SpoilerService) findMovieIndex(id string) int {
	for i, movie := range s.movies {
		if movie.ID == id {
			return i
		}
	}
	return -1
}

func (s *SpoilerService) generateAndUploadScreenshots(movieID, filePath string) ([]string, string, error) {
	fastpicService := NewFastpicService(s.settings.FastpicSID)

	// Create temporary directory for screenshots
	tempDir, err := os.MkdirTemp("", "screenshots_*")
	if err != nil {
		return nil, "", fmt.Errorf("failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Get video duration
	duration, err := s.getVideoDuration(filePath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get video duration: %v", err)
	}

	log.Printf("Generating %d screenshots for %s (duration: %.2fs)", s.settings.ScreenshotCount, filepath.Base(filePath), duration)

	// Generate screenshots at regular intervals
	var screenshotPaths []string
	interval := duration / float64(s.settings.ScreenshotCount+1)

	for i := 1; i <= s.settings.ScreenshotCount; i++ {
		// Check for cancellation before generating each screenshot
		select {
		case <-s.cancelCtx.Done():
			return nil, "", fmt.Errorf("screenshot generation cancelled: %v", s.cancelCtx.Err())
		default:
		}

		timestamp := interval * float64(i)
		screenshotPath := filepath.Join(tempDir, fmt.Sprintf("screenshot_%d.jpg", i))

		err := s.generateScreenshot(filePath, screenshotPath, timestamp)
		if err != nil {
			log.Printf("Failed to generate screenshot %d for %s: %v", i, filepath.Base(filePath), err)
			continue
		}
		screenshotPaths = append(screenshotPaths, screenshotPath)
	}

	if len(screenshotPaths) == 0 {
		return nil, "", fmt.Errorf("no screenshots generated")
	}

	log.Printf("Generated %d screenshots, starting upload...", len(screenshotPaths))

	// Update state to uploading using movie ID instead of path
	s.updateMovieByID(movieID, func(m *Movie) {
		m.ProcessingState = StateUploadingScreenshots
	})
	s.emitState()

	// Check for cancellation before getting upload ID
	select {
	case <-s.cancelCtx.Done():
		return nil, "", fmt.Errorf("upload cancelled: %v", s.cancelCtx.Err())
	default:
	}

	// Get fastpic upload ID with context
	uploadID, err := fastpicService.getFastpicUploadID(s.cancelCtx)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get fastpic upload ID: %v", err)
	}

	// Upload screenshots to fastpic
	var screenshotURLs []string
	var albumURL string

	for i, screenshotPath := range screenshotPaths {
		// Check for cancellation before each upload
		select {
		case <-s.cancelCtx.Done():
			return nil, "", fmt.Errorf("upload cancelled: %v", s.cancelCtx.Err())
		default:
		}

		fileName := fmt.Sprintf("%s_screenshot_%d.jpg", strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath)), i+1)

		log.Printf("Uploading screenshot %d/%d: %s", i+1, len(screenshotPaths), fileName)

		// Upload with context
		result, err := fastpicService.uploadToFastpic(s.cancelCtx, screenshotPath, fileName, uploadID)
		if err != nil {
			log.Printf("Failed to upload screenshot %s: %v", fileName, err)
			// If cancelled, return immediately
			if s.cancelCtx.Err() != nil {
				return nil, "", fmt.Errorf("upload cancelled: %v", s.cancelCtx.Err())
			}
			continue
		}

		screenshotURLs = append(screenshotURLs, result.BBThumb)
		if albumURL == "" {
			albumURL = result.AlbumLink
		}

		log.Printf("Successfully uploaded screenshot %d: %s", i+1, result.BBThumb)
	}

	if len(screenshotURLs) == 0 {
		return nil, "", fmt.Errorf("failed to upload any screenshots")
	}

	log.Printf("Successfully uploaded %d/%d screenshots", len(screenshotURLs), len(screenshotPaths))
	return screenshotURLs, albumURL, nil
}

func (s *SpoilerService) findMovieIndexByPath(filePath string) int {
	for i, movie := range s.movies {
		if movie.FilePath == filePath {
			return i
		}
	}
	return -1
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
		// Check if error is due to context cancellation
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
		// Check if error is due to context cancellation
		if s.cancelCtx.Err() != nil {
			return fmt.Errorf("screenshot generation cancelled: %v", s.cancelCtx.Err())
		}
		return fmt.Errorf("ffmpeg command failed: %v", err)
	}

	return nil
}

func (s *SpoilerService) getMediaInfoWithFFProbe(filePath string) (MediaInfo, error) {
	// Use a simple command without cancellation context for immediate analysis
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

func (s *SpoilerService) GetExpandedFilePaths(paths []string) ([]string, error) {
	var files []string
	videoExtensions := map[string]bool{
		".mp4": true, ".avi": true, ".mkv": true, ".mov": true,
		".wmv": true, ".flv": true, ".webm": true, ".m4v": true,
		".mpg": true, ".mpeg": true, ".3gp": true, ".ogv": true,
	}

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
					ext := strings.ToLower(filepath.Ext(filePath))
					if videoExtensions[ext] {
						files = append(files, filePath)
					}
				}
				return nil
			})
			if err != nil {
				return nil, err
			}
		} else {
			ext := strings.ToLower(filepath.Ext(path))
			if videoExtensions[ext] {
				files = append(files, path)
			}
		}
	}

	sort.Strings(files)
	return files, nil
}

// GenerateResultForMovie generates spoiler for a single movie
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

// GenerateResult generates the formatted spoiler list for all movies
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

	// Handle screenshots with newline separator
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
}

// Template management
func (s *SpoilerService) GetTemplate() string {
	return s.template
}

func (s *SpoilerService) SetTemplate(template string) {
	s.template = template
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
