package backend

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"spoilr/backend/img_uploaders"
	"strings"
	"sync"

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

// UploaderRequirements tracks what uploaders are needed based on template
type UploaderRequirements struct {
	NeedsFastpic bool
	NeedsImgbox  bool
	NeedsHamster bool

	FastpicContactSheet bool
	FastpicScreenshots  bool
	ImgboxContactSheet  bool
	ImgboxScreenshots   bool
	HamsterContactSheet bool
	HamsterScreenshots  bool
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
			MtnArgs:                  config.MtnArgs,
			ImageMiniatureSize:       config.ImageMiniatureSize,
			HamsterEmail:             config.HamsterEmail,
			HamsterPassword:          config.HamsterPassword,
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
	return `[spoiler="%FILE_NAME% | %FILE_SIZE%"]
File: %FILE_NAME%
Size: %FILE_SIZE%
Duration: %DURATION%
Video: %VIDEO_CODEC% / %VIDEO_FPS% FPS / %WIDTH%x%HEIGHT% / %VIDEO_BIT_RATE%
Audio: %AUDIO_CODEC% / %AUDIO_SAMPLE_RATE% / %AUDIO_CHANNELS% / %AUDIO_BIT_RATE%

%CONTACT_SHEET_FP%

%SCREENSHOTS_FP%
[/spoiler]`
}

// getUploaderRequirements analyzes template to determine which uploaders are needed
func (s *SpoilerService) getUploaderRequirements() UploaderRequirements {
	req := UploaderRequirements{}

	// Check what types of content are needed first
	needsContactSheet := strings.Contains(s.template, "CONTACT_SHEET")
	needsScreenshots := strings.Contains(s.template, "SCREENSHOTS")

	// Early return if no image content is needed
	if !needsContactSheet && !needsScreenshots {
		return req
	}

	// Check for fastpic hosting suffix
	if strings.Contains(s.template, "_FP") {
		req.NeedsFastpic = true
		if needsContactSheet {
			req.FastpicContactSheet = true
		}
		if needsScreenshots {
			req.FastpicScreenshots = true
		}
	}

	// Check for imgbox hosting suffix
	if strings.Contains(s.template, "_IB") {
		req.NeedsImgbox = true
		if needsContactSheet {
			req.ImgboxContactSheet = true
		}
		if needsScreenshots {
			req.ImgboxScreenshots = true
		}
	}

	// Check for hamster hosting suffix
	if strings.Contains(s.template, "_HAM") {
		req.NeedsHamster = true
		if needsContactSheet {
			req.HamsterContactSheet = true
		}
		if needsScreenshots {
			req.HamsterScreenshots = true
		}
	}

	fmt.Printf("%+v\n", req)

	return req
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

func (s *SpoilerService) AddMovies(filePaths []string) error {
	// First: expand all file paths without filtering
	expandedPaths, err := s.GetExpandedFilePaths(filePaths)
	if err != nil {
		return err
	}

	if len(expandedPaths) == 0 {
		return nil
	}

	// Emit all files as movies with analyzing state
	var movieIDs []string
	for _, path := range expandedPaths {
		fileInfo, err := os.Stat(path)
		if err != nil {
			continue
		}

		movie := Movie{
			ID:                uuid.New().String(),
			FileName:          filepath.Base(path),
			FilePath:          path,
			FileSize:          FormatFileSize(fileInfo.Size()),
			FileSizeBytes:     fileInfo.Size(),
			Params:            make(map[string]string),
			ScreenshotURLs:    make([]string, 0),
			ScreenshotURLsIB:  make([]string, 0),
			ScreenshotURLsHam: make([]string, 0),
			ProcessingState:   StateAnalyzingMedia,
		}

		s.movies = append(s.movies, movie)
		movieIDs = append(movieIDs, movie.ID)
	}
	s.emitState()

	// Second: check each file and remove non-video files
	var wg sync.WaitGroup
	var mu sync.Mutex
	var validMovieIDs []string

	for _, movieID := range movieIDs {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()

			movie, exists := s.getMovieByID(id)
			if !exists {
				return
			}

			mediaInfo, isVideo, err := GetVideoMediaInfo(movie.FilePath)

			mu.Lock()
			defer mu.Unlock()

			if !isVideo || err != nil {
				// Remove non-video file
				for i, m := range s.movies {
					if m.ID == id {
						s.movies = append(s.movies[:i], s.movies[i+1:]...)
						break
					}
				}
				if err != nil {
					log.Printf("Failed to analyze media %s: %v", movie.FileName, err)
				} else {
					log.Printf("Skipped non-video file: %s", movie.FileName)
				}
			} else {
				// Update video file with media info
				s.updateMovieByID(id, func(m *Movie) {
					ExtractMediaInfo(m, mediaInfo)
					m.ProcessingState = StatePending
				})
				validMovieIDs = append(validMovieIDs, id)
			}
		}(movieID)
	}

	wg.Wait()

	// Emit final state with only video files
	s.emitState()

	log.Printf("Added %d video files out of %d total files", len(validMovieIDs), len(expandedPaths))
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

func (s *SpoilerService) addMovieError(id string, errorMsg string) {
	s.updateMovieByID(id, func(m *Movie) {
		if m.Errors == nil {
			m.Errors = make([]string, 0)
		}
		m.Errors = append(m.Errors, errorMsg)
	})
}

func (s *SpoilerService) ResetMovieStatuses() {
	for i := range s.movies {
		// Reset processing state to pending for all movies that have been analyzed
		if s.movies[i].ProcessingState != StateAnalyzingMedia {
			s.movies[i].ProcessingState = StatePending
		}
		// Clear any processing errors and individual errors
		s.movies[i].ProcessingError = ""
		s.movies[i].Errors = make([]string, 0) // Clear individual errors

		// Clear processing results - Fastpic fields
		s.movies[i].ContactSheetURL = ""
		s.movies[i].ContactSheetBigURL = ""
		s.movies[i].ScreenshotURLs = make([]string, 0)
		s.movies[i].ScreenshotBigURLs = make([]string, 0)
		s.movies[i].ScreenshotAlbum = ""

		// Clear imgbox results
		s.movies[i].ContactSheetURLIB = ""
		s.movies[i].ContactSheetBigURLIB = ""
		s.movies[i].ScreenshotURLsIB = make([]string, 0)
		s.movies[i].ScreenshotBigURLsIB = make([]string, 0)

		// Clear hamster results
		s.movies[i].ContactSheetURLHam = ""
		s.movies[i].ContactSheetBigURLHam = ""
		s.movies[i].ScreenshotURLsHam = make([]string, 0)
		s.movies[i].ScreenshotBigURLsHam = make([]string, 0)
	}
	s.emitState()
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

// Improved concurrent processing with triple uploader support
func (s *SpoilerService) processAllMoviesConcurrently() error {
	pendingMovies := s.getPendingMovies()
	if len(pendingMovies) == 0 {
		return nil
	}

	requirements := s.getUploaderRequirements()
	tempDir, err := s.createTempDirectory()
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	uploaderServices, err := s.initializeUploaderServices(requirements)
	if err != nil {
		return err
	}

	log.Printf("Starting concurrent media processing for %d movies (screenshot limit: %d, upload limit: %d)",
		len(pendingMovies), s.settings.MaxConcurrentScreenshots, s.settings.MaxConcurrentUploads)

	s.processMoviesConcurrently(pendingMovies, tempDir, uploaderServices, requirements)
	return nil
}

// Create temporary directory for processing
func (s *SpoilerService) createTempDirectory() (string, error) {
	tempDir, err := os.MkdirTemp("", "media_processing_*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %v", err)
	}
	return tempDir, nil
}

// UploaderServices holds all uploader service instances
type UploaderServices struct {
	Fastpic *img_uploaders.FastpicService
	Imgbox  *img_uploaders.ImgboxService
	Hamster *img_uploaders.HamsterService
}

// Initialize required uploader services based on requirements
func (s *SpoilerService) initializeUploaderServices(requirements UploaderRequirements) (*UploaderServices, error) {
	services := &UploaderServices{}
	imageMiniatureSize := s.configManager.GetConfig().ImageMiniatureSize

	if requirements.NeedsFastpic {
		services.Fastpic = img_uploaders.NewFastpicService(s.settings.FastpicSID, imageMiniatureSize)
		err := services.Fastpic.GetFastpicUploadID(s.cancelCtx)
		if err != nil {
			return nil, fmt.Errorf("failed to get fastpic upload ID: %v", err)
		}
		log.Printf("Fastpic service initialized")
	}

	if requirements.NeedsImgbox {
		services.Imgbox = img_uploaders.NewImgboxService(imageMiniatureSize)
		log.Printf("Imgbox service initialized")
	}

	if requirements.NeedsHamster {
		services.Hamster = img_uploaders.NewHamsterService(s.settings.HamsterEmail, s.settings.HamsterPassword)
		log.Printf("Hamster service initialized")
	}

	return services, nil
}

// Process all movies concurrently
func (s *SpoilerService) processMoviesConcurrently(movies []Movie, tempDir string, services *UploaderServices, requirements UploaderRequirements) {
	var wg sync.WaitGroup
	for _, movie := range movies {
		wg.Add(1)
		go func(movie Movie) {
			defer wg.Done()
			s.processMovieWithLimits(movie, tempDir, services.Fastpic, services.Imgbox, services.Hamster, requirements)
		}(movie)
	}
	wg.Wait()
}

func (s *SpoilerService) processMovieWithLimits(movie Movie, tempDir string, fastpicService *img_uploaders.FastpicService, imgboxService *img_uploaders.ImgboxService, hamsterService *img_uploaders.HamsterService, requirements UploaderRequirements) {
	s.clearMovieErrors(movie.ID)
	s.updateMovieState(movie.ID, StateWaitingForScreenshotSlot)

	movieTempDir, err := s.createMovieTempDirectory(tempDir, movie.ID)
	if err != nil {
		s.setMovieError(movie.ID, fmt.Sprintf("Failed to create temp directory: %v", err))
		return
	}

	contactSheetPath, screenshotPaths, err := s.generateMediaConcurrently(movie, movieTempDir, requirements)
	if err != nil {
		s.setMovieError(movie.ID, fmt.Sprintf("Media generation failed: %v", err))
		return
	}

	if !s.hasMediaToUpload(contactSheetPath, screenshotPaths) {
		s.setMovieError(movie.ID, "No media generated")
		return
	}

	s.updateMovieState(movie.ID, StateWaitingForUploadSlot)

	err = s.uploadMediaConcurrently(movie, contactSheetPath, screenshotPaths, fastpicService, imgboxService, hamsterService, requirements)
	if err != nil {
		s.setMovieError(movie.ID, fmt.Sprintf("Upload failed: %v", err))
		return
	}

	s.finalizeMovieProcessing(movie.ID)
}

// Clear previous movie errors
func (s *SpoilerService) clearMovieErrors(movieID string) {
	s.updateMovieByID(movieID, func(m *Movie) {
		m.Errors = make([]string, 0)
	})
}

// Update movie processing state
func (s *SpoilerService) updateMovieState(movieID string, state ProcessingState) {
	s.updateMovieByID(movieID, func(m *Movie) {
		m.ProcessingState = state
	})
	s.emitState()
}

// Set movie error state
func (s *SpoilerService) setMovieError(movieID string, errorMsg string) {
	s.updateMovieByID(movieID, func(m *Movie) {
		m.ProcessingState = StateError
		m.ProcessingError = errorMsg
	})
	s.emitState()
}

// Create movie-specific temporary directory
func (s *SpoilerService) createMovieTempDirectory(tempDir, movieID string) (string, error) {
	movieTempDir := filepath.Join(tempDir, movieID)
	if err := os.MkdirAll(movieTempDir, 0755); err != nil {
		return "", err
	}
	return movieTempDir, nil
}

// Check if we have any media to upload
func (s *SpoilerService) hasMediaToUpload(contactSheetPath string, screenshotPaths []string) bool {
	return contactSheetPath != "" || len(screenshotPaths) > 0
}

// Finalize movie processing and set final state
func (s *SpoilerService) finalizeMovieProcessing(movieID string) {
	movie, exists := s.getMovieByID(movieID)
	if !exists {
		return
	}

	finalState := StateCompleted
	if len(movie.Errors) > 0 {
		log.Printf("Movie %s completed with %d warnings/errors", movie.FileName, len(movie.Errors))
	} else {
		log.Printf("Successfully processed movie: %s", movie.FileName)
	}

	s.updateMovieByID(movieID, func(m *Movie) {
		m.ProcessingState = finalState
	})
	s.emitState()
}

// Generate contact sheet and screenshots with proper concurrency control
func (s *SpoilerService) generateMediaConcurrently(movie Movie, tempDir string, requirements UploaderRequirements) (string, []string, error) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var generationStarted bool
	var contactSheetPath string
	var screenshotPaths []string

	needsContactSheet := s.needsContactSheet(requirements)
	needsScreenshots := s.needsScreenshots(requirements)

	if needsContactSheet {
		wg.Add(1)
		go s.generateContactSheetAsync(&wg, &mu, &generationStarted, movie, tempDir, &contactSheetPath)
	}

	if needsScreenshots && s.settings.ScreenshotCount > 0 {
		screenshotPaths = make([]string, s.settings.ScreenshotCount)
		s.generateScreenshotsAsync(&wg, &mu, &generationStarted, movie, tempDir, screenshotPaths)
	}

	wg.Wait()

	if s.cancelCtx.Err() != nil {
		return "", nil, s.cancelCtx.Err()
	}

	validScreenshots := s.filterValidScreenshots(screenshotPaths)
	return contactSheetPath, validScreenshots, nil
}

// Check if contact sheet is needed
func (s *SpoilerService) needsContactSheet(requirements UploaderRequirements) bool {
	return requirements.FastpicContactSheet || requirements.ImgboxContactSheet || requirements.HamsterContactSheet
}

// Check if screenshots are needed
func (s *SpoilerService) needsScreenshots(requirements UploaderRequirements) bool {
	return requirements.FastpicScreenshots || requirements.ImgboxScreenshots || requirements.HamsterScreenshots
}

// Generate contact sheet asynchronously
func (s *SpoilerService) generateContactSheetAsync(wg *sync.WaitGroup, mu *sync.Mutex, generationStarted *bool, movie Movie, tempDir string, contactSheetPath *string) {
	defer wg.Done()

	select {
	case s.screenshotSemaphore <- struct{}{}:
		defer func() { <-s.screenshotSemaphore }()

		s.markGenerationStarted(mu, generationStarted, movie.ID)

		path, err := s.generateMovieContactSheet(movie.FilePath, tempDir)
		*contactSheetPath = path

		if err != nil {
			s.addMovieError(movie.ID, fmt.Sprintf("Contact sheet generation failed: %v", err))
			log.Printf("Failed to generate contact sheet for %s: %v", movie.FileName, err)
		}

	case <-s.cancelCtx.Done():
		return
	}
}

// Generate screenshots asynchronously
func (s *SpoilerService) generateScreenshotsAsync(wg *sync.WaitGroup, mu *sync.Mutex, generationStarted *bool, movie Movie, tempDir string, screenshotPaths []string) {
	interval := movie.Duration / float64(s.settings.ScreenshotCount+1)

	for i := 0; i < s.settings.ScreenshotCount; i++ {
		wg.Add(1)
		go s.generateSingleScreenshotAsync(wg, mu, generationStarted, movie, tempDir, screenshotPaths, i, interval)
	}
}

// Generate a single screenshot asynchronously
func (s *SpoilerService) generateSingleScreenshotAsync(wg *sync.WaitGroup, mu *sync.Mutex, generationStarted *bool, movie Movie, tempDir string, screenshotPaths []string, index int, interval float64) {
	defer wg.Done()

	select {
	case s.screenshotSemaphore <- struct{}{}:
		defer func() { <-s.screenshotSemaphore }()

		s.markGenerationStarted(mu, generationStarted, movie.ID)

		timestamp := interval * float64(index+1)
		outputPath := filepath.Join(tempDir, fmt.Sprintf("screenshot_%d.jpg", index+1))

		err := s.generateScreenshot(movie.FilePath, outputPath, timestamp)
		if err == nil {
			screenshotPaths[index] = outputPath
		} else {
			s.addMovieError(movie.ID, fmt.Sprintf("Screenshot %d generation failed: %v", index+1, err))
			log.Printf("Failed to generate screenshot %d for %s: %v", index+1, movie.FileName, err)
		}

	case <-s.cancelCtx.Done():
		return
	}
}

// Mark generation as started (thread-safe)
func (s *SpoilerService) markGenerationStarted(mu *sync.Mutex, generationStarted *bool, movieID string) {
	mu.Lock()
	defer mu.Unlock()

	if !*generationStarted {
		*generationStarted = true
		s.updateMovieByID(movieID, func(m *Movie) {
			m.ProcessingState = StateGeneratingScreenshots
		})
		s.emitState()
	}
}

// Filter out failed screenshots
func (s *SpoilerService) filterValidScreenshots(screenshotPaths []string) []string {
	var validScreenshots []string
	for _, path := range screenshotPaths {
		if path != "" {
			validScreenshots = append(validScreenshots, path)
		}
	}
	return validScreenshots
}

// Upload media with proper concurrency control to all three services
func (s *SpoilerService) uploadMediaConcurrently(movie Movie, contactSheetPath string, screenshotPaths []string, fastpicService *img_uploaders.FastpicService, imgboxService *img_uploaders.ImgboxService, hamsterService *img_uploaders.HamsterService, requirements UploaderRequirements) error {
	var wg sync.WaitGroup
	var mu sync.Mutex
	var uploadStarted bool

	baseFileName := strings.TrimSuffix(filepath.Base(movie.FilePath), filepath.Ext(movie.FilePath))

	s.uploadContactSheets(&wg, &mu, &uploadStarted, movie, contactSheetPath, baseFileName, fastpicService, imgboxService, hamsterService, requirements)
	s.uploadScreenshots(&wg, &mu, &uploadStarted, movie, screenshotPaths, baseFileName, fastpicService, imgboxService, hamsterService, requirements)

	wg.Wait()

	if s.cancelCtx.Err() != nil {
		return s.cancelCtx.Err()
	}

	return nil
}

// Upload contact sheets to all required services
func (s *SpoilerService) uploadContactSheets(wg *sync.WaitGroup, mu *sync.Mutex, uploadStarted *bool, movie Movie, contactSheetPath, baseFileName string, fastpicService *img_uploaders.FastpicService, imgboxService *img_uploaders.ImgboxService, hamsterService *img_uploaders.HamsterService, requirements UploaderRequirements) {
	if contactSheetPath == "" {
		return
	}

	if requirements.FastpicContactSheet && fastpicService != nil {
		wg.Add(1)
		go s.uploadContactSheetToFastpic(wg, mu, uploadStarted, movie, contactSheetPath, baseFileName, fastpicService)
	}

	if requirements.ImgboxContactSheet && imgboxService != nil {
		wg.Add(1)
		go s.uploadContactSheetToImgbox(wg, mu, uploadStarted, movie, contactSheetPath, imgboxService)
	}

	if requirements.HamsterContactSheet && hamsterService != nil {
		wg.Add(1)
		go s.uploadContactSheetToHamster(wg, mu, uploadStarted, movie, contactSheetPath, hamsterService)
	}
}

// Upload screenshots to all required services
func (s *SpoilerService) uploadScreenshots(wg *sync.WaitGroup, mu *sync.Mutex, uploadStarted *bool, movie Movie, screenshotPaths []string, baseFileName string, fastpicService *img_uploaders.FastpicService, imgboxService *img_uploaders.ImgboxService, hamsterService *img_uploaders.HamsterService, requirements UploaderRequirements) {
	if requirements.FastpicScreenshots && fastpicService != nil {
		s.uploadScreenshotsToFastpic(wg, mu, uploadStarted, movie, screenshotPaths, baseFileName, fastpicService)
	}

	if requirements.ImgboxScreenshots && imgboxService != nil {
		s.uploadScreenshotsToImgbox(wg, mu, uploadStarted, movie, screenshotPaths, imgboxService)
	}

	if requirements.HamsterScreenshots && hamsterService != nil {
		s.uploadScreenshotsToHamster(wg, mu, uploadStarted, movie, screenshotPaths, hamsterService)
	}
}

// Upload contact sheet to Fastpic
func (s *SpoilerService) uploadContactSheetToFastpic(wg *sync.WaitGroup, mu *sync.Mutex, uploadStarted *bool, movie Movie, contactSheetPath, baseFileName string, fastpicService *img_uploaders.FastpicService) {
	defer wg.Done()

	select {
	case s.uploadSemaphore <- struct{}{}:
		defer func() { <-s.uploadSemaphore }()

		s.markUploadStarted(mu, uploadStarted, movie.ID)

		fileName := fmt.Sprintf("%s_contact_sheet.jpg", baseFileName)
		result, err := fastpicService.UploadToFastpic(s.cancelCtx, contactSheetPath, fileName)
		if err != nil {
			s.addMovieError(movie.ID, fmt.Sprintf("Fastpic contact sheet upload failed: %v", err))
			log.Printf("Failed to upload contact sheet to fastpic for %s: %v", movie.FileName, err)
			return
		}

		s.updateMovieByID(movie.ID, func(m *Movie) {
			m.ContactSheetURL = result.BBThumb
			m.ContactSheetBigURL = result.BBBig
			if m.ScreenshotAlbum == "" {
				m.ScreenshotAlbum = result.AlbumLink
			}
		})

	case <-s.cancelCtx.Done():
		return
	}
}

// Upload contact sheet to Imgbox
func (s *SpoilerService) uploadContactSheetToImgbox(wg *sync.WaitGroup, mu *sync.Mutex, uploadStarted *bool, movie Movie, contactSheetPath string, imgboxService *img_uploaders.ImgboxService) {
	defer wg.Done()

	select {
	case s.uploadSemaphore <- struct{}{}:
		defer func() { <-s.uploadSemaphore }()

		s.markUploadStarted(mu, uploadStarted, movie.ID)

		result, err := imgboxService.UploadImage(s.cancelCtx, contactSheetPath)
		if err != nil {
			s.addMovieError(movie.ID, fmt.Sprintf("Imgbox contact sheet upload failed: %v", err))
			log.Printf("Failed to upload contact sheet to imgbox for %s: %v", movie.FileName, err)
			return
		}

		s.updateMovieByID(movie.ID, func(m *Movie) {
			m.ContactSheetURLIB = result.BBThumb
			m.ContactSheetBigURLIB = result.BBBig
		})

	case <-s.cancelCtx.Done():
		return
	}
}

// Upload contact sheet to Hamster
func (s *SpoilerService) uploadContactSheetToHamster(wg *sync.WaitGroup, mu *sync.Mutex, uploadStarted *bool, movie Movie, contactSheetPath string, hamsterService *img_uploaders.HamsterService) {
	defer wg.Done()

	select {
	case s.uploadSemaphore <- struct{}{}:
		defer func() { <-s.uploadSemaphore }()

		s.markUploadStarted(mu, uploadStarted, movie.ID)

		result, err := hamsterService.UploadImage(s.cancelCtx, contactSheetPath)
		if err != nil {
			s.addMovieError(movie.ID, fmt.Sprintf("Hamster contact sheet upload failed: %v", err))
			log.Printf("Failed to upload contact sheet to hamster for %s: %v", movie.FileName, err)
			return
		}

		s.updateMovieByID(movie.ID, func(m *Movie) {
			m.ContactSheetURLHam = result.BBThumb
			m.ContactSheetBigURLHam = result.BBBig
		})

	case <-s.cancelCtx.Done():
		return
	}
}

// Mark upload as started (thread-safe)
func (s *SpoilerService) markUploadStarted(mu *sync.Mutex, uploadStarted *bool, movieID string) {
	mu.Lock()
	defer mu.Unlock()

	if !*uploadStarted {
		*uploadStarted = true
		s.updateMovieByID(movieID, func(m *Movie) {
			m.ProcessingState = StateUploadingScreenshots
		})
		s.emitState()
	}
}

// Upload screenshots to Fastpic
func (s *SpoilerService) uploadScreenshotsToFastpic(wg *sync.WaitGroup, mu *sync.Mutex, uploadStarted *bool, movie Movie, screenshotPaths []string, baseFileName string, fastpicService *img_uploaders.FastpicService) {
	for i, screenshotPath := range screenshotPaths {
		wg.Add(1)
		go s.uploadSingleScreenshotToFastpic(wg, mu, uploadStarted, movie, screenshotPath, baseFileName, i, fastpicService)
	}
}

// Upload screenshots to Imgbox
func (s *SpoilerService) uploadScreenshotsToImgbox(wg *sync.WaitGroup, mu *sync.Mutex, uploadStarted *bool, movie Movie, screenshotPaths []string, imgboxService *img_uploaders.ImgboxService) {
	for i, screenshotPath := range screenshotPaths {
		wg.Add(1)
		go s.uploadSingleScreenshotToImgbox(wg, mu, uploadStarted, movie, screenshotPath, i, imgboxService)
	}
}

// Upload screenshots to Hamster
func (s *SpoilerService) uploadScreenshotsToHamster(wg *sync.WaitGroup, mu *sync.Mutex, uploadStarted *bool, movie Movie, screenshotPaths []string, hamsterService *img_uploaders.HamsterService) {
	for i, screenshotPath := range screenshotPaths {
		wg.Add(1)
		go s.uploadSingleScreenshotToHamster(wg, mu, uploadStarted, movie, screenshotPath, i, hamsterService)
	}
}

// Upload single screenshot to Fastpic
func (s *SpoilerService) uploadSingleScreenshotToFastpic(wg *sync.WaitGroup, mu *sync.Mutex, uploadStarted *bool, movie Movie, screenshotPath, baseFileName string, index int, fastpicService *img_uploaders.FastpicService) {
	defer wg.Done()

	select {
	case s.uploadSemaphore <- struct{}{}:
		defer func() { <-s.uploadSemaphore }()

		s.markUploadStarted(mu, uploadStarted, movie.ID)

		fileName := fmt.Sprintf("%s_screenshot_%d.jpg", baseFileName, index+1)
		result, err := fastpicService.UploadToFastpic(s.cancelCtx, screenshotPath, fileName)
		if err != nil {
			s.addMovieError(movie.ID, fmt.Sprintf("Fastpic screenshot %d upload failed: %v", index+1, err))
			log.Printf("Failed to upload screenshot %d to fastpic for %s: %v", index+1, movie.FileName, err)
			return
		}

		s.updateMovieByID(movie.ID, func(m *Movie) {
			s.ensureScreenshotSliceSize(&m.ScreenshotURLs, index)
			s.ensureScreenshotSliceSize(&m.ScreenshotBigURLs, index)

			m.ScreenshotURLs[index] = result.BBThumb
			m.ScreenshotBigURLs[index] = result.BBBig
			if m.ScreenshotAlbum == "" {
				m.ScreenshotAlbum = result.AlbumLink
			}
		})

	case <-s.cancelCtx.Done():
		return
	}
}

// Upload single screenshot to Imgbox
func (s *SpoilerService) uploadSingleScreenshotToImgbox(wg *sync.WaitGroup, mu *sync.Mutex, uploadStarted *bool, movie Movie, screenshotPath string, index int, imgboxService *img_uploaders.ImgboxService) {
	defer wg.Done()

	select {
	case s.uploadSemaphore <- struct{}{}:
		defer func() { <-s.uploadSemaphore }()

		s.markUploadStarted(mu, uploadStarted, movie.ID)

		result, err := imgboxService.UploadImage(s.cancelCtx, screenshotPath)
		if err != nil {
			s.addMovieError(movie.ID, fmt.Sprintf("Imgbox screenshot %d upload failed: %v", index+1, err))
			log.Printf("Failed to upload screenshot %d to imgbox for %s: %v", index+1, movie.FileName, err)
			return
		}

		s.updateMovieByID(movie.ID, func(m *Movie) {
			s.ensureScreenshotSliceSize(&m.ScreenshotURLsIB, index)
			s.ensureScreenshotSliceSize(&m.ScreenshotBigURLsIB, index)

			m.ScreenshotURLsIB[index] = result.BBThumb
			m.ScreenshotBigURLsIB[index] = result.BBBig
		})

	case <-s.cancelCtx.Done():
		return
	}
}

// Upload single screenshot to Hamster
func (s *SpoilerService) uploadSingleScreenshotToHamster(wg *sync.WaitGroup, mu *sync.Mutex, uploadStarted *bool, movie Movie, screenshotPath string, index int, hamsterService *img_uploaders.HamsterService) {
	defer wg.Done()

	select {
	case s.uploadSemaphore <- struct{}{}:
		defer func() { <-s.uploadSemaphore }()

		s.markUploadStarted(mu, uploadStarted, movie.ID)

		result, err := hamsterService.UploadImage(s.cancelCtx, screenshotPath)
		if err != nil {
			s.addMovieError(movie.ID, fmt.Sprintf("Hamster screenshot %d upload failed: %v", index+1, err))
			log.Printf("Failed to upload screenshot %d to hamster for %s: %v", index+1, movie.FileName, err)
			return
		}

		s.updateMovieByID(movie.ID, func(m *Movie) {
			s.ensureScreenshotSliceSize(&m.ScreenshotURLsHam, index)
			s.ensureScreenshotSliceSize(&m.ScreenshotBigURLsHam, index)

			m.ScreenshotURLsHam[index] = result.BBThumb
			m.ScreenshotBigURLsHam[index] = result.BBBig
		})

	case <-s.cancelCtx.Done():
		return
	}
}

// Ensure screenshot slice has enough capacity
func (s *SpoilerService) ensureScreenshotSliceSize(slice *[]string, index int) {
	for len(*slice) <= index {
		*slice = append(*slice, "")
	}
}

func (s *SpoilerService) generateMovieContactSheet(videoPath, tempDir string) (string, error) {
	// Check if mtn is available before trying to use it
	if _, err := exec.LookPath("mtn"); err != nil {
		// Emit event that mtn is missing (only once per processing session)
		if s.app != nil {
			s.app.Event.Emit("mtn-missing", map[string]string{
				"message": "MTN (Movie Thumbnailer) is not installed or not found in PATH. Contact sheet generation will be skipped.",
			})
		}
		log.Printf("MTN not found, skipping contact sheet generation for %s", filepath.Base(videoPath))
		return "", nil // Return empty string to skip contact sheet
	}

	// Parse user-configured MTN arguments
	mtnArgs := s.parseMtnArgs()

	// Build command arguments: start with "mtn", add user args, add output dir, add video path
	cmdArgs := append([]string{}, mtnArgs...)
	cmdArgs = append(cmdArgs, "-O", tempDir, videoPath)

	cmd := exec.CommandContext(s.cancelCtx, "mtn", cmdArgs...)

	// Capture both stdout and stderr for better error reporting
	output, err := cmd.CombinedOutput()
	if err != nil {
		if s.cancelCtx.Err() != nil {
			return "", fmt.Errorf("contact sheet generation cancelled: %v", s.cancelCtx.Err())
		}

		// Include the actual mtn output in the error message
		outputStr := strings.TrimSpace(string(output))
		if outputStr != "" {
			return "", fmt.Errorf("mtn command failed: %v\nOutput: %s", err, outputStr)
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

	// Log output for debugging when no contact sheet is found
	outputStr := strings.TrimSpace(string(output))
	if outputStr != "" {
		log.Printf("MTN output for %s: %s", filepath.Base(videoPath), outputStr)
	}

	return "", fmt.Errorf("contact sheet file not found after generation - no .jpg files in %s", tempDir)
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
					files = append(files, filePath)
				}
				return nil
			})
			if err != nil {
				return nil, err
			}
		} else {
			files = append(files, path)
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

	tmp = s.replaceBasicPlaceholders(tmp, movie)
	tmp = s.replaceContactSheetPlaceholders(tmp, movie)
	tmp = s.replaceScreenshotPlaceholders(tmp, movie)
	tmp = s.replaceParameterPlaceholders(tmp, movie)

	return tmp
}

// Replace basic movie information placeholders
func (s *SpoilerService) replaceBasicPlaceholders(template string, movie Movie) string {
	replacements := map[string]string{
		"%FILE_NAME%":      movie.FileName,
		"%FILE_SIZE%":      movie.FileSize,
		"%DURATION%":       movie.DurationFormatted,
		"%WIDTH%":          movie.Width,
		"%HEIGHT%":         movie.Height,
		"%BIT_RATE%":       movie.BitRate,
		"%VIDEO_BIT_RATE%": movie.VideoBitRate,
		"%AUDIO_BIT_RATE%": movie.AudioBitRate,
		"%VIDEO_CODEC%":    movie.VideoCodec,
		"%AUDIO_CODEC%":    movie.AudioCodec,
	}

	for placeholder, value := range replacements {
		template = strings.ReplaceAll(template, placeholder, value)
	}

	return template
}

// Replace contact sheet placeholders for all services
func (s *SpoilerService) replaceContactSheetPlaceholders(template string, movie Movie) string {
	// Fastpic contact sheets
	template = s.replaceIfNotEmpty(template, "%CONTACT_SHEET_FP%", movie.ContactSheetURL)
	template = s.replaceIfNotEmpty(template, "%CONTACT_SHEET_FP_BIG%", movie.ContactSheetBigURL)

	// Imgbox contact sheets
	template = s.replaceIfNotEmpty(template, "%CONTACT_SHEET_IB%", movie.ContactSheetURLIB)
	template = s.replaceIfNotEmpty(template, "%CONTACT_SHEET_IB_BIG%", movie.ContactSheetBigURLIB)

	// Hamster contact sheets
	template = s.replaceIfNotEmpty(template, "%CONTACT_SHEET_HAM%", movie.ContactSheetURLHam)
	template = s.replaceIfNotEmpty(template, "%CONTACT_SHEET_HAM_BIG%", movie.ContactSheetBigURLHam)

	return template
}

// Replace screenshot placeholders for all services
func (s *SpoilerService) replaceScreenshotPlaceholders(template string, movie Movie) string {
	template = s.replaceFastpicScreenshots(template, movie)
	template = s.replaceImgboxScreenshots(template, movie)
	template = s.replaceHamsterScreenshots(template, movie)
	return template
}

// Replace Fastpic screenshot placeholders
func (s *SpoilerService) replaceFastpicScreenshots(template string, movie Movie) string {
	// Regular screenshots (BBThumb)
	fastpicScreenshots := s.filterNonEmptyStrings(movie.ScreenshotURLs)
	template = s.replaceScreenshotGroup(template, "%SCREENSHOTS_FP%", "%SCREENSHOTS_FP_SPACED%", fastpicScreenshots)

	// Big screenshots (BBBig)
	fastpicScreenshotsBig := s.filterNonEmptyStrings(movie.ScreenshotBigURLs)
	template = s.replaceScreenshotGroup(template, "%SCREENSHOTS_FP_BIG%", "%SCREENSHOTS_FP_BIG_SPACED%", fastpicScreenshotsBig)

	return template
}

// Replace Imgbox screenshot placeholders
func (s *SpoilerService) replaceImgboxScreenshots(template string, movie Movie) string {
	// Regular screenshots (BBThumb)
	imgboxScreenshots := s.filterNonEmptyStrings(movie.ScreenshotURLsIB)
	template = s.replaceScreenshotGroup(template, "%SCREENSHOTS_IB%", "%SCREENSHOTS_IB_SPACED%", imgboxScreenshots)

	// Big screenshots (BBBig)
	imgboxScreenshotsBig := s.filterNonEmptyStrings(movie.ScreenshotBigURLsIB)
	template = s.replaceScreenshotGroup(template, "%SCREENSHOTS_IB_BIG%", "%SCREENSHOTS_IB_BIG_SPACED%", imgboxScreenshotsBig)

	return template
}

// Replace Hamster screenshot placeholders
func (s *SpoilerService) replaceHamsterScreenshots(template string, movie Movie) string {
	// Regular screenshots (BBThumb)
	hamsterScreenshots := s.filterNonEmptyStrings(movie.ScreenshotURLsHam)
	template = s.replaceScreenshotGroup(template, "%SCREENSHOTS_HAM%", "%SCREENSHOTS_HAM_SPACED%", hamsterScreenshots)

	// Big screenshots (BBBig)
	hamsterScreenshotsBig := s.filterNonEmptyStrings(movie.ScreenshotBigURLsHam)
	template = s.replaceScreenshotGroup(template, "%SCREENSHOTS_HAM_BIG%", "%SCREENSHOTS_HAM_BIG_SPACED%", hamsterScreenshotsBig)

	return template
}

// Replace screenshot group with both newline and space-separated versions
func (s *SpoilerService) replaceScreenshotGroup(template, newlinePlaceholder, spacePlaceholder string, screenshots []string) string {
	if len(screenshots) > 0 {
		screenshotsStr := strings.Join(screenshots, "\n")
		screenshotsSpaced := strings.Join(screenshots, " ")
		template = strings.ReplaceAll(template, newlinePlaceholder, screenshotsStr)
		template = strings.ReplaceAll(template, spacePlaceholder, screenshotsSpaced)
	} else {
		template = strings.ReplaceAll(template, newlinePlaceholder, "")
		template = strings.ReplaceAll(template, spacePlaceholder, "")
	}
	return template
}

// Replace parameter placeholders with movie-specific parameters
func (s *SpoilerService) replaceParameterPlaceholders(template string, movie Movie) string {
	paramPattern := regexp.MustCompile(`%[^%]+%`)
	return paramPattern.ReplaceAllStringFunc(template, func(param string) string {
		if value, exists := movie.Params[param]; exists && value != "" {
			return value
		}
		return "−"
	})
}

// Helper function to replace placeholder only if value is not empty
func (s *SpoilerService) replaceIfNotEmpty(template, placeholder, value string) string {
	if value != "" {
		return strings.ReplaceAll(template, placeholder, value)
	}
	return strings.ReplaceAll(template, placeholder, "")
}

// Helper function to filter out empty strings from slice
func (s *SpoilerService) filterNonEmptyStrings(urls []string) []string {
	var filtered []string
	for _, url := range urls {
		if url != "" {
			filtered = append(filtered, url)
		}
	}
	return filtered
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
		MtnArgs:                  settings.MtnArgs,
		ImageMiniatureSize:       settings.ImageMiniatureSize,
		HamsterEmail:             settings.HamsterEmail,
		HamsterPassword:          settings.HamsterPassword,
	}

	if err := s.configManager.UpdateConfig(config); err != nil {
		log.Printf("Failed to save settings: %v", err)
	}

	s.initSemaphores() // Reinitialize semaphores with new limits
}

func (s *SpoilerService) parseMtnArgs() []string {
	if s.settings.MtnArgs == "" {
		// Return default args if empty
		return []string{"-b", "2", "-w", "1200", "-c", "4", "-r", "4", "-g", "0", "-k", "1C1C1C", "-L", "4:2", "-F", "F0FFFF:10"}
	}

	// Simple argument parsing - split on spaces but handle quoted arguments
	args := []string{}
	current := ""
	inQuotes := false

	for i, char := range s.settings.MtnArgs {
		switch char {
		case '"':
			inQuotes = !inQuotes
		case ' ':
			if !inQuotes {
				if current != "" {
					args = append(args, current)
					current = ""
				}
			} else {
				current += string(char)
			}
		default:
			current += string(char)
		}

		// Add the last argument if we're at the end
		if i == len(s.settings.MtnArgs)-1 && current != "" {
			args = append(args, current)
		}
	}

	return args
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

func (s *SpoilerService) GetTemplatePresets() []TemplatePreset {
	config := s.configManager.GetConfig()
	return config.TemplatePresets
}

func (s *SpoilerService) GetCurrentPresetID() string {
	config := s.configManager.GetConfig()
	return config.CurrentPresetID
}

func (s *SpoilerService) SaveTemplatePreset(name, template string) (TemplatePreset, error) {
	if name == "" {
		return TemplatePreset{}, fmt.Errorf("preset name cannot be empty")
	}
	if template == "" {
		return TemplatePreset{}, fmt.Errorf("template cannot be empty")
	}

	preset := TemplatePreset{
		ID:       "", // Will be generated in config service
		Name:     name,
		Template: template,
	}

	err := s.configManager.SaveTemplatePreset(preset)
	if err != nil {
		return TemplatePreset{}, err
	}

	// Return the preset with generated ID
	presets := s.configManager.GetConfig().TemplatePresets
	for _, p := range presets {
		if p.Name == name && p.Template == template {
			return p, nil
		}
	}

	return preset, nil
}

func (s *SpoilerService) DeleteTemplatePreset(presetID string) error {
	return s.configManager.DeleteTemplatePreset(presetID)
}

func (s *SpoilerService) SetCurrentPreset(presetID string) error {
	err := s.configManager.SetCurrentPreset(presetID)
	if err != nil {
		return err
	}

	// Update the service's current template
	config := s.configManager.GetConfig()
	s.template = config.Template

	return nil
}
