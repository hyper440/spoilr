package backend

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/structs"
	"github.com/knadh/koanf/v2"
)

type SpoilerConfig struct {
	ScreenshotCount          int    `json:"screenshotCount" koanf:"screenshot_count"`
	FastpicSID               string `json:"fastpicSid" koanf:"fastpic_sid"`
	ScreenshotQuality        int    `json:"screenshotQuality" koanf:"screenshot_quality"`
	MaxConcurrentScreenshots int    `json:"maxConcurrentScreenshots" koanf:"max_concurrent_screenshots"`
	MaxConcurrentUploads     int    `json:"maxConcurrentUploads" koanf:"max_concurrent_uploads"`
	Template                 string `json:"template" koanf:"template"`
	MtnArgs                  string `json:"mtnArgs" koanf:"mtn_args"`
	ImageMiniatureSize       int    `json:"imageMiniatureSize" koanf:"image_miniature_size"`
}

var SpoilerAppConfig SpoilerConfig
var ConfigPath string

var DefaultSpoilerConfig = SpoilerConfig{
	ScreenshotCount:          6,
	FastpicSID:               "",
	ScreenshotQuality:        2,
	MaxConcurrentScreenshots: 3,
	MaxConcurrentUploads:     2,
	Template:                 "",
	MtnArgs:                  "-b 2 -w 1200 -c 4 -r 4 -g 0 -k 1C1C1C -L 4:2 -F F0FFFF:10",
	ImageMiniatureSize:       350,
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

	SpoilerAppConfig = config
	return saveSpoilerAppConfig()
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
	if c.Template == "" {
		c.Template = DefaultSpoilerConfig.Template
	}
	if c.MtnArgs == "" {
		c.MtnArgs = DefaultSpoilerConfig.MtnArgs
	}

	return c
}
