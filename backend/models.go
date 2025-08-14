package backend

// Movie represents a media file with its metadata
type Movie struct {
	ID            int               `json:"id"`
	FileName      string            `json:"fileName"`
	FilePath      string            `json:"filePath"`
	FileSize      string            `json:"fileSize"`
	FileSizeBytes int64             `json:"fileSizeBytes"`
	Duration      string            `json:"duration"`
	Width         string            `json:"width"`
	Height        string            `json:"height"`
	ScreenListURL string            `json:"screenListURL"`
	Params        map[string]string `json:"params"`
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
	Language        string `json:"language"`
	ConvertToGB     bool   `json:"convertToGb"`
	CenterAlign     bool   `json:"centerAlign"`
	AcceptOnlyLinks bool   `json:"acceptOnlyLinks"`
	HideEmpty       bool   `json:"hideEmpty"`
	UIFontSize      int    `json:"uiFontSize"`
	ListFontSize    int    `json:"listFontSize"`
	TextFontSize    int    `json:"textFontSize"`
}

// LinkProcessResult represents the result of link processing
type LinkProcessResult struct {
	OriginalLinks  []string `json:"originalLinks"`
	ProcessedLinks []string `json:"processedLinks"`
	Errors         []string `json:"errors"`
}

// TemplateData represents data for template processing
type TemplateData struct {
	Movies   []Movie     `json:"movies"`
	Settings AppSettings `json:"settings"`
	Template string      `json:"template"`
}
