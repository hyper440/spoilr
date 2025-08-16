package img_uploaders

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"golang.org/x/net/html"
)

type FastpicService struct {
	sid                string
	uploadID           string
	imageMiniatureSize int
}

type FastpicUploadResult struct {
	AlbumLink string `json:"albumLink"`
	Direct    string `json:"direct"`
	BBThumb   string `json:"bbThumb"`
	BBBig     string `json:"bbBig"`
}

func NewFastpicService(sid string, imageMiniatureSize int) *FastpicService {
	return &FastpicService{
		sid:                sid,
		imageMiniatureSize: imageMiniatureSize,
	}
}

func (f *FastpicService) GetFastpicUploadID(ctx context.Context) error {
	client := &http.Client{Timeout: 30 * time.Second}

	req, err := http.NewRequestWithContext(ctx, "GET", "https://new.fastpic.org/", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	// Add fp_sid if available
	if f.sid != "" {
		req.AddCookie(&http.Cookie{Name: "fp_sid", Value: f.sid})
		log.Printf("Using existing fastpic SID for authentication")
	}

	resp, err := client.Do(req)
	if err != nil {
		// Check if error is due to context cancellation
		if ctx.Err() != nil {
			return fmt.Errorf("request cancelled: %v", ctx.Err())
		}
		return fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("fastpic returned status code %d", resp.StatusCode)
	}

	// If no SID was set, try to parse it from Set-Cookie
	if f.sid == "" {
		for _, cookie := range resp.Cookies() {
			if cookie.Name == "fp_sid" && cookie.Value != "" {
				f.sid = cookie.Value
				log.Printf("Automatically obtained fp_sid: %s", f.sid)
				break
			}
		}
		if f.sid == "" {
			log.Printf("Warning: could not find fp_sid in response cookies")
		}
	}

	// Parse HTML with goquery
	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to parse HTML: %v", err)
	}

	// Find <script> containing "upload_id"
	var scriptText string
	doc.Find("script").EachWithBreak(func(i int, s *goquery.Selection) bool {
		text := s.Text()
		if text != "" && regexp.MustCompile(`"upload_id"`).MatchString(text) {
			scriptText = text
			return false // stop iterating
		}
		return true
	})

	if scriptText == "" {
		return fmt.Errorf("could not find script containing upload_id")
	}

	// Extract upload_id using regex
	re := regexp.MustCompile(`"upload_id"\s*:\s*'([^']+)'`)
	matches := re.FindStringSubmatch(scriptText)
	if len(matches) < 2 {
		return fmt.Errorf("upload_id not found in script")
	}

	uploadID := matches[1]
	f.uploadID = uploadID
	log.Printf("Successfully obtained fastpic upload ID: %s", uploadID)
	return nil
}

// uploadToFastpic uploads image to fastpic
func (f *FastpicService) UploadToFastpic(ctx context.Context, filePath, fileName string) (*FastpicUploadResult, error) {
	log.Printf("Starting upload of %s to fastpic...", fileName)

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

	fields := map[string]string{
		"uploading":                 "1",
		"fp":                        "not-loaded",
		"upload_id":                 f.uploadID,
		"check_thumb":               "size",
		"thumb_text":                "",
		"thumb_size":                strconv.Itoa(f.imageMiniatureSize),
		"check_thumb_size_vertical": "false",
		"check_orig_resize":         "false",
		"orig_resize":               "1200",
		"check_resize_frontend":     "false",
		"check_optimization":        "false",
		"check_poster":              "false",
		"delete_after":              "0",
	}

	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			return nil, fmt.Errorf("failed to write form field %s: %v", key, err)
		}
	}

	part, err := writer.CreateFormFile("file1", fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %v", err)
	}

	if _, err := io.Copy(part, file); err != nil {
		return nil, fmt.Errorf("failed to copy file data: %v", err)
	}

	writer.Close()

	req, err := http.NewRequestWithContext(ctx, "POST", "https://new.fastpic.org/v2upload/", &buffer)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")

	if f.sid != "" {
		req.AddCookie(&http.Cookie{Name: "fp_sid", Value: f.sid})
		req.AddCookie(&http.Cookie{Name: "pp", Value: "1"})
	}

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("upload cancelled: %v", ctx.Err())
		}
		return nil, fmt.Errorf("upload request failed: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %v", err)
	}

	log.Printf("Fastpic response: %s", string(body))

	var respJSON struct {
		ThumbLink string `json:"thumb_link"`
		ViewLink  string `json:"view_link"`
		AlbumLink string `json:"album_link"`
		Codes     string `json:"codes"`
	}
	if err := json.Unmarshal(body, &respJSON); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %v", err)
	}

	result := &FastpicUploadResult{
		AlbumLink: "https://new.fastpic.org" + respJSON.AlbumLink,
		Direct:    extractDirectLink(respJSON.Codes),
	}

	// Extract BBCode values
	result.BBThumb, result.BBBig = extractBBCodes(respJSON.Codes)

	log.Printf("Upload completed. Direct: %s, BBThumb: %s, BBBig: %s",
		result.Direct, result.BBThumb, result.BBBig)
	return result, nil
}

// extractDirectLink extracts the direct image URL from the codes HTML
func extractDirectLink(codesHTML string) string {
	// Use regex to find the direct link input value
	re := regexp.MustCompile(`<input[^>]*value="(https://[^"]*\.jpg)"[^>]*>`)
	matches := re.FindStringSubmatch(codesHTML)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

// extractBBCodes extracts both BBCode formats from the codes HTML
func extractBBCodes(codesHTML string) (bbThumb, bbBig string) {
	doc, err := html.Parse(strings.NewReader(codesHTML))
	if err != nil {
		log.Printf("Failed to parse HTML codes: %v", err)
		return "", ""
	}

	var parseNode func(*html.Node)
	parseNode = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "input" {
			var value string
			for _, attr := range n.Attr {
				if attr.Key == "value" {
					value = attr.Val
					break
				}
			}

			// Check if it's a BBCode format
			if strings.HasPrefix(value, "[URL=") && strings.Contains(value, "[IMG]") {
				// Distinguish between thumbnail and big image BBCode
				if strings.Contains(value, "/thumb/") && strings.Contains(value, ".jpeg") {
					bbThumb = value
				} else if strings.Contains(value, "/big/") && strings.Contains(value, ".jpg") {
					bbBig = value
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			parseNode(c)
		}
	}

	parseNode(doc)
	return bbThumb, bbBig
}
