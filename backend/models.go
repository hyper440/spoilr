package backend

// Movie represents a media file with its metadata
type Movie struct {
	ID              int               `json:"id"`
	FileName        string            `json:"fileName"`
	FilePath        string            `json:"filePath"`
	FileSize        string            `json:"fileSize"`
	FileSizeBytes   int64             `json:"fileSizeBytes"`
	Duration        string            `json:"duration"`
	Width           string            `json:"width"`
	Height          string            `json:"height"`
	BitRate         string            `json:"bitRate"`
	VideoBitRate    string            `json:"videoBitRate"`
	AudioBitRate    string            `json:"audioBitRate"`
	VideoCodec      string            `json:"videoCodec"`
	AudioCodec      string            `json:"audioCodec"`
	ScreenshotURLs  []string          `json:"screenshotUrls"`
	ScreenshotAlbum string            `json:"screenshotAlbum"`
	Params          map[string]string `json:"params"`
	ProcessingState string            `json:"processingState"` // "pending", "processing", "completed", "error"
}

// MediaInfo represents extracted media information
type MediaInfo struct {
	General map[string]string `json:"general"`
	Video   map[string]string `json:"video"`
	Audio   map[string]string `json:"audio"`
}

// ProcessProgress represents file processing progress
type ProcessProgress struct {
	Current   int    `json:"current"`
	Total     int    `json:"total"`
	FileName  string `json:"fileName"`
	Message   string `json:"message"`
	Completed bool   `json:"completed"`
	Error     string `json:"error,omitempty"`
}

// AppSettings represents application settings
type AppSettings struct {
	CenterAlign       bool   `json:"centerAlign"`
	HideEmpty         bool   `json:"hideEmpty"`
	UIFontSize        int    `json:"uiFontSize"`
	ListFontSize      int    `json:"listFontSize"`
	TextFontSize      int    `json:"textFontSize"`
	ScreenshotCount   int    `json:"screenshotCount"`
	FastpicSID        string `json:"fastpicSid"`
	ScreenshotQuality int    `json:"screenshotQuality"`
}

// FastpicUploadResult represents the result of fastpic upload
type FastpicUploadResult struct {
	AlbumLink string `json:"albumLink"`
	Direct    string `json:"direct"`
	BBThumb   string `json:"bbThumb"`
	BBBig     string `json:"bbBig"`
	HTMLThumb string `json:"htmlThumb"`
	MDThumb   string `json:"mdThumb"`
}

// TemplateData represents data for template processing
type TemplateData struct {
	Movies   []Movie     `json:"movies"`
	Settings AppSettings `json:"settings"`
	Template string      `json:"template"`
}
