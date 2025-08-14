package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

type SpoilerService struct {
	app      *application.App
	movies   []Movie
	settings AppSettings
	template string
	nextID   int
}

func NewSpoilerService() *SpoilerService {
	return &SpoilerService{
		movies: make([]Movie, 0),
		settings: AppSettings{
			CenterAlign:       false,
			HideEmpty:         true,
			UIFontSize:        12,
			ListFontSize:      10,
			TextFontSize:      12,
			ScreenshotCount:   6,
			FastpicSID:        "",
			ScreenshotQuality: 2,
		},
		template: getDefaultTemplate(),
		nextID:   1,
	}
}

func (s *SpoilerService) SetApp(app *application.App) {
	s.app = app
}

func (s *SpoilerService) GetMovies() []Movie {
	return s.movies
}

func (s *SpoilerService) AddMovie(movie Movie) {
	movie.ID = s.nextID
	s.nextID++
	s.movies = append(s.movies, movie)
}

func (s *SpoilerService) RemoveMovie(id int) {
	for i, movie := range s.movies {
		if movie.ID == id {
			s.movies = append(s.movies[:i], s.movies[i+1:]...)
			break
		}
	}
}

func (s *SpoilerService) ClearMovies() {
	s.movies = make([]Movie, 0)
}

func (s *SpoilerService) ProcessFilesAsync(filePaths []string) error {
	go func() {
		ctx := context.Background()
		s.ProcessFiles(ctx, filePaths)
	}()
	return nil
}

// AddPendingFiles adds files to the list with pending status
func (s *SpoilerService) AddPendingFiles(filePaths []string) {
	for _, path := range filePaths {
		movie := Movie{
			ID:              s.nextID,
			FileName:        filepath.Base(path),
			FilePath:        path,
			Params:          make(map[string]string),
			ScreenshotURLs:  make([]string, 0),
			ProcessingState: "pending",
		}
		s.nextID++
		s.movies = append(s.movies, movie)
	}

	// Emit event to update frontend
	if s.app != nil {
		s.app.Event.Emit("moviesUpdated", s.movies)
	}
}

func (s *SpoilerService) ProcessFiles(ctx context.Context, filePaths []string) error {
	total := len(filePaths)

	s.EmitProgress(ProcessProgress{
		Current: 0,
		Total:   total,
		Message: "Starting processing...",
	})

	for i, path := range filePaths {
		select {
		case <-ctx.Done():
			s.EmitProgress(ProcessProgress{
				Current: i,
				Total:   total,
				Message: "Cancelled",
				Error:   "Operation cancelled",
			})
			return ctx.Err()
		default:
			// Update movie state to processing
			s.updateMovieState(path, "processing")

			s.EmitProgress(ProcessProgress{
				Current:  i,
				Total:    total,
				FileName: filepath.Base(path),
				Message:  fmt.Sprintf("Processing %s", filepath.Base(path)),
			})

			movie, err := s.processFile(path)
			if err != nil {
				s.updateMovieState(path, "error")
				s.EmitProgress(ProcessProgress{
					Current:  i + 1,
					Total:    total,
					FileName: filepath.Base(path),
					Message:  fmt.Sprintf("Error processing %s", filepath.Base(path)),
					Error:    err.Error(),
				})
				continue
			}

			// Update the existing movie entry
			s.updateMovieData(path, movie)
		}
	}

	s.EmitProgress(ProcessProgress{
		Current:   total,
		Total:     total,
		Message:   "Completed",
		Completed: true,
	})

	return nil
}

func (s *SpoilerService) updateMovieState(filePath, state string) {
	for i := range s.movies {
		if s.movies[i].FilePath == filePath {
			s.movies[i].ProcessingState = state
			break
		}
	}
	// Emit update event
	if s.app != nil {
		s.app.Event.Emit("moviesUpdated", s.movies)
	}
}

func (s *SpoilerService) updateMovieData(filePath string, processedMovie Movie) {
	for i := range s.movies {
		if s.movies[i].FilePath == filePath {
			processedMovie.ID = s.movies[i].ID
			processedMovie.ProcessingState = "completed"
			s.movies[i] = processedMovie
			break
		}
	}
	// Emit update event
	if s.app != nil {
		s.app.Event.Emit("moviesUpdated", s.movies)
	}
}

func (s *SpoilerService) EmitProgress(progress ProcessProgress) {
	if s.app != nil {
		s.app.Event.Emit("progress", progress)
	}
}

func (s *SpoilerService) processFile(filePath string) (Movie, error) {
	movie := Movie{
		FilePath:       filePath,
		FileName:       filepath.Base(filePath),
		Params:         make(map[string]string),
		ScreenshotURLs: make([]string, 0),
	}

	// Get file info
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return movie, err
	}

	// Calculate file size
	sizeBytes := fileInfo.Size()
	movie.FileSizeBytes = sizeBytes
	movie.FileSize = formatFileSize(sizeBytes)

	// Use ffprobe to get media info
	mediaInfo, err := s.getMediaInfoWithFFProbe(filePath)
	if err != nil {
		return movie, err
	}

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

	// Generate screenshots if fastpic SID is configured
	if s.settings.FastpicSID != "" && s.settings.ScreenshotCount > 0 {
		screenshotURLs, albumURL, err := s.generateAndUploadScreenshots(filePath)
		if err == nil {
			movie.ScreenshotURLs = screenshotURLs
			movie.ScreenshotAlbum = albumURL
		}
	}

	return movie, nil
}

func (s *SpoilerService) generateAndUploadScreenshots(filePath string) ([]string, string, error) {
	fastpicService := NewFastpicService(s.settings.FastpicSID)

	// Create temporary directory for screenshots
	tempDir, err := os.MkdirTemp("", "screenshots_*")
	if err != nil {
		return nil, "", err
	}
	defer os.RemoveAll(tempDir)

	// Get video duration
	duration, err := s.getVideoDuration(filePath)
	if err != nil {
		return nil, "", err
	}

	// Generate screenshots at regular intervals
	var screenshotPaths []string
	interval := duration / float64(s.settings.ScreenshotCount+1)

	for i := 1; i <= s.settings.ScreenshotCount; i++ {
		timestamp := interval * float64(i)
		screenshotPath := filepath.Join(tempDir, fmt.Sprintf("screenshot_%d.jpg", i))

		err := s.generateScreenshot(filePath, screenshotPath, timestamp)
		if err != nil {
			continue
		}
		screenshotPaths = append(screenshotPaths, screenshotPath)
	}

	if len(screenshotPaths) == 0 {
		return nil, "", fmt.Errorf("no screenshots generated")
	}

	// Get fastpic upload ID
	uploadID, err := fastpicService.getFastpicUploadID()
	if err != nil {
		return nil, "", err
	}

	// Upload screenshots to fastpic
	var screenshotURLs []string
	var albumURL string

	for i, screenshotPath := range screenshotPaths {
		fileName := fmt.Sprintf("%s_screenshot_%d.jpg", strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath)), i+1)
		result, err := fastpicService.uploadToFastpic(screenshotPath, fileName, uploadID)
		if err != nil {
			continue
		}

		screenshotURLs = append(screenshotURLs, result.BBThumb)
		if albumURL == "" {
			albumURL = result.AlbumLink
		}
	}

	return screenshotURLs, albumURL, nil
}

func (s *SpoilerService) getVideoDuration(filePath string) (float64, error) {
	cmd := exec.Command("ffprobe",
		"-v", "quiet",
		"-show_entries", "format=duration",
		"-of", "csv=p=0",
		filePath,
	)

	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	duration, err := strconv.ParseFloat(strings.TrimSpace(string(output)), 64)
	if err != nil {
		return 0, err
	}

	return duration, nil
}

func (s *SpoilerService) generateScreenshot(videoPath, outputPath string, timestamp float64) error {
	cmd := exec.Command("ffmpeg",
		"-ss", fmt.Sprintf("%.2f", timestamp),
		"-i", videoPath,
		"-vframes", "1",
		"-q:v", fmt.Sprintf("%d", s.settings.ScreenshotQuality),
		"-y",
		outputPath,
	)

	return cmd.Run()
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
		return MediaInfo{}, err
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
func (s *SpoilerService) GenerateResultForMovie(movieID int) string {
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
		if movie.FileName == "" || movie.ProcessingState != "completed" {
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

	// Handle screenshots
	if len(movie.ScreenshotURLs) > 0 {
		screenshotsStr := strings.Join(movie.ScreenshotURLs, "\n")
		tmp = strings.ReplaceAll(tmp, "%SCREENSHOTS%", screenshotsStr)
	} else {
		tmp = strings.ReplaceAll(tmp, "%SCREENSHOTS%", "")
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

func getDefaultTemplate() string {
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
