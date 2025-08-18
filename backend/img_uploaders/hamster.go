package img_uploaders

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"time"

	http "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
)

type HamsterService struct {
	email     string
	password  string
	authToken string
	loggedIn  bool
	client    tls_client.HttpClient
}

type HamsterUploadResult struct {
	ID           string `json:"id"`
	URL          string `json:"url"`
	ViewerURL    string `json:"viewer_url"`
	ThumbnailURL string `json:"thumbnail_url"`
	BBThumb      string `json:"bbThumb"`
	BBBig        string `json:"bbBig"`
}

// HamsterResponse represents the JSON response structure from hamster.is
type HamsterResponse struct {
	Image struct {
		URL       string `json:"url"`
		URLViewer string `json:"url_viewer"`
		Thumb     struct {
			URL string `json:"url"`
		} `json:"thumb"`
	} `json:"image"`
}

func NewHamsterService(email, password string) *HamsterService {
	jar := tls_client.NewCookieJar()
	options := []tls_client.HttpClientOption{
		tls_client.WithTimeoutSeconds(60),
		tls_client.WithClientProfile(profiles.Chrome_120),
		tls_client.WithCookieJar(jar),
	}

	client, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), options...)
	if err != nil {
		log.Printf("Failed to create TLS client: %v", err)
		return nil
	}

	return &HamsterService{
		email:    email,
		password: password,
		client:   client,
	}
}

// extractAuthToken extracts auth_token from JavaScript in the page
func (h *HamsterService) extractAuthToken(htmlContent string) (string, error) {
	// Look for PF.obj.config.auth_token = "token_value";
	pattern := `PF\.obj\.config\.auth_token\s*=\s*"([^"]+)"`
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(htmlContent)
	if len(matches) < 2 {
		return "", fmt.Errorf("auth token not found in page")
	}
	return matches[1], nil
}

// login performs authentication with hamster.is
func (h *HamsterService) Login(ctx context.Context) error {
	if h.loggedIn {
		return nil
	}

	// Step 1: Get the homepage to extract auth_token
	req, err := http.NewRequest(http.MethodGet, "https://hamster.is/", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header = http.Header{
		"accept":          {"text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8"},
		"accept-encoding": {"gzip, deflate, br, zstd"},
		"connection":      {"keep-alive"},
		"user-agent":      {"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/139.0.0.0 Safari/537.36"},
		http.HeaderOrderKey: {
			"accept",
			"accept-encoding",
			"connection",
			"user-agent",
		},
	}

	resp, err := h.client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("request cancelled: %v", ctx.Err())
		}
		return fmt.Errorf("failed to load hamster.is homepage: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("hamster.is returned status code %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read homepage response: %v", err)
	}

	// Extract auth token from JavaScript
	authToken, err := h.extractAuthToken(string(body))
	if err != nil {
		return fmt.Errorf("failed to extract auth token: %v", err)
	}

	h.authToken = authToken
	log.Printf("Extracted auth token: %s", authToken[:10]+"...")

	// Step 2: Login with credentials using proper URL encoding
	formData := url.Values{}
	formData.Set("login-subject", h.email)
	formData.Set("password", h.password)
	formData.Set("auth_token", h.authToken)

	loginReq, err := http.NewRequest(http.MethodPost, "https://hamster.is/login", bytes.NewBufferString(formData.Encode()))
	if err != nil {
		return fmt.Errorf("failed to create login request: %v", err)
	}

	loginReq.Header = http.Header{
		"accept":          {"text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8"},
		"accept-encoding": {"gzip, deflate, br, zstd"},
		"content-type":    {"application/x-www-form-urlencoded"},
		"connection":      {"keep-alive"},
		"origin":          {"https://hamster.is"},
		"referer":         {"https://hamster.is/"},
		"user-agent":      {"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/139.0.0.0 Safari/537.36"},
		http.HeaderOrderKey: {
			"accept",
			"accept-encoding",
			"content-type",
			"connection",
			"origin",
			"referer",
			"user-agent",
		},
	}

	// Note: Python code uses allow_redirects=True, so we should handle redirects
	loginResp, err := h.client.Do(loginReq)
	if err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("login request cancelled: %v", ctx.Err())
		}
		return fmt.Errorf("login request failed: %v", err)
	}
	defer loginResp.Body.Close()

	// Handle redirects manually if needed (check if it's a 3xx response)
	if loginResp.StatusCode >= 300 && loginResp.StatusCode < 400 {
		location := loginResp.Header.Get("Location")
		log.Printf("Login returned redirect %d to: %s", loginResp.StatusCode, location)

		if location != "" {
			// Follow the redirect
			redirectReq, err := http.NewRequest(http.MethodGet, location, nil)
			if err != nil {
				return fmt.Errorf("failed to create redirect request: %v", err)
			}

			// Copy headers from original request (except method-specific ones)
			redirectReq.Header.Set("accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8")
			redirectReq.Header.Set("user-agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/139.0.0.0 Safari/537.36")
			redirectReq.Header.Set("referer", "https://hamster.is/login")

			loginResp.Body.Close() // Close the redirect response
			loginResp, err = h.client.Do(redirectReq)
			if err != nil {
				return fmt.Errorf("failed to follow redirect: %v", err)
			}
			defer loginResp.Body.Close()
		}
	}

	// Debug: Print login response status
	log.Printf("Login response status: %d", loginResp.StatusCode)

	// Check for KEEP_LOGIN cookie in the client's cookie jar (not just the response)
	// The cookie gets stored in the session during redirects
	hamsterURL, _ := url.Parse("https://hamster.is/")
	cookies := h.client.GetCookieJar().Cookies(hamsterURL)

	log.Printf("Cookies in session for hamster.is:")
	keepLoginFound := false
	for _, cookie := range cookies {
		log.Printf("Session Cookie: %s = %s", cookie.Name, cookie.Value)
		if cookie.Name == "KEEP_LOGIN" {
			keepLoginFound = true
			log.Printf("Found KEEP_LOGIN cookie in session!")
		}
	}

	if !keepLoginFound {
		loginBody, _ := io.ReadAll(loginResp.Body)
		return fmt.Errorf("login failed: status %d - KEEP_LOGIN cookie not found in session. Response: %s",
			loginResp.StatusCode, string(loginBody))
	}

	log.Printf("Login successful!")
	h.loggedIn = true
	return nil
}

// getTimestamp returns current timestamp in milliseconds
func (h *HamsterService) getTimestamp() string {
	return strconv.FormatInt(time.Now().UnixMilli(), 10)
}

// uploadToHamster uploads an image to hamster.is
func (h *HamsterService) uploadToHamster(ctx context.Context, filePath, fileName string) (*HamsterUploadResult, error) {
	log.Printf("Starting upload of %s to hamster.is...", fileName)

	if !h.loggedIn {
		if err := h.Login(ctx); err != nil {
			return nil, fmt.Errorf("failed to login: %v", err)
		}
	}

	if _, err := os.Stat(filePath); err != nil {
		return nil, fmt.Errorf("failed to stat file: %v", err)
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	// Determine content type
	contentType := "application/octet-stream"
	ext := filepath.Ext(fileName)
	switch ext {
	case ".jpg", ".jpeg":
		contentType = "image/jpeg"
	case ".png":
		contentType = "image/png"
	case ".gif":
		contentType = "image/gif"
	case ".webp":
		contentType = "image/webp"
	case ".bmp":
		contentType = "image/bmp"
	}

	timestamp := h.getTimestamp()

	var buffer bytes.Buffer
	writer := multipart.NewWriter(&buffer)

	// Add the image file first
	part, err := writer.CreateFormFile("source", fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %v", err)
	}

	if _, err := io.Copy(part, file); err != nil {
		return nil, fmt.Errorf("failed to copy file data: %v", err)
	}

	// Add all required form fields
	fields := map[string]string{
		"type":       "file",
		"action":     "upload",
		"timestamp":  timestamp,
		"auth_token": h.authToken,
		"nsfw":       "1",
		"mimetype":   contentType,
		"checksum":   "",
	}

	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			return nil, fmt.Errorf("failed to write form field %s: %v", key, err)
		}
	}

	writer.Close()

	req, err := http.NewRequest(http.MethodPost, "https://hamster.is/json", &buffer)
	if err != nil {
		return nil, fmt.Errorf("failed to create upload request: %v", err)
	}

	req.Header = http.Header{
		"accept":         {"application/json"},
		"content-type":   {writer.FormDataContentType()},
		"origin":         {"https://hamster.is"},
		"referer":        {"https://hamster.is/"},
		"sec-fetch-dest": {"empty"},
		"sec-fetch-mode": {"cors"},
		"sec-fetch-site": {"same-origin"},
		"user-agent":     {"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/139.0.0.0 Safari/537.36"},
		http.HeaderOrderKey: {
			"accept",
			"content-type",
			"origin",
			"referer",
			"sec-fetch-dest",
			"sec-fetch-mode",
			"sec-fetch-site",
			"user-agent",
		},
	}

	log.Printf("Uploading: %s", fileName)
	log.Printf("Content-Type: %s", contentType)
	log.Printf("Timestamp: %s", timestamp)
	log.Printf("Auth Token: %s", h.authToken[:10]+"...")

	resp, err := h.client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("upload cancelled: %v", ctx.Err())
		}
		return nil, fmt.Errorf("upload request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("upload failed: status %d - %s", resp.StatusCode, string(respBody))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read upload response: %v", err)
	}

	log.Printf("Hamster response: %s", string(body))

	var respJSON HamsterResponse
	if err := json.Unmarshal(body, &respJSON); err != nil {
		return nil, fmt.Errorf("failed to parse upload JSON: %v", err)
	}

	result := &HamsterUploadResult{
		URL:          respJSON.Image.URL,
		ViewerURL:    respJSON.Image.URLViewer,
		ThumbnailURL: respJSON.Image.Thumb.URL,
	}

	// Generate BBCode formats
	result.BBThumb = fmt.Sprintf("[URL=%s][IMG]%s[/IMG][/URL]", result.ViewerURL, result.ThumbnailURL)
	result.BBBig = fmt.Sprintf("[URL=%s][IMG]%s[/IMG][/URL]", result.ViewerURL, result.URL)

	log.Printf("Upload completed. URL: %s, Viewer: %s, Thumbnail: %s",
		result.URL, result.ViewerURL, result.ThumbnailURL)

	return result, nil
}

// UploadImage is the main public method to upload an image
func (h *HamsterService) UploadImage(ctx context.Context, filePath string) (*HamsterUploadResult, error) {
	fileName := filepath.Base(filePath)

	// Validate file exists and get basic info
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("file not found: %v", err)
	}

	if fileInfo.Size() == 0 {
		return nil, fmt.Errorf("file is empty")
	}

	// Check if file extension suggests it's an image
	ext := filepath.Ext(fileName)
	validExts := []string{".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp"}
	isValidExt := slices.Contains(validExts, ext)

	if !isValidExt {
		log.Printf("Warning: file %s may not be a valid image format", fileName)
	}

	return h.uploadToHamster(ctx, filePath, fileName)
}
