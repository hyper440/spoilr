package backend

// Movie represents a media file with its metadata
type Movie struct {
	ID                string            `json:"id"`
	FileName          string            `json:"fileName"`
	FilePath          string            `json:"filePath"`
	FileSize          string            `json:"fileSize"`
	FileSizeBytes     int64             `json:"fileSizeBytes"`
	Duration          string            `json:"duration"`
	Width             string            `json:"width"`
	Height            string            `json:"height"`
	BitRate           string            `json:"bitRate"`
	VideoBitRate      string            `json:"videoBitRate"`
	AudioBitRate      string            `json:"audioBitRate"`
	VideoCodec        string            `json:"videoCodec"`
	AudioCodec        string            `json:"audioCodec"`
	ScreenshotURLs    []string          `json:"screenshotUrls"`    // BBThumb URLs
	ScreenshotBigURLs []string          `json:"screenshotBigUrls"` // BBBig URLs
	ScreenshotAlbum   string            `json:"screenshotAlbum"`
	ThumbnailURL      string            `json:"thumbnailUrl"`    // BBThumb URL
	ThumbnailBigURL   string            `json:"thumbnailBigUrl"` // BBBig URL
	Params            map[string]string `json:"params"`
	ProcessingState   string            `json:"processingState"`           // State constants defined below
	ProcessingError   string            `json:"processingError,omitempty"` // Error details if processing fails
	Errors            []string          `json:"errors,omitempty"`          // Individual errors that occurred during processing
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
}

// FastpicUploadResult represents the result of fastpic upload
type FastpicUploadResult struct {
	AlbumLink string `json:"albumLink"`
	Direct    string `json:"direct"`
	BBThumb   string `json:"bbThumb"`
	BBBig     string `json:"bbBig"`
}

// TemplateData represents data for template processing
type TemplateData struct {
	Movies   []Movie     `json:"movies"`
	Settings AppSettings `json:"settings"`
	Template string      `json:"template"`
}
