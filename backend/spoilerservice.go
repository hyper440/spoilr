package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
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
			Language:     "English",
			ConvertToGB:  true,
			CenterAlign:  false,
			HideEmpty:    true,
			UIFontSize:   12,
			ListFontSize: 10,
			TextFontSize: 12,
		},
		template: getDefaultTemplate(),
		nextID:   1,
	}
}

// SetApp sets the application instance for event emission
func (s *SpoilerService) SetApp(app *application.App) {
	s.app = app
}

// GetMovies returns all movies
func (s *SpoilerService) GetMovies() []Movie {
	return s.movies
}

// AddMovie adds a single movie
func (s *SpoilerService) AddMovie(movie Movie) {
	movie.ID = s.nextID
	s.nextID++
	s.movies = append(s.movies, movie)
}

// RemoveMovie removes a movie by ID
func (s *SpoilerService) RemoveMovie(id int) {
	for i, movie := range s.movies {
		if movie.ID == id {
			s.movies = append(s.movies[:i], s.movies[i+1:]...)
			break
		}
	}
}

// ClearMovies removes all movies
func (s *SpoilerService) ClearMovies() {
	s.movies = make([]Movie, 0)
}

// ProcessFilesAsync starts processing files asynchronously and emits progress events
func (s *SpoilerService) ProcessFilesAsync(filePaths []string) error {
	go func() {
		ctx := context.Background()
		s.ProcessFiles(ctx, filePaths)
	}()
	return nil
}

// ProcessFiles processes multiple file paths and emits progress events
func (s *SpoilerService) ProcessFiles(ctx context.Context, filePaths []string) error {
	total := len(filePaths)
	for i, path := range filePaths {
		select {
		case <-ctx.Done():
			// Emit cancellation event
			s.EmitProgress(ProcessProgress{
				Current: i,
				Total:   total,
				Message: "Cancelled",
				Error:   "Operation cancelled",
			})
			return ctx.Err()
		default:
			// Emit progress
			s.EmitProgress(ProcessProgress{
				Current:  i + 1,
				Total:    total,
				FileName: filepath.Base(path),
				Message:  fmt.Sprintf("Processing %s", filepath.Base(path)),
			})

			movie, err := s.processFile(path)
			if err != nil {
				s.EmitProgress(ProcessProgress{
					Current:  i + 1,
					Total:    total,
					FileName: filepath.Base(path),
					Message:  fmt.Sprintf("Error processing %s", filepath.Base(path)),
					Error:    err.Error(),
				})
				continue
			}

			s.AddMovie(movie)
		}
	}

	// Emit completion
	s.EmitProgress(ProcessProgress{
		Current:   total,
		Total:     total,
		Message:   "Completed",
		Completed: true,
	})

	return nil
}

// EmitProgress emits a progress event through the Wails event system
func (s *SpoilerService) EmitProgress(progress ProcessProgress) {
	if s.app != nil {
		s.app.Event.Emit("progress", progress)
	}
}

// processFile processes a single file and extracts media info
func (s *SpoilerService) processFile(filePath string) (Movie, error) {
	movie := Movie{
		FilePath: filePath,
		FileName: filepath.Base(filePath),
		Params:   make(map[string]string),
	}

	// Get file info
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return movie, err
	}

	// Calculate file size
	sizeBytes := fileInfo.Size()
	movie.FileSizeBytes = sizeBytes

	sizeMB := float64(sizeBytes) / (1024 * 1024)
	if s.settings.ConvertToGB && sizeMB >= 1024 {
		sizeGB := sizeMB / 1024
		movie.FileSize = fmt.Sprintf("%.2f GB", sizeGB)
	} else {
		movie.FileSize = fmt.Sprintf("%.2f MB", sizeMB)
	}

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

	return movie, nil
}

// getMediaInfoWithFFProbe uses ffprobe to extract media information
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
			Tags     map[string]string `json:"tags"`
		} `json:"format"`
		Streams []struct {
			CodecType string            `json:"codec_type"`
			CodecName string            `json:"codec_name"`
			Width     int               `json:"width"`
			Height    int               `json:"height"`
			Duration  string            `json:"duration"`
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
		case "audio":
			mediaInfo.Audio["codec_name"] = stream.CodecName
			if stream.Duration != "" {
				mediaInfo.Audio["duration"] = stream.Duration
			}
		}
	}

	return mediaInfo, nil
}

// GetExpandedFilePaths expands directories to individual files
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
					return nil // Skip errors
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

// ProcessURLs processes clipboard URLs
func (s *SpoilerService) ProcessURLs(text string, acceptOnlyLinks bool) []Movie {
	var movies []Movie
	var pattern string

	if acceptOnlyLinks {
		pattern = `https?://[^\s]+`
	} else {
		pattern = `\S+`
	}

	re := regexp.MustCompile(pattern)
	matches := re.FindAllString(text, -1)

	for _, match := range matches {
		movie := Movie{
			ID:            s.nextID,
			ScreenListURL: match,
			Params:        make(map[string]string),
		}
		s.nextID++
		movies = append(movies, movie)
		s.movies = append(s.movies, movie)
	}

	return movies
}

// GenerateResult generates the formatted spoiler list
func (s *SpoilerService) GenerateResult() string {
	if len(s.movies) == 0 {
		return ""
	}

	// Calculate max lengths for alignment
	maxFileSize, maxWidth, maxHeight := 0, 0, 0
	for _, movie := range s.movies {
		if movie.FileName == "" {
			continue
		}
		if len(movie.FileSize) > maxFileSize {
			maxFileSize = len(movie.FileSize)
		}
		if len(movie.Width) > maxWidth {
			maxWidth = len(movie.Width)
		}
		if len(movie.Height) > maxHeight {
			maxHeight = len(movie.Height)
		}
	}

	var result strings.Builder

	for _, movie := range s.movies {
		if movie.FileName == "" {
			continue
		}

		tmp := s.template

		// Replace basic placeholders
		if s.settings.CenterAlign {
			spaces := maxFileSize - len(movie.FileSize)
			fileSize := strings.Repeat("&#x2008;", spaces) +
				strings.ReplaceAll(movie.FileSize, "GB", "GB&#x200A;") +
				strings.Repeat("&#x2008;", spaces)
			tmp = strings.ReplaceAll(tmp, "%FILE_SIZE%", fileSize)

			widthHeightSpaces := maxWidth + maxHeight - len(movie.Width) - len(movie.Height)
			width := strings.Repeat("&#x2008;", widthHeightSpaces) + movie.Width
			height := movie.Height + strings.Repeat("&#x2008;", widthHeightSpaces)
			tmp = strings.ReplaceAll(tmp, "%WIDTH%", width)
			tmp = strings.ReplaceAll(tmp, "%HEIGHT%", height)
		} else {
			tmp = strings.ReplaceAll(tmp, "%FILE_SIZE%", movie.FileSize)
			tmp = strings.ReplaceAll(tmp, "%WIDTH%", movie.Width)
			tmp = strings.ReplaceAll(tmp, "%HEIGHT%", movie.Height)
		}

		tmp = strings.ReplaceAll(tmp, "%DURATION%", movie.Duration)
		tmp = strings.ReplaceAll(tmp, "%FILE_NAME%", movie.FileName)

		// Handle image URL
		if movie.ScreenListURL != "" {
			imgPattern := regexp.MustCompile(`^https?://.*\.(jpg|png)$`)
			if imgPattern.MatchString(movie.ScreenListURL) {
				tmp = strings.ReplaceAll(tmp, "%IMG%", fmt.Sprintf("[IMG]%s[/IMG]", movie.ScreenListURL))
			} else {
				tmp = strings.ReplaceAll(tmp, "%IMG%", movie.ScreenListURL)
			}
		}

		// Replace parameter placeholders
		paramPattern := regexp.MustCompile(`%[^%]+%`)
		tmp = paramPattern.ReplaceAllStringFunc(tmp, func(param string) string {
			if value, exists := movie.Params[param]; exists && value != "" {
				return value
			}

			// Handle special cases
			switch param {
			case "%Video@FrameRate%":
				return s.tryParams(movie, []string{"%Video@FrameRate_Original%", "%Video@FrameRate_Nominal%", "%Video@FrameRate_Maximum%"})
			case "%Video@FrameRate/String%":
				return s.tryParams(movie, []string{"%Video@FrameRate_Original/String%", "%Video@FrameRate_Nominal/String%", "%Video@FrameRate_Maximum/String%"})
			case "%Video@BitRate/String%":
				return s.tryParams(movie, []string{"%Video@BitRate_Nominal/String%", "%Video@BitRate_Maximum/String%", "%Video@Format_Profile%"})
			case "%Audio@BitRate/String%":
				return s.tryParams(movie, []string{"%Audio@BitRate_Nominal/String%", "%Audio@BitRate_Maximum/String%", "%Audio@Format_AdditionalFeatures%"})
			default:
				return "−"
			}
		})

		result.WriteString(tmp)
		result.WriteString("\n")
	}

	return strings.ReplaceAll(result.String(), "&#x2008;&#x2008;", "&#x2007;")
}

// tryParams tries multiple parameter options and returns the first non-empty value
func (s *SpoilerService) tryParams(movie Movie, params []string) string {
	for _, param := range params {
		if value, exists := movie.Params[param]; exists && value != "" {
			return value
		}
	}
	return "−"
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

// Link processing (unbender)
func (s *SpoilerService) ProcessLinks(text string, replaceInOriginal bool) LinkProcessResult {
	linkPattern := regexp.MustCompile(`https?://[^\s]+`)
	links := linkPattern.FindAllString(text, -1)

	result := LinkProcessResult{
		OriginalLinks:  make([]string, 0),
		ProcessedLinks: make([]string, 0),
		Errors:         make([]string, 0),
	}

	for _, link := range links {
		result.OriginalLinks = append(result.OriginalLinks, link)

		processed, err := s.processLink(link)
		if err != nil {
			result.Errors = append(result.Errors, err.Error())
			result.ProcessedLinks = append(result.ProcessedLinks, "Error: "+err.Error())
		} else {
			result.ProcessedLinks = append(result.ProcessedLinks, processed)
		}
	}

	return result
}

// processLink processes a single link to get the direct URL
func (s *SpoilerService) processLink(url string) (string, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// This is a simplified version - you'd need to implement specific logic
	// for different image hosting services like imagebam.com, imagevenue.com, etc.
	// For now, just return the original URL
	return url, nil
}

// Link filtering
func (s *SpoilerService) FilterLinks(text string, allowFilters, blockFilters []string) ([]string, []string) {
	linkPattern := regexp.MustCompile(`https?://[^\s]+`)
	allLinks := linkPattern.FindAllString(text, -1)

	var filteredLinks []string

	for _, link := range allLinks {
		// Check if all allow filters match
		allowMatch := true
		for _, filter := range allowFilters {
			if filter != "" && !strings.Contains(link, filter) {
				allowMatch = false
				break
			}
		}

		// Check if any block filter matches
		blockMatch := false
		for _, filter := range blockFilters {
			if filter != "" && strings.Contains(link, filter) {
				blockMatch = true
				break
			}
		}

		if allowMatch && !blockMatch {
			filteredLinks = append(filteredLinks, link)
		}
	}

	return allLinks, filteredLinks
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

func getDefaultTemplate() string {
	return `[spoiler="%FILE_NAME%"]
File: %FILE_NAME%
Size: %FILE_SIZE%
Duration: %DURATION%
Resolution: %WIDTH%x%HEIGHT%

%IMG%
[/spoiler]`
}
