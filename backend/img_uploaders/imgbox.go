package img_uploaders

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"os"
	"path/filepath"
	"slices"
	"strconv"

	"github.com/PuerkitoBio/goquery"
	http "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
)

type ImgboxService struct {
	imageMiniatureSize int
	csrfToken          string
	tokenID            string
	tokenSecret        string
	client             tls_client.HttpClient
}

type ImgboxUploadResult struct {
	ID           string `json:"id"`
	Slug         string `json:"slug"`
	Name         string `json:"name"`
	URL          string `json:"url"`
	OriginalURL  string `json:"original_url"`
	ThumbnailURL string `json:"thumbnail_url"`
	SquareURL    string `json:"square_url"`
	CreatedAt    string `json:"created_at"`
	BBThumb      string `json:"bbThumb"`
	BBBig        string `json:"bbBig"`
}

type ImgboxResponse struct {
	Files []ImgboxUploadResult `json:"files"`
}

func NewImgboxService(imageMiniatureSize int) *ImgboxService {
	jar := tls_client.NewCookieJar()
	options := []tls_client.HttpClientOption{
		tls_client.WithTimeoutSeconds(60),
		tls_client.WithClientProfile(profiles.Chrome_120),
		tls_client.WithNotFollowRedirects(),
		tls_client.WithCookieJar(jar),
	}

	client, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), options...)
	if err != nil {
		log.Printf("Failed to create TLS client: %v", err)
		return nil
	}

	return &ImgboxService{
		imageMiniatureSize: imageMiniatureSize,
		client:             client,
	}
}

// Replace the tokenResponse struct and unmarshaling logic in the initializeTokens method
func (i *ImgboxService) initializeTokens(ctx context.Context) error {
	// Step 1: Get CSRF token from homepage
	req, err := http.NewRequest(http.MethodGet, "https://imgbox.com/", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header = http.Header{
		"accept":                    {"text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.7"},
		"accept-language":           {"en-US,en;q=0.9"},
		"connection":                {"keep-alive"},
		"host":                      {"imgbox.com"},
		"sec-ch-ua":                 {`"Chromium";v="136", "Google Chrome";v="136", "Not.A/Brand";v="99"`},
		"sec-ch-ua-mobile":          {"?0"},
		"sec-ch-ua-platform":        {`"Windows"`},
		"sec-fetch-dest":            {"document"},
		"sec-fetch-mode":            {"navigate"},
		"sec-fetch-site":            {"none"},
		"sec-fetch-user":            {"?1"},
		"upgrade-insecure-requests": {"1"},
		"user-agent":                {"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"},
		http.HeaderOrderKey: {
			"accept",
			"accept-language",
			"connection",
			"host",
			"sec-ch-ua",
			"sec-ch-ua-mobile",
			"sec-ch-ua-platform",
			"sec-fetch-dest",
			"sec-fetch-mode",
			"sec-fetch-site",
			"sec-fetch-user",
			"upgrade-insecure-requests",
			"user-agent",
		},
	}

	resp, err := i.client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("request cancelled: %v", ctx.Err())
		}
		return fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("imgbox returned status code %d", resp.StatusCode)
	}

	// Parse HTML to extract CSRF token
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to parse HTML: %v", err)
	}

	// Find authenticity_token input
	csrfToken, exists := doc.Find(`input[name="authenticity_token"]`).Attr("value")
	if !exists || csrfToken == "" {
		return fmt.Errorf("CSRF token not found")
	}

	i.csrfToken = csrfToken
	log.Printf("Successfully obtained CSRF token: %s", csrfToken[:10]+"...")

	// Step 2: Generate upload tokens
	req, err = http.NewRequest(http.MethodPost, "https://imgbox.com/ajax/token/generate", nil)
	if err != nil {
		return fmt.Errorf("failed to create token request: %v", err)
	}

	req.Header = http.Header{
		"x-csrf-token": {i.csrfToken},
		"content-type": {"application/x-www-form-urlencoded"},
		"user-agent":   {"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"},
		"accept":       {"*/*"},
		"origin":       {"https://imgbox.com"},
		"referer":      {"https://imgbox.com/"},
		http.HeaderOrderKey: {
			"accept",
			"content-type",
			"origin",
			"referer",
			"user-agent",
			"x-csrf-token",
		},
	}

	resp, err = i.client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return fmt.Errorf("token request cancelled: %v", ctx.Err())
		}
		return fmt.Errorf("token request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("token generation failed with status code %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read token response: %v", err)
	}

	// Modified struct to handle both string and number types for token_id
	var tokenResponse struct {
		TokenID     json.Number `json:"token_id"`
		TokenSecret string      `json:"token_secret"`
	}

	if err := json.Unmarshal(body, &tokenResponse); err != nil {
		return fmt.Errorf("failed to parse token JSON: %v", err)
	}

	// Convert json.Number to string
	tokenIDStr := string(tokenResponse.TokenID)
	if tokenIDStr == "" {
		return fmt.Errorf("missing token_id in response")
	}

	if tokenResponse.TokenSecret == "" {
		return fmt.Errorf("missing token_secret in response")
	}

	i.tokenID = tokenIDStr
	i.tokenSecret = tokenResponse.TokenSecret

	log.Printf("Successfully obtained upload tokens: ID=%s, Secret=%s",
		i.tokenID[:8]+"...", i.tokenSecret[:8]+"...")

	return nil
}

func (i *ImgboxService) uploadToImgbox(ctx context.Context, filePath, fileName string) (*ImgboxUploadResult, error) {
	log.Printf("Starting upload of %s to imgbox...", fileName)

	// Initialize tokens if not already done
	if i.csrfToken == "" || i.tokenID == "" || i.tokenSecret == "" {
		if err := i.initializeTokens(ctx); err != nil {
			return nil, fmt.Errorf("failed to initialize tokens: %v", err)
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

	var buffer bytes.Buffer
	writer := multipart.NewWriter(&buffer)

	// Add all form fields
	fields := map[string]string{
		"token_id":         i.tokenID,
		"token_secret":     i.tokenSecret,
		"content_type":     "2",
		"thumbnail_size":   strconv.Itoa(i.imageMiniatureSize) + "r",
		"gallery_id":       "null",
		"gallery_secret":   "null",
		"comments_enabled": "null",
	}

	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			return nil, fmt.Errorf("failed to write form field %s: %v", key, err)
		}
	}

	// Add the file
	part, err := writer.CreateFormFile("files[]", fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %v", err)
	}

	if _, err := io.Copy(part, file); err != nil {
		return nil, fmt.Errorf("failed to copy file data: %v", err)
	}

	writer.Close()

	req, err := http.NewRequest(http.MethodPost, "https://imgbox.com/upload/process", &buffer)
	if err != nil {
		return nil, fmt.Errorf("failed to create upload request: %v", err)
	}

	req.Header = http.Header{
		"content-type": {writer.FormDataContentType()},
		"user-agent":   {"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/136.0.0.0 Safari/537.36"},
		"accept":       {"*/*"},
		"origin":       {"https://imgbox.com"},
		"referer":      {"https://imgbox.com/"},
		http.HeaderOrderKey: {
			"accept",
			"content-type",
			"origin",
			"referer",
			"user-agent",
		},
	}

	resp, err := i.client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("upload cancelled: %v", ctx.Err())
		}
		return nil, fmt.Errorf("upload request failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read upload response: %v", err)
	}

	log.Printf("Imgbox response: %s", string(body))

	var respJSON ImgboxResponse
	if err := json.Unmarshal(body, &respJSON); err != nil {
		return nil, fmt.Errorf("failed to parse upload JSON: %v", err)
	}

	if len(respJSON.Files) == 0 {
		return &ImgboxUploadResult{}, fmt.Errorf("no files in upload response")
	}

	result := &respJSON.Files[0]

	result.BBThumb = fmt.Sprintf("[URL=%s][IMG]%s[/IMG][/URL]", result.URL, result.ThumbnailURL)
	result.BBBig = fmt.Sprintf("[URL=%s][IMG]%s[/IMG][/URL]", result.URL, result.OriginalURL)

	log.Printf("Upload completed. URL: %s, Original: %s, Thumbnail: %s",
		result.URL, result.OriginalURL, result.ThumbnailURL)

	return result, nil
}

// UploadImage is the main public method to upload an image
func (i *ImgboxService) UploadImage(ctx context.Context, filePath string) (*ImgboxUploadResult, error) {
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

	return i.uploadToImgbox(ctx, filePath, fileName)
}
