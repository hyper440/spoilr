package backend

// Movie represents a media file with its metadata
type Movie struct {
	ID            string `json:"id"`
	FileName      string `json:"fileName"`
	FilePath      string `json:"filePath"`
	FileSize      string `json:"fileSize"`
	FileSizeBytes int64  `json:"fileSizeBytes"`
	Duration      string `json:"duration"`
	Width         string `json:"width"`
	Height        string `json:"height"`
	BitRate       string `json:"bitRate"`
	VideoBitRate  string `json:"videoBitRate"`
	AudioBitRate  string `json:"audioBitRate"`
	VideoCodec    string `json:"videoCodec"`
	AudioCodec    string `json:"audioCodec"`

	// Fastpic URLs
	ContactSheetURL    string   `json:"contactSheetUrl"`    // MTN-generated contact sheet (small)
	ContactSheetBigURL string   `json:"contactSheetBigUrl"` // MTN-generated contact sheet (big)
	ScreenshotURLs     []string `json:"screenshotUrls"`     // Individual screenshots (small)
	ScreenshotBigURLs  []string `json:"screenshotBigUrls"`  // Individual screenshots (big)
	ScreenshotAlbum    string   `json:"screenshotAlbum"`    // Album link

	// Imgbox Results - Renamed from Thumbnail* to ContactSheet*
	ContactSheetURLIB    string   `json:"contactSheetUrlIb"`    // MTN-generated contact sheet (small)
	ContactSheetBigURLIB string   `json:"contactSheetBigUrlIb"` // MTN-generated contact sheet (big)
	ScreenshotURLsIB     []string `json:"screenshotUrlsIb"`     // Individual screenshots (small)
	ScreenshotBigURLsIB  []string `json:"screenshotBigUrlsIb"`  // Individual screenshots (big)

	Params          map[string]string `json:"params"`
	ProcessingState string            `json:"processingState"`           // State constants defined below
	ProcessingError string            `json:"processingError,omitempty"` // Error details if processing fails
	Errors          []string          `json:"errors,omitempty"`          // Individual errors that occurred during processing
}

// Processing state constants
const (
	StatePending                  = "pending"
	StateAnalyzingMedia           = "analyzing_media"
	StateWaitingForScreenshotSlot = "waiting_for_screenshot_slot"
	StateGeneratingScreenshots    = "generating_screenshots"
	StateWaitingForUploadSlot     = "waiting_for_upload_slot"
	StateUploadingScreenshots     = "uploading_screenshots"
	StateCompleted                = "completed"
	StateError                    = "error"
)

// AppState represents the current application state
type AppState struct {
	Processing bool    `json:"processing"`
	Movies     []Movie `json:"movies"`
}

// MediaInfo represents extracted media information
type MediaInfo struct {
	General map[string]string `json:"general"`
	Video   map[string]string `json:"video"`
	Audio   map[string]string `json:"audio"`
}

// AppSettings represents application settings
type AppSettings struct {
	ScreenshotCount          int    `json:"screenshotCount"`
	FastpicSID               string `json:"fastpicSid"`
	ScreenshotQuality        int    `json:"screenshotQuality"`
	MaxConcurrentScreenshots int    `json:"maxConcurrentScreenshots"` // Max parallel screenshot generation
	MaxConcurrentUploads     int    `json:"maxConcurrentUploads"`     // Max parallel uploads
	MtnArgs                  string `json:"mtnArgs"`                  // MTN command line arguments
	ImageMiniatureSize       int    `json:"imageMiniatureSize"`
}

// TemplateData represents data for template processing
type TemplateData struct {
	Movies   []Movie     `json:"movies"`
	Settings AppSettings `json:"settings"`
	Template string      `json:"template"`
}
