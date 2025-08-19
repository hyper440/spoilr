package backend

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/structs"
	"github.com/knadh/koanf/v2"
)

type SpoilerConfig struct {
	ScreenshotCount          int              `json:"screenshotCount" koanf:"screenshot_count"`
	FastpicSID               string           `json:"fastpicSid" koanf:"fastpic_sid"`
	ScreenshotQuality        int              `json:"screenshotQuality" koanf:"screenshot_quality"`
	MaxConcurrentScreenshots int              `json:"maxConcurrentScreenshots" koanf:"max_concurrent_screenshots"`
	MaxConcurrentUploads     int              `json:"maxConcurrentUploads" koanf:"max_concurrent_uploads"`
	CurrentPresetID          string           `json:"currentPresetId" koanf:"current_preset_id"`
	TemplatePresets          []TemplatePreset `json:"templatePresets" koanf:"template_presets"`
	MtnArgs                  string           `json:"mtnArgs" koanf:"mtn_args"`
	ImageMiniatureSize       int              `json:"imageMiniatureSize" koanf:"image_miniature_size"`
	// Hamster settings
	HamsterEmail    string `json:"hamsterEmail" koanf:"hamster_email"`
	HamsterPassword string `json:"hamsterPassword" koanf:"hamster_password"`
}

var SpoilerAppConfig SpoilerConfig
var ConfigPath string

func getDefaultTemplate() string {
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

func getDefaultPresets() []TemplatePreset {
	return []TemplatePreset{
		{
			ID:       "default-pl",
			Name:     "PL Default",
			Template: getDefaultTemplate(),
		},
		{
			ID:   "default-emp",
			Name: "EMP Default",
			Template: `[spoiler=%FILE_NAME% | %FILE_SIZE%]
File: %FILE_NAME%
Size: %FILE_SIZE%
Duration: %DURATION%
Video: %VIDEO_CODEC% / %VIDEO_FPS% FPS / %WIDTH%x%HEIGHT% / %VIDEO_BIT_RATE%
Audio: %AUDIO_CODEC% / %AUDIO_SAMPLE_RATE% / %AUDIO_CHANNELS% / %AUDIO_BIT_RATE%

%CONTACT_SHEET_HAM%

%SCREENSHOTS_HAM%
[/spoiler]`,
		},
	}
}

var DefaultSpoilerConfig = SpoilerConfig{
	ScreenshotCount:          6,
	FastpicSID:               "",
	ScreenshotQuality:        2,
	MaxConcurrentScreenshots: 3,
	MaxConcurrentUploads:     2,
	CurrentPresetID:          "default-pl",
	TemplatePresets:          getDefaultPresets(),
	MtnArgs:                  "-b 2 -w 1200 -c 4 -r 4 -g 0 -k 1C1C1C -L 4:2 -F F0FFFF:10",
	ImageMiniatureSize:       350,
	HamsterEmail:             "",
	HamsterPassword:          "",
}

type ConfigService struct{}

func NewConfigService() *ConfigService {
	return &ConfigService{}
}

func (g *ConfigService) GetConfig() SpoilerConfig {
	initSpoilerConfigPath()
	if _, err := os.Stat(ConfigPath); os.IsNotExist(err) {
		fmt.Println("Created a new spoiler settings config")
		SpoilerAppConfig = DefaultSpoilerConfig
		saveSpoilerAppConfig()
	}

	file, _ := os.ReadFile(ConfigPath)
	if len(file) == 0 {
		fmt.Println("config file is empty")
		SpoilerAppConfig = DefaultSpoilerConfig
	} else {
		SpoilerAppConfig = loadSpoilerAppConfig()
	}

	log.Println("Spoiler Config", SpoilerAppConfig)
	return SpoilerAppConfig
}

func (g *ConfigService) UpdateConfig(config SpoilerConfig) error {
	// Validate some values
	if config.ScreenshotCount < 0 || config.ScreenshotCount > 20 {
		return fmt.Errorf("screenshot count must be between 0 and 20")
	}
	if config.MaxConcurrentScreenshots < 1 {
		return fmt.Errorf("max concurrent screenshots must be at least 1")
	}
	if config.MaxConcurrentUploads < 1 {
		return fmt.Errorf("max concurrent uploads must be at least 1")
	}
	if config.ScreenshotQuality < 1 || config.ScreenshotQuality > 31 {
		return fmt.Errorf("screenshot quality must be between 1 and 31")
	}
	if config.ImageMiniatureSize < 100 || config.ImageMiniatureSize > 800 {
		return fmt.Errorf("image miniature size must be between 100 and 800")
	}

	// Ensure we always have at least one preset
	if len(config.TemplatePresets) == 0 {
		config.TemplatePresets = getDefaultPresets()
		config.CurrentPresetID = "default-pl"
	}

	// Validate current preset ID exists
	found := false
	for _, preset := range config.TemplatePresets {
		if preset.ID == config.CurrentPresetID {
			found = true
			break
		}
	}
	if !found {
		config.CurrentPresetID = config.TemplatePresets[0].ID
	}

	SpoilerAppConfig = config
	return saveSpoilerAppConfig()
}

func (g *ConfigService) SaveTemplatePreset(preset TemplatePreset) error {
	config := g.GetConfig()

	found := false
	for i, p := range config.TemplatePresets {
		if p.ID == preset.ID {
			config.TemplatePresets[i] = preset
			found = true
			break
		}
	}
	if !found {
		// Generate ID if not provided
		if preset.ID == "" {
			preset.ID = uuid.New().String()
		}
		config.TemplatePresets = append(config.TemplatePresets, preset)
	}

	return g.UpdateConfig(config)
}

func (g *ConfigService) DeleteTemplatePreset(presetID string) error {
	config := g.GetConfig()

	// Don't allow deletion if only one preset left
	if len(config.TemplatePresets) <= 1 {
		return fmt.Errorf("cannot delete the last template preset")
	}

	// Find and remove the preset
	for i, preset := range config.TemplatePresets {
		if preset.ID == presetID {
			config.TemplatePresets = append(config.TemplatePresets[:i], config.TemplatePresets[i+1:]...)

			// If we deleted the current preset, switch to first available
			if config.CurrentPresetID == presetID {
				config.CurrentPresetID = config.TemplatePresets[0].ID
			}

			return g.UpdateConfig(config)
		}
	}

	return fmt.Errorf("preset not found")
}

func (g *ConfigService) SetCurrentPreset(presetID string) error {
	config := g.GetConfig()

	// Find the preset
	for _, preset := range config.TemplatePresets {
		if preset.ID == presetID {
			config.CurrentPresetID = presetID
			return g.UpdateConfig(config)
		}
	}

	return fmt.Errorf("preset not found")
}

func (g *ConfigService) GetCurrentTemplate() string {
	config := g.GetConfig()

	// Find current preset
	for _, preset := range config.TemplatePresets {
		if preset.ID == config.CurrentPresetID {
			return preset.Template
		}
	}

	// Fallback to first preset if current preset not found
	if len(config.TemplatePresets) > 0 {
		return config.TemplatePresets[0].Template
	}

	// Ultimate fallback
	return getDefaultTemplate()
}

func initSpoilerConfigPath() {
	// First try portable config in executable directory
	wdDir, err := os.Getwd()
	if err == nil {
		portableConfigPath := filepath.Join(wdDir, "spoilr.config")

		// If config exists in executable directory, use it
		if _, err := os.Stat(portableConfigPath); err == nil {
			ConfigPath = portableConfigPath
			return
		}
	}

	// Fall back to default location
	userConfigDir := filepath.Join(getUserConfigDir(), "/spoilr")
	ConfigPath = filepath.Join(userConfigDir, "spoilr.config")
}

func getUserConfigDir() string {
	dirname, err := os.UserConfigDir()
	if err != nil {
		log.Fatal(err)
	}
	return dirname
}

func saveSpoilerAppConfig() error {
	initSpoilerConfigPath()
	k := koanf.New(".")

	err := k.Load(structs.Provider(SpoilerAppConfig, "koanf"), nil)
	if err != nil {
		fmt.Println(err)
		return err
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(ConfigPath), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %v", err)
	}

	b, err := k.Marshal(yaml.Parser())
	if err != nil {
		fmt.Println(err)
		return err
	}

	err = os.WriteFile(ConfigPath, b, 0644)
	if err != nil {
		fmt.Println(err)
		return err
	}

	return nil
}

func loadSpoilerAppConfig() SpoilerConfig {
	var c SpoilerConfig
	var k = koanf.New(".")
	if err := k.Load(file.Provider(ConfigPath), yaml.Parser()); err != nil {
		log.Printf("error parsing spoiler app config: %v", err)
		return DefaultSpoilerConfig
	}
	err := k.Unmarshal("", &c)
	if err != nil {
		log.Printf("error unmarshaling spoiler app config: %v", err)
		return DefaultSpoilerConfig
	}

	// Validate and set defaults for invalid values
	if c.ScreenshotCount < 0 || c.ScreenshotCount > 20 {
		c.ScreenshotCount = DefaultSpoilerConfig.ScreenshotCount
	}
	if c.MaxConcurrentScreenshots < 1 {
		c.MaxConcurrentScreenshots = DefaultSpoilerConfig.MaxConcurrentScreenshots
	}
	if c.MaxConcurrentUploads < 1 {
		c.MaxConcurrentUploads = DefaultSpoilerConfig.MaxConcurrentUploads
	}
	if c.ScreenshotQuality < 1 || c.ScreenshotQuality > 31 {
		c.ScreenshotQuality = DefaultSpoilerConfig.ScreenshotQuality
	}
	if c.ImageMiniatureSize < 100 || c.ImageMiniatureSize > 800 {
		c.ImageMiniatureSize = DefaultSpoilerConfig.ImageMiniatureSize
	}
	if c.MtnArgs == "" {
		c.MtnArgs = DefaultSpoilerConfig.MtnArgs
	}

	// Ensure we have presets and current preset ID
	if len(c.TemplatePresets) == 0 {
		c.TemplatePresets = getDefaultPresets()
	}
	if c.CurrentPresetID == "" {
		c.CurrentPresetID = c.TemplatePresets[0].ID
	}

	// Validate current preset ID exists
	found := false
	for _, preset := range c.TemplatePresets {
		if preset.ID == c.CurrentPresetID {
			found = true
			break
		}
	}
	if !found {
		c.CurrentPresetID = c.TemplatePresets[0].ID
	}

	return c
}
