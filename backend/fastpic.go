package backend

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"regexp"
	"time"
)

type FastpicService struct {
	sid string
}

func NewFastpicService(sid string) *FastpicService {
	return &FastpicService{sid: sid}
}

// getFastpicUploadID gets upload ID from fastpic
func (f *FastpicService) getFastpicUploadID() (string, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	req, err := http.NewRequest("GET", "https://new.fastpic.org/", nil)
	if err != nil {
		return "", err
	}

	if f.sid != "" {
		req.AddCookie(&http.Cookie{Name: "fp_sid", Value: f.sid})
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Use regex to extract upload_id from JavaScript
	re := regexp.MustCompile(`"upload_id"\s*:\s*'([^']+)'`)
	matches := re.FindSubmatch(body)
	if len(matches) < 2 {
		return "", fmt.Errorf("could not find upload_id in response")
	}

	return string(matches[1]), nil
}

// uploadToFastpic uploads image to fastpic
func (f *FastpicService) uploadToFastpic(filePath, fileName, uploadID string) (*FastpicUploadResult, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var buffer bytes.Buffer
	writer := multipart.NewWriter(&buffer)

	// Add form fields
	fields := map[string]string{
		"uploading":                 "1",
		"fp":                        "not-loaded",
		"upload_id":                 uploadID,
		"check_thumb":               "size",
		"thumb_text":                "",
		"thumb_size":                "350",
		"check_thumb_size_vertical": "false",
		"check_orig_resize":         "false",
		"orig_resize":               "1200",
		"check_resize_frontend":     "false",
		"check_optimization":        "false",
		"check_poster":              "false",
		"delete_after":              "0",
	}

	for key, value := range fields {
		writer.WriteField(key, value)
	}

	// Add file
	part, err := writer.CreateFormFile("file1", fileName)
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(part, file)
	if err != nil {
		return nil, err
	}

	writer.Close()

	// Create request
	req, err := http.NewRequest("POST", "https://new.fastpic.org/v2upload/", &buffer)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	if f.sid != "" {
		req.AddCookie(&http.Cookie{Name: "fp_sid", Value: f.sid})
		req.AddCookie(&http.Cookie{Name: "pp", Value: "1"})
	}

	// Send request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Parse response according to the new format
	var response struct {
		ThumbLink string `json:"thumb_link"`
		ViewLink  string `json:"view_link"`
		AlbumLink string `json:"album_link"`
		Codes     string `json:"codes"`
	}

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil {
		return nil, err
	}

	result := &FastpicUploadResult{
		AlbumLink: "https://new.fastpic.org" + response.AlbumLink,
		Direct:    response.ViewLink,
	}

	// Extract BBCode thumbnail from the codes HTML
	// Look for the BBCode input value
	bbCodeRegex := regexp.MustCompile(`\[URL=[^\]]+\]\[IMG\]([^\[]+)\[/IMG\]\[/URL\]`)
	matches := bbCodeRegex.FindStringSubmatch(response.Codes)
	if len(matches) > 1 {
		result.BBThumb = matches[1]
	}

	// Extract HTML thumbnail
	htmlRegex := regexp.MustCompile(`<img src="([^"]+)"`)
	htmlMatches := htmlRegex.FindStringSubmatch(response.Codes)
	if len(htmlMatches) > 1 {
		result.HTMLThumb = fmt.Sprintf(`<a href="%s" target="_blank"><img src="%s" border="0"></a>`, response.ViewLink, htmlMatches[1])
	}

	// Extract Markdown thumbnail
	mdRegex := regexp.MustCompile(`\[!\[FastPic\.Ru\]\(([^\)]+)\)\]\(([^\)]+)\)`)
	mdMatches := mdRegex.FindStringSubmatch(response.Codes)
	if len(mdMatches) > 2 {
		result.MDThumb = fmt.Sprintf("[![FastPic.Ru](%s)](%s)", mdMatches[1], mdMatches[2])
	}

	// For BBBig, use the same thumbnail URL (fastpic doesn't seem to provide separate big image in new format)
	result.BBBig = result.BBThumb

	return result, nil
}
