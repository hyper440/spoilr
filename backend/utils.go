package backend

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func GetVideoMediaInfo(filePath string) (MediaInfo, bool, error) {
	cmd := exec.Command("ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		filePath,
	)

	output, err := cmd.Output()
	if err != nil {
		return MediaInfo{}, false, nil // Not a video file or ffprobe failed
	}

	var result struct {
		Format struct {
			Duration string            `json:"duration"`
			Size     string            `json:"size"`
			BitRate  string            `json:"bit_rate"`
			Tags     map[string]string `json:"tags"`
		} `json:"format"`
		Streams []struct {
			CodecType     string            `json:"codec_type"`
			CodecName     string            `json:"codec_name"`
			Width         int               `json:"width"`
			Height        int               `json:"height"`
			Duration      string            `json:"duration"`
			BitRate       string            `json:"bit_rate"`
			RFrameRate    string            `json:"r_frame_rate"`
			AvgFrameRate  string            `json:"avg_frame_rate"`
			SampleRate    string            `json:"sample_rate"`
			Channels      int               `json:"channels"`
			ChannelLayout string            `json:"channel_layout"`
			Tags          map[string]string `json:"tags"`
		} `json:"streams"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return MediaInfo{}, false, fmt.Errorf("failed to parse ffprobe output: %v", err)
	}

	// Check if it has video streams
	hasVideo := false
	for _, stream := range result.Streams {
		if stream.CodecType == "video" {
			hasVideo = true
			break
		}
	}

	if !hasVideo {
		return MediaInfo{}, false, nil
	}

	// Build MediaInfo
	mediaInfo := MediaInfo{
		General: make(map[string]string),
		Video:   make(map[string]string),
		Audio:   make(map[string]string),
	}

	// General info
	mediaInfo.General["duration"] = result.Format.Duration
	mediaInfo.General["size"] = result.Format.Size
	mediaInfo.General["bit_rate"] = result.Format.BitRate

	// Process streams
	for _, stream := range result.Streams {
		switch stream.CodecType {
		case "video":
			mediaInfo.Video["codec_name"] = stream.CodecName
			if stream.Width > 0 {
				mediaInfo.Video["width"] = strconv.Itoa(stream.Width)
			}
			if stream.Height > 0 {
				mediaInfo.Video["height"] = strconv.Itoa(stream.Height)
			}
			if stream.Duration != "" {
				mediaInfo.Video["duration"] = stream.Duration
			}
			if stream.BitRate != "" {
				mediaInfo.Video["bit_rate"] = stream.BitRate
			}
			if stream.BitRate == "" && stream.Tags != nil {
				if br, ok := stream.Tags["BPS"]; ok {
					mediaInfo.Video["bit_rate"] = br
				}
			}

			// Extract framerate info
			if stream.RFrameRate != "" {
				mediaInfo.Video["r_frame_rate"] = stream.RFrameRate
				// Convert to decimal
				if fps := parseFrameRate(stream.RFrameRate); fps > 0 {
					mediaInfo.Video["fps_decimal"] = fmt.Sprintf("%.3f", fps)
				}
			}
			if stream.AvgFrameRate != "" {
				mediaInfo.Video["avg_frame_rate"] = stream.AvgFrameRate
			}

		case "audio":
			mediaInfo.Audio["codec_name"] = stream.CodecName
			if stream.Duration != "" {
				mediaInfo.Audio["duration"] = stream.Duration
			}
			if stream.BitRate != "" {
				mediaInfo.Audio["bit_rate"] = stream.BitRate
			}
			if stream.BitRate == "" && stream.Tags != nil {
				if br, ok := stream.Tags["BPS"]; ok {
					mediaInfo.Audio["bit_rate"] = br
				}
			}

			// Extract audio-specific info
			if stream.SampleRate != "" {
				mediaInfo.Audio["sample_rate"] = stream.SampleRate
			}
			if stream.Channels > 0 {
				mediaInfo.Audio["channels"] = strconv.Itoa(stream.Channels)
			}
			if stream.ChannelLayout != "" {
				mediaInfo.Audio["channel_layout"] = stream.ChannelLayout
			}
		}
	}

	return mediaInfo, true, nil
}

func parseFrameRate(frameRate string) float64 {
	if frameRate == "" || frameRate == "0/0" {
		return 0
	}

	parts := strings.Split(frameRate, "/")
	if len(parts) != 2 {
		return 0
	}

	numerator, err1 := strconv.ParseFloat(parts[0], 64)
	denominator, err2 := strconv.ParseFloat(parts[1], 64)

	if err1 != nil || err2 != nil || denominator == 0 {
		return 0
	}

	return numerator / denominator
}

func ExtractMediaInfo(movie *Movie, mediaInfo MediaInfo) {
	if duration, ok := mediaInfo.General["duration"]; ok {
		if dur, err := strconv.ParseFloat(duration, 64); err == nil {
			movie.DurationFormatted = FormatDuration(time.Duration(dur * float64(time.Second)))
			movie.Duration = dur
		}
	}

	if width, ok := mediaInfo.Video["width"]; ok {
		movie.Width = width
	}

	if height, ok := mediaInfo.Video["height"]; ok {
		movie.Height = height
	}

	// Extract bitrates
	if bitRate, ok := mediaInfo.Video["bit_rate"]; ok && bitRate != "" {
		movie.VideoBitRate = FormatBitRate(bitRate)
	} else if overallBitRateStr, ok := mediaInfo.General["bit_rate"]; ok && overallBitRateStr != "" {
		if overall, err := strconv.ParseFloat(overallBitRateStr, 64); err == nil {
			estimatedVideoBitRate := overall * 0.8
			movie.VideoBitRate = FormatBitRate(fmt.Sprintf("%.0f", estimatedVideoBitRate))
		}
	}

	if bitRate, ok := mediaInfo.Audio["bit_rate"]; ok && bitRate != "" {
		movie.AudioBitRate = FormatBitRate(bitRate)
	} else if overallBitRateStr, ok := mediaInfo.General["bit_rate"]; ok && overallBitRateStr != "" {
		if overall, err := strconv.ParseFloat(overallBitRateStr, 64); err == nil {
			estimatedAudioBitRate := overall * 0.1
			movie.AudioBitRate = FormatBitRate(fmt.Sprintf("%.0f", estimatedAudioBitRate))
		}
	}

	if codec, ok := mediaInfo.Video["codec_name"]; ok {
		movie.VideoCodec = codec
	}

	if codec, ok := mediaInfo.Audio["codec_name"]; ok {
		movie.AudioCodec = codec
	}

	if overallBitRate, ok := mediaInfo.General["bit_rate"]; ok {
		movie.BitRate = FormatBitRate(overallBitRate)
	}

	// Store formatted video info
	if rFrameRate, ok := mediaInfo.Video["r_frame_rate"]; ok {
		movie.Params["%VIDEO_FPS_FRACTIONAL%"] = rFrameRate
	}
	if fpsDecimal, ok := mediaInfo.Video["fps_decimal"]; ok {
		movie.Params["%VIDEO_FPS%"] = fpsDecimal
	}

	// Store formatted audio info
	if sampleRate, ok := mediaInfo.Audio["sample_rate"]; ok {
		movie.Params["%AUDIO_SAMPLE_RATE%"] = formatSampleRate(sampleRate)
	}
	if channels, ok := mediaInfo.Audio["channels"]; ok {
		movie.Params["%AUDIO_CHANNELS%"] = formatChannels(channels)
	}

	// Store all raw parameters
	for key, value := range mediaInfo.General {
		movie.Params[fmt.Sprintf("%%General@%s%%", key)] = value
	}

	for key, value := range mediaInfo.Video {
		movie.Params[fmt.Sprintf("%%Video@%s%%", key)] = value
	}

	for key, value := range mediaInfo.Audio {
		movie.Params[fmt.Sprintf("%%Audio@%s%%", key)] = value
	}
}

func formatSampleRate(sampleRateStr string) string {
	if sampleRateStr == "" {
		return ""
	}

	sampleRate, err := strconv.ParseFloat(sampleRateStr, 64)
	if err != nil {
		return sampleRateStr
	}

	if sampleRate >= 1000 {
		return fmt.Sprintf("%.1f kHz", sampleRate/1000)
	}
	return fmt.Sprintf("%.0f Hz", sampleRate)
}

func formatChannels(channelsStr string) string {
	if channelsStr == "" {
		return ""
	}

	channels, err := strconv.Atoi(channelsStr)
	if err != nil {
		return channelsStr
	}

	switch channels {
	case 1:
		return "1 channel (mono)"
	case 2:
		return "2 channels (stereo)"
	case 6:
		return "6 channels (5.1)"
	case 8:
		return "8 channels (7.1)"
	default:
		return fmt.Sprintf("%d channels", channels)
	}
}

func FormatDuration(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60

	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}

func FormatBitRate(bitRateStr string) string {
	if bitRateStr == "" {
		return ""
	}

	bitRate, err := strconv.ParseFloat(bitRateStr, 64)
	if err != nil {
		return bitRateStr
	}

	kbps := bitRate / 1000
	if kbps >= 1000 {
		return fmt.Sprintf("%.1f Mbps", kbps/1000)
	}
	return fmt.Sprintf("%.0f kbps", kbps)
}

func FormatFileSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
