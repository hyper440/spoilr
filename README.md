# Spoilr

A desktop app for creating formatted movie/video spoiler posts with automatic screenshot generation and image hosting.

![demo](/media/spoilr_demo.webp)

## Features

- **Drag & Drop** - Add video files instantly
- **Auto Analysis** - Extract resolution, codecs, bitrates, duration
- **Screenshot Generation** - Configurable count and quality
- **Thumbnail Grids** - Generate movie thumbnails (requires MTN)
- **FastPic Integration** - Automatic image upload and BBCode generation
- **Custom Templates** - Customize output format with variable placeholders
- **Concurrent Processing** - Parallel screenshot generation and uploads

## Supported Platforms

- Windows 10/11 AMD64/ARM64
- macOS 10.15+ AMD64 (Can deploy to macOS 10.13+)
- macOS 11.0+ ARM64
- Ubuntu 24.04 AMD64/ARM64 (other Linux may work too!)

## Requirements

- FFmpeg (ffmpeg, ffprobe)
- [MTN](https://gitlab.com/movie_thumbnailer/mtn) (optional, for thumbnails)
- FastPic cookie (optional, for uploads to account)

## Usage

1. Drag video files into the app
2. Configure settings (screenshot count, quality, FastPic SID)
3. Click "Start Processing"
4. Copy generated BBCode spoiler text

## Build

Follow wails3 guilde [https://v3alpha.wails.io/getting-started/installation/](https://v3alpha.wails.io/getting-started/installation/)
