//go:build windows

package backend

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/sqweek/dialog"
	"golang.org/x/sys/windows/registry"
)

const (
	kMinimumCompatibleVersion = "86.0.616.0"
	kInstallKeyPath           = "Software\\Microsoft\\EdgeUpdate\\ClientState\\"
)

var (
	// WebView2 channel UUIDs (stable, beta, dev, canary)
	kChannelUuids = []string{
		"{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}", // stable
		"{2CD8A007-E189-409D-A2C8-9AF4EF3C72AA}", // beta
		"{0D50BFEC-CD6A-4F9A-964C-C7416E3ACB10}", // dev
		"{65C35B14-6C1D-4122-AC46-7148CC9D6497}", // canary
	}
)

// WebView2Version represents a WebView2 version
type WebView2Version struct {
	Major, Minor, Build, Patch int
	Channel                    string
	Path                       string
}

// String returns the version as a string
func (v WebView2Version) String() string {
	return fmt.Sprintf("%d.%d.%d.%d", v.Major, v.Minor, v.Build, v.Patch)
}

// Compare compares this version with another version
// Returns: -1 if this < other, 0 if equal, 1 if this > other
func (v WebView2Version) Compare(other WebView2Version) int {
	if v.Major != other.Major {
		if v.Major < other.Major {
			return -1
		}
		return 1
	}
	if v.Minor != other.Minor {
		if v.Minor < other.Minor {
			return -1
		}
		return 1
	}
	if v.Build != other.Build {
		if v.Build < other.Build {
			return -1
		}
		return 1
	}
	if v.Patch != other.Patch {
		if v.Patch < other.Patch {
			return -1
		}
		return 1
	}
	return 0
}

// WebView2Handler handles WebView2 detection and installation
type WebView2Handler struct {
	installerData []byte
}

// NewWebView2Handler creates a new WebView2Handler
func NewWebView2Handler(installerData []byte) *WebView2Handler {
	return &WebView2Handler{
		installerData: installerData,
	}
}

// parseVersion parses a version string like "86.0.616.0"
func parseVersion(versionStr string) (WebView2Version, error) {
	parts := strings.Split(versionStr, ".")
	if len(parts) != 4 {
		return WebView2Version{}, fmt.Errorf("invalid version format: %s", versionStr)
	}

	var version WebView2Version
	var err error

	if version.Major, err = strconv.Atoi(parts[0]); err != nil {
		return WebView2Version{}, fmt.Errorf("invalid major version: %s", parts[0])
	}
	if version.Minor, err = strconv.Atoi(parts[1]); err != nil {
		return WebView2Version{}, fmt.Errorf("invalid minor version: %s", parts[1])
	}
	if version.Build, err = strconv.Atoi(parts[2]); err != nil {
		return WebView2Version{}, fmt.Errorf("invalid build version: %s", parts[2])
	}
	if version.Patch, err = strconv.Atoi(parts[3]); err != nil {
		return WebView2Version{}, fmt.Errorf("invalid patch version: %s", parts[3])
	}

	return version, nil
}

// findWebView2Installation looks for WebView2 installation in registry
func (h *WebView2Handler) findWebView2Installation() (*WebView2Version, error) {
	minimumVersion, err := parseVersion(kMinimumCompatibleVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to parse minimum version: %w", err)
	}

	// Check each channel
	for _, channelUuid := range kChannelUuids {
		keyPath := kInstallKeyPath + channelUuid

		// Try both HKLM and HKCU, both 64-bit and 32-bit registry views
		for _, rootKey := range []registry.Key{registry.LOCAL_MACHINE, registry.CURRENT_USER} {
			for _, access := range []uint32{registry.READ, registry.READ | registry.WOW64_32KEY} {
				version, err := h.checkRegistryPath(rootKey, keyPath, access)
				if err != nil {
					continue // Try next combination
				}

				// Check if version meets minimum requirement
				if version.Compare(minimumVersion) >= 0 {
					// Determine channel name
					switch channelUuid {
					case "{F3017226-FE2A-4295-8BDF-00C3A9A7E4C5}":
						version.Channel = "stable"
					case "{2CD8A007-E189-409D-A2C8-9AF4EF3C72AA}":
						version.Channel = "beta"
					case "{0D50BFEC-CD6A-4F9A-964C-C7416E3ACB10}":
						version.Channel = "dev"
					case "{65C35B14-6C1D-4122-AC46-7148CC9D6497}":
						version.Channel = "canary"
					default:
						version.Channel = "unknown"
					}

					return version, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("WebView2 not found or version too old")
}

// checkRegistryPath checks a specific registry path for WebView2
func (h *WebView2Handler) checkRegistryPath(rootKey registry.Key, keyPath string, access uint32) (*WebView2Version, error) {
	regKey, err := registry.OpenKey(rootKey, keyPath, access)
	if err != nil {
		return nil, err
	}
	defer regKey.Close()

	// Get the EBWebView value which contains the installation path
	embeddedEdgeSubFolder, _, err := regKey.GetStringValue("EBWebView")
	if err != nil {
		return nil, err
	}

	if embeddedEdgeSubFolder == "" {
		return nil, fmt.Errorf("empty EBWebView value")
	}

	// Extract version from path (version is typically the last folder name)
	versionString := filepath.Base(embeddedEdgeSubFolder)
	version, err := parseVersion(versionString)
	if err != nil {
		return nil, err
	}

	// Verify the installation path exists
	if _, err := os.Stat(embeddedEdgeSubFolder); err != nil {
		return nil, fmt.Errorf("WebView2 path does not exist: %s", embeddedEdgeSubFolder)
	}

	version.Path = embeddedEdgeSubFolder
	return &version, nil
}

// CheckWebView2 checks if WebView2 is available on the system
func (h *WebView2Handler) CheckWebView2() (*WebView2Version, error) {
	return h.findWebView2Installation()
}

// RunInstaller extracts and runs the embedded WebView2 installer
func (h *WebView2Handler) RunInstaller() error {
	if len(h.installerData) == 0 {
		return fmt.Errorf("no installer data provided")
	}

	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "recrafter_webview2_*")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir) // Clean up when done

	// Write installer to temp file
	installerPath := filepath.Join(tempDir, "MicrosoftEdgeWebview2Setup.exe")
	if err := os.WriteFile(installerPath, h.installerData, 0755); err != nil {
		return fmt.Errorf("failed to write installer: %w", err)
	}

	log.Printf("Running WebView2 installer: %s", installerPath)

	// Run the installer
	cmd := exec.Command(installerPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("installer failed: %w", err)
	}

	return nil
}

// HandleMissingWebView2 shows dialog and handles user choice for missing WebView2
func (h *WebView2Handler) HandleMissingWebView2() error {
	message := `WebView2 Runtime is required to run this application.

WebView2 Runtime is not installed on your system. Would you like to install it now?

Click "Yes" to run the installer, or "No" to close the application.`

	// Show dialog with Install/Exit options
	result := dialog.Message("%s", message).
		Title("WebView2 Runtime Required").
		YesNo()

	if !result {
		// User chose "No" (Exit)
		log.Println("User chose not to install WebView2. Exiting...")
		os.Exit(1)
	}

	// User chose "Yes" (Install)
	log.Println("User chose to install WebView2. Running installer...")

	// Show progress dialog
	go func() {
		dialog.Message("Installing WebView2 Runtime...\n\nPlease wait while the installer completes.").
			Title("Installing WebView2").
			Info()
	}()

	// Run the installer
	if err := h.RunInstaller(); err != nil {
		log.Printf("Failed to run WebView2 installer: %v", err)
		return fmt.Errorf("failed to install WebView2 Runtime: %w\n\nPlease download and install WebView2 manually from:\nhttps://developer.microsoft.com/microsoft-edge/webview2/", err)
	}

	// Verify installation
	if _, err := h.CheckWebView2(); err != nil {
		log.Printf("WebView2 installation verification failed: %v", err)
		return fmt.Errorf("WebView2 installation completed, but WebView2 Runtime could not be detected.\n\nPlease restart the application or install WebView2 manually from:\nhttps://developer.microsoft.com/microsoft-edge/webview2/")
	}

	log.Println("WebView2 installation completed successfully")
	dialog.Message("WebView2 Runtime has been installed successfully!\n\nThe application will now continue loading.").
		Title("Installation Complete").
		Info()

	return nil
}

// EnsureWebView2Available checks for WebView2 and handles installation if missing
func (h *WebView2Handler) EnsureWebView2Available() (*WebView2Version, error) {
	log.Println("Checking WebView2 availability...")

	version, err := h.CheckWebView2()
	if err != nil {
		log.Printf("WebView2 not found: %v", err)

		if installErr := h.HandleMissingWebView2(); installErr != nil {
			return nil, installErr
		}

		// Double-check after installation attempt
		version, err = h.CheckWebView2()
		if err != nil {
			log.Printf("WebView2 still not available after installation attempt: %v", err)
			return nil, fmt.Errorf("WebView2 Runtime is still not available.\n\nPlease install WebView2 manually from:\nhttps://developer.microsoft.com/microsoft-edge/webview2/")
		}
	}

	log.Printf("WebView2 is available: %s (%s channel) at %s", version.String(), version.Channel, version.Path)
	return version, nil
}
