package service

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"bucketbird/backend/internal/storage"

	"github.com/google/uuid"
	"github.com/wader/goutubedl"
)

type YouTubeImportInput struct {
	URL               string
	DestinationPrefix string
	CookieHeader      string
}

type YouTubeImportProgress struct {
	Stage              string  `json:"stage"`
	Kind               string  `json:"kind,omitempty"`
	Index              int     `json:"index,omitempty"`
	Total              int     `json:"total,omitempty"`
	Imported           int     `json:"imported,omitempty"`
	Failed             int     `json:"failed,omitempty"`
	TotalBytes         int64   `json:"totalBytes,omitempty"`
	VideoTitle         string  `json:"videoTitle,omitempty"`
	VideoID            string  `json:"videoId,omitempty"`
	Message            string  `json:"message,omitempty"`
	Error              string  `json:"error,omitempty"`
	Destination        string  `json:"destination,omitempty"`
	BytesRead          int64   `json:"bytesRead,omitempty"`
	TotalBytesExpected int64   `json:"totalBytesExpected,omitempty"`
	Percent            float64 `json:"percent,omitempty"`
	SpeedBytesPerSec   float64 `json:"speedBytesPerSec,omitempty"`
	Skipped            bool    `json:"skipped,omitempty"`
	SkippedCount       int     `json:"skippedCount,omitempty"`
}

type YouTubeImportedItem struct {
	Title       string `json:"title"`
	Key         string `json:"key"`
	VideoID     string `json:"videoId"`
	SizeBytes   int64  `json:"sizeBytes"`
	ContentType string `json:"contentType"`
}

type YouTubeImportError struct {
	Title   string `json:"title,omitempty"`
	VideoID string `json:"videoId,omitempty"`
	Error   string `json:"error"`
}

type YouTubeImportResult struct {
	Kind       string                `json:"kind"`
	Imported   int                   `json:"imported"`
	Skipped    int                   `json:"skipped"`
	TotalBytes int64                 `json:"totalBytes"`
	Items      []YouTubeImportedItem `json:"items"`
	Errors     []YouTubeImportError  `json:"errors"`
}

var fileNameSanitizer = regexp.MustCompile(`[^a-zA-Z0-9\-\._ ]+`)

const (
	youtubeVideoIDMetadataKey    = "bucketbird-video-id"
	youtubeVideoTitleMetadataKey = "bucketbird-video-title"
)

// parseSize converts size strings like "10.50MiB" or "1.5GiB" to bytes
func parseSize(sizeStr string) (int64, error) {
	sizeStr = strings.TrimSpace(sizeStr)
	if sizeStr == "" {
		return 0, fmt.Errorf("empty size string")
	}

	// Remove leading ~ if present (approximate size)
	sizeStr = strings.TrimPrefix(sizeStr, "~")
	sizeStr = strings.TrimSpace(sizeStr)

	// Remove all internal whitespace (e.g., "  50.64MiB" becomes "50.64MiB")
	sizeStr = strings.ReplaceAll(sizeStr, " ", "")

	var multiplier int64 = 1
	var numStr string

	// Extract unit and number
	switch {
	case strings.HasSuffix(sizeStr, "GiB"):
		multiplier = 1024 * 1024 * 1024
		numStr = strings.TrimSuffix(sizeStr, "GiB")
	case strings.HasSuffix(sizeStr, "MiB"):
		multiplier = 1024 * 1024
		numStr = strings.TrimSuffix(sizeStr, "MiB")
	case strings.HasSuffix(sizeStr, "KiB"):
		multiplier = 1024
		numStr = strings.TrimSuffix(sizeStr, "KiB")
	case strings.HasSuffix(sizeStr, "GB"):
		multiplier = 1000 * 1000 * 1000
		numStr = strings.TrimSuffix(sizeStr, "GB")
	case strings.HasSuffix(sizeStr, "MB"):
		multiplier = 1000 * 1000
		numStr = strings.TrimSuffix(sizeStr, "MB")
	case strings.HasSuffix(sizeStr, "KB"):
		multiplier = 1000
		numStr = strings.TrimSuffix(sizeStr, "KB")
	case strings.HasSuffix(sizeStr, "B"):
		multiplier = 1
		numStr = strings.TrimSuffix(sizeStr, "B")
	default:
		// Assume bytes if no unit
		numStr = sizeStr
	}

	numStr = strings.TrimSpace(numStr)
	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse size number %q: %w", numStr, err)
	}

	return int64(num * float64(multiplier)), nil
}

// parseSpeed converts speed strings like "1.23MiB/s" to bytes per second
func parseSpeed(speedStr string) (float64, error) {
	speedStr = strings.TrimSpace(speedStr)
	if speedStr == "" {
		return 0, nil
	}

	// Remove all internal whitespace
	speedStr = strings.ReplaceAll(speedStr, " ", "")

	// Remove /s suffix
	speedStr = strings.TrimSuffix(speedStr, "/s")

	bytes, err := parseSize(speedStr)
	if err != nil {
		return 0, err
	}

	return float64(bytes), nil
}

// progressLineRegex matches yt-dlp progress output like:
// [download]   12.5% of ~10.50MiB at  1.23MiB/s ETA 00:08
// [download]  99.2% of ~  50.64MiB at    6.65MiB/s ETA 00:00 (frag 247/248)
var progressLineRegex = regexp.MustCompile(`\[download\]\s+(\d+\.?\d*)%\s+of\s+~?\s*([\d.]+\s*\w+)(?:.*?at\s+([\d.]+\s*\w+/s))?`)

// parseProgressLine parses a yt-dlp progress line and returns percent, total bytes, and speed
func parseProgressLine(line string) (percent float64, totalBytes int64, speed float64, ok bool) {
	matches := progressLineRegex.FindStringSubmatch(line)
	if len(matches) < 3 {
		return 0, 0, 0, false
	}

	// Parse percentage
	percent, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0, 0, 0, false
	}

	// Parse total size
	totalBytes, err = parseSize(matches[2])
	if err != nil {
		return 0, 0, 0, false
	}

	// Parse speed (optional)
	if len(matches) > 3 && matches[3] != "" {
		speed, _ = parseSpeed(matches[3])
	}

	return percent, totalBytes, speed, true
}

// buildYtDlpCommand constructs the yt-dlp command with appropriate flags
func buildYtDlpCommand(ctx context.Context, videoURL, outputPath, cookieFile string) *exec.Cmd {
	args := []string{
		"--newline",         // Print progress on separate lines
		"--no-playlist",     // Download only single video (not playlist)
		"-f", "bestvideo[ext=mp4]+bestaudio[ext=m4a]/best[ext=mp4]/best", // Format selection
		"-o", outputPath,    // Output path
		"--no-mtime",        // Don't set file modification time
		"--no-continue",     // Don't resume partial downloads
	}

	if cookieFile != "" {
		args = append(args, "--cookies", cookieFile)
	}

	args = append(args, videoURL)

	cmd := exec.CommandContext(ctx, "yt-dlp", args...)
	return cmd
}

// downloadYouTubeVideoViaSubprocess downloads a video using yt-dlp subprocess and streams progress
func (s *BucketService) downloadYouTubeVideoViaSubprocess(
	ctx context.Context,
	videoURL string,
	cookieFile string,
	outputPath string,
	progressCallback func(int64, int64, float64),
) error {
	cmd := buildYtDlpCommand(ctx, videoURL, outputPath, cookieFile)

	// Log the command being run
	cmdStr := "yt-dlp"
	if cookieFile != "" {
		cmdStr = fmt.Sprintf("yt-dlp --cookies %s --newline --no-playlist -f 'bestvideo[ext=mp4]+bestaudio[ext=m4a]/best[ext=mp4]/best' -o %s --no-mtime --no-continue %s",
			cookieFile, outputPath, videoURL)
	} else {
		cmdStr = fmt.Sprintf("yt-dlp --newline --no-playlist -f 'bestvideo[ext=mp4]+bestaudio[ext=m4a]/best[ext=mp4]/best' -o %s --no-mtime --no-continue %s",
			outputPath, videoURL)
	}
	s.logger.Info("running yt-dlp subprocess", "command", cmdStr)

	// Capture stdout and stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start yt-dlp: %w", err)
	}

	// Read stderr in background for error messages
	stderrDone := make(chan struct{})
	var stderrLines []string
	go func() {
		defer close(stderrDone)
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			stderrLines = append(stderrLines, line)
			s.logger.Debug("yt-dlp stderr", "line", line)
		}
	}()

	// Read stdout for progress
	scanner := bufio.NewScanner(stdout)
	var lastTotalBytes int64
	progressCount := 0
	for scanner.Scan() {
		line := scanner.Text()
		s.logger.Debug("yt-dlp stdout", "line", line)

		// Parse progress line
		percent, totalBytes, speed, ok := parseProgressLine(line)
		if ok {
			progressCount++
			s.logger.Debug("parsed progress",
				"percent", percent,
				"total_bytes", totalBytes,
				"speed", speed,
			)

			if progressCallback != nil {
				if totalBytes > 0 {
					lastTotalBytes = totalBytes
				} else if lastTotalBytes > 0 {
					totalBytes = lastTotalBytes
				}

				// Calculate bytes downloaded from percentage
				bytesRead := int64(float64(totalBytes) * (percent / 100.0))

				progressCallback(bytesRead, totalBytes, speed)
			}
		}
	}

	s.logger.Info("finished reading yt-dlp output", "progress_updates", progressCount)

	// Wait for stderr reader to finish
	<-stderrDone

	// Wait for command to complete
	if err := cmd.Wait(); err != nil {
		// Combine stderr for error message
		stderrText := strings.Join(stderrLines, "\n")
		if stderrText != "" {
			return fmt.Errorf("yt-dlp failed: %w: %s", err, stderrText)
		}
		return fmt.Errorf("yt-dlp failed: %w", err)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading yt-dlp output: %w", err)
	}

	return nil
}

func (s *BucketService) ImportYouTube(
	ctx context.Context,
	bucketID,
	userID uuid.UUID,
	input YouTubeImportInput,
	encryptionKey []byte,
	progress func(YouTubeImportProgress),
) (*YouTubeImportResult, error) {
	url := strings.TrimSpace(input.URL)
	if url == "" {
		return nil, fmt.Errorf("youtube url is required")
	}

	bucketName, err := s.getBucketName(ctx, bucketID, userID)
	if err != nil {
		return nil, err
	}

	store, err := s.GetObjectStore(ctx, bucketID, userID, encryptionKey)
	if err != nil {
		return nil, err
	}

	result := &YouTubeImportResult{
		Kind:   "video",
		Items:  make([]YouTubeImportedItem, 0),
		Errors: make([]YouTubeImportError, 0),
	}

	prefix := normalizeObjectPrefix(input.DestinationPrefix)

	emitProgress(progress, YouTubeImportProgress{
		Stage:       "resolving",
		Message:     "Resolving YouTube link",
		Destination: prefix,
	})

	// Determine which cookie to use: request cookie or stored cookie
	cookieHeader := input.CookieHeader
	cookieSource := "request"
	if cookieHeader == "" {
		s.logger.Info("no cookie in request, checking profile",
			"user_id", userID.String(),
		)
		// Try to get stored cookie from user profile
		profile, err := s.profiles.GetByUserID(ctx, userID)
		if err != nil {
			s.logger.Warn("failed to get profile for youtube cookie",
				"user_id", userID.String(),
				"error", err.Error(),
			)
		} else {
			s.logger.Info("profile retrieved",
				"user_id", userID.String(),
				"has_youtube_cookie", profile.YoutubeCookie != nil,
				"youtube_cookie_empty", profile.YoutubeCookie == nil || *profile.YoutubeCookie == "",
			)
			if profile.YoutubeCookie != nil && *profile.YoutubeCookie != "" {
				cookieHeader = *profile.YoutubeCookie
				cookieSource = "profile"
				s.logger.Info("using stored youtube cookie for user",
					"user_id", userID.String(),
					"cookie_length", len(cookieHeader),
				)
			}
		}
	} else {
		s.logger.Info("using cookie from request",
			"cookie_length", len(cookieHeader),
		)
	}

	// Create cookie file if cookies are available
	var cookieFile string
	if cookieHeader != "" {
		tmpFile, err := createCookieFile(cookieHeader)
		if err != nil {
			return nil, fmt.Errorf("failed to create cookie file: %w", err)
		}
		cookieFile = tmpFile
		defer os.Remove(cookieFile)

		// Log cookie file details for debugging
		s.logger.Info("created cookie file",
			"source", cookieSource,
			"file_path", cookieFile,
		)

		// Read and log the first 500 characters of the cookie file for debugging
		if content, err := os.ReadFile(cookieFile); err == nil {
			preview := string(content)
			if len(preview) > 500 {
				preview = preview[:500] + "..."
			}
			s.logger.Info("cookie file content preview", "content", preview)
		}
	}

	videos, kind, err := s.resolveYouTubeVideos(ctx, url, cookieFile, result, progress)
	if err != nil {
		return nil, err
	}
	result.Kind = kind
	totalVideos := len(videos)

	emitProgress(progress, YouTubeImportProgress{
		Stage:       "resolved",
		Kind:        kind,
		Total:       totalVideos,
		Message:     fmt.Sprintf("Found %d item(s)", totalVideos),
		Destination: prefix,
	})

	for i, video := range videos {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		emitProgress(progress, YouTubeImportProgress{
			Stage:      "starting",
			Kind:       kind,
			Index:      i + 1,
			Total:      totalVideos,
			VideoTitle: video.Title,
			VideoID:    video.ID,
			Message:    fmt.Sprintf("Downloading %q", video.Title),
		})

		progressFn := func(bytesRead int64, total int64, speed float64) {
			emitProgress(progress, YouTubeImportProgress{
				Stage:              "downloading",
				Kind:               kind,
				Index:              i + 1,
				Total:              totalVideos,
				VideoTitle:         video.Title,
				VideoID:            video.ID,
				BytesRead:          bytesRead,
				TotalBytesExpected: total,
				Percent:            computePercent(bytesRead, total),
				SpeedBytesPerSec:   speed,
			})
		}

		item, skipped, downloadErr := s.downloadYouTubeVideo(ctx, store, bucketName, prefix, cookieFile, video, progressFn)
		if downloadErr != nil {
			s.logger.Warn("failed to import youtube video",
				"title", video.Title,
				"video_id", video.ID,
				"error", downloadErr,
			)
			emitProgress(progress, YouTubeImportProgress{
				Stage:      "error",
				Kind:       kind,
				Index:      i + 1,
				Total:      totalVideos,
				VideoTitle: video.Title,
				VideoID:    video.ID,
				Error:      downloadErr.Error(),
			})
			result.Errors = append(result.Errors, YouTubeImportError{
				Title:   video.Title,
				VideoID: video.ID,
				Error:   downloadErr.Error(),
			})
			continue
		}

		if skipped {
			result.Skipped++
			emitProgress(progress, YouTubeImportProgress{
				Stage:      "skipped",
				Kind:       kind,
				Index:      i + 1,
				Total:      totalVideos,
				VideoTitle: video.Title,
				VideoID:    video.ID,
				Message:    fmt.Sprintf("%q already exists, skipping", video.Title),
				Skipped:    true,
			})
			continue
		}

		result.Items = append(result.Items, *item)
		result.Imported++
		result.TotalBytes += item.SizeBytes

		emitProgress(progress, YouTubeImportProgress{
			Stage:       "downloaded",
			Kind:        kind,
			Index:       i + 1,
			Total:       totalVideos,
			VideoTitle:  video.Title,
			VideoID:     video.ID,
			Message:     fmt.Sprintf("Downloaded %q", video.Title),
			Imported:    result.Imported,
			Failed:      len(result.Errors),
			TotalBytes:  result.TotalBytes,
			Destination: item.Key,
		})
	}

	if result.Imported > 0 {
		go func() {
			if err := s.recalculateBucketSize(context.Background(), bucketID, userID, encryptionKey); err != nil {
				s.logger.Error("failed to recalculate bucket size after youtube import",
					"bucket_id", bucketID.String(),
					"error", err,
				)
			}
		}()
	}

	emitProgress(progress, YouTubeImportProgress{
		Stage:        "finished",
		Kind:         kind,
		Imported:     result.Imported,
		Failed:       len(result.Errors),
		SkippedCount: result.Skipped,
		Total:        totalVideos,
		TotalBytes:   result.TotalBytes,
		Message:      "Import complete",
	})

	return result, nil
}

type videoInfo struct {
	ID           string
	Title        string
	DownloadURL  string
	Ext          string
	Filesize     int64
	Format       string
	FormatID     string
	Width        int64
	Height       int64
	ThumbnailURL string
}

func (s *BucketService) resolveYouTubeVideos(
	ctx context.Context,
	url string,
	cookieFile string,
	result *YouTubeImportResult,
	progress func(YouTubeImportProgress),
) ([]*videoInfo, string, error) {
	opts := goutubedl.Options{
		Type: goutubedl.TypeAny,
	}

	if cookieFile != "" {
		opts.Cookies = cookieFile
		s.logger.Info("resolving youtube url with cookies",
			"url", url,
			"cookie_file", cookieFile,
			"yt-dlp_command", fmt.Sprintf("yt-dlp --cookies %s %s", cookieFile, url),
		)
	} else {
		s.logger.Info("resolving youtube url without cookies",
			"url", url,
			"yt-dlp_command", fmt.Sprintf("yt-dlp %s", url),
		)
	}

	gResult, err := goutubedl.New(ctx, url, opts)
	if err != nil {
		return nil, "", fmt.Errorf("failed to resolve youtube url: %w", err)
	}

	info := gResult.Info

	// Check if it's a playlist or single video
	kind := "video"
	var videos []*videoInfo

	if info.Entries != nil && len(info.Entries) > 0 {
		// It's a playlist
		kind = "playlist"
		for _, entry := range info.Entries {
			if entry.ID == "" {
				continue
			}

			// Select best format with both video and audio
			format := selectBestFormat(entry.Formats)
			if format == nil {
				s.logger.Warn("no suitable format found", "video_id", entry.ID, "title", entry.Title)
				result.Errors = append(result.Errors, YouTubeImportError{
					Title:   entry.Title,
					VideoID: entry.ID,
					Error:   "no downloadable format found",
				})
				continue
			}

			videos = append(videos, &videoInfo{
				ID:           entry.ID,
				Title:        entry.Title,
				DownloadURL:  "", // Will be handled by goutubedl.Download()
				Ext:          format.Ext,
				Filesize:     int64(format.Filesize),
				Format:       format.Format,
				FormatID:     format.FormatID,
				Width:        int64(format.Width),
				Height:       int64(format.Height),
				ThumbnailURL: entry.Thumbnail,
			})
		}
	} else {
		// Single video
		format := selectBestFormat(info.Formats)
		if format == nil {
			return nil, "", errors.New("no downloadable format found")
		}

		videos = append(videos, &videoInfo{
			ID:           info.ID,
			Title:        info.Title,
			DownloadURL:  "", // Will be handled by goutubedl.Download()
			Ext:          format.Ext,
			Filesize:     int64(format.Filesize),
			Format:       format.Format,
			FormatID:     format.FormatID,
			Width:        int64(format.Width),
			Height:       int64(format.Height),
			ThumbnailURL: info.Thumbnail,
		})
	}

	return videos, kind, nil
}

func selectBestFormat(formats []goutubedl.Format) *goutubedl.Format {
	if len(formats) == 0 {
		return nil
	}

	// Prefer formats with both audio and video (acodec and vcodec present)
	var bestFormat *goutubedl.Format
	var bestScore int64

	for i := range formats {
		format := &formats[i]

		// Prefer mp4 container
		score := int64(0)
		if format.Ext == "mp4" {
			score += 1000000
		}

		// Prefer formats with both audio and video
		hasAudio := format.ACodec != "" && format.ACodec != "none"
		hasVideo := format.VCodec != "" && format.VCodec != "none"

		if hasAudio && hasVideo {
			score += 10000000
		}

		// Add resolution score
		score += int64(format.Width * format.Height)

		// Add filesize as tiebreaker (prefer larger = better quality)
		if format.Filesize > 0 {
			score += int64(format.Filesize) / 1000000 // MB
		}

		if bestFormat == nil || score > bestScore {
			bestFormat = format
			bestScore = score
		}
	}

	return bestFormat
}

func (s *BucketService) downloadYouTubeVideo(
	ctx context.Context,
	store *storage.ObjectStore,
	bucketName string,
	prefix string,
	cookieFile string,
	video *videoInfo,
	progress func(int64, int64, float64),
) (*YouTubeImportedItem, bool, error) {
	// Determine content type and extension
	contentType := "video/mp4"
	ext := ".mp4"
	if video.Ext != "" {
		ext = "." + video.Ext
		switch video.Ext {
		case "mp4":
			contentType = "video/mp4"
		case "webm":
			contentType = "video/webm"
		case "m4a":
			contentType = "audio/mp4"
		default:
			contentType = "application/octet-stream"
		}
	}

	primaryFilename := buildYouTubeFilename(video.Title, ext)
	primaryKey := primaryFilename
	if prefix != "" {
		primaryKey = prefix + primaryFilename
	}

	legacyFilename := buildYouTubeFilenameWithID(video.Title, video.ID, ext)
	legacyKey := legacyFilename
	if prefix != "" {
		legacyKey = prefix + legacyFilename
	}

	// Check if file already exists
	primaryHead, err := store.HeadObject(ctx, bucketName, primaryKey)
	if err != nil && !isNotFoundError(err) {
		return nil, false, err
	}
	if err == nil && metadataMatchesYouTubeVideo(primaryHead.Metadata, video.ID) {
		return &YouTubeImportedItem{
			Title:       video.Title,
			Key:         primaryKey,
			VideoID:     video.ID,
			SizeBytes:   0,
			ContentType: contentType,
		}, true, nil
	}
	if err != nil && isNotFoundError(err) {
		primaryHead = nil
	}

	if _, err := store.HeadObject(ctx, bucketName, legacyKey); err == nil {
		return &YouTubeImportedItem{
			Title:       video.Title,
			Key:         legacyKey,
			VideoID:     video.ID,
			SizeBytes:   0,
			ContentType: contentType,
		}, true, nil
	} else if !isNotFoundError(err) {
		return nil, false, err
	}

	key := primaryKey
	if primaryHead != nil {
		// A file already exists with the desired title, fall back to the legacy naming
		key = legacyKey
	}

	metadata := map[string]string{
		youtubeVideoIDMetadataKey: video.ID,
	}
	if video.Title != "" {
		metadata[youtubeVideoTitleMetadataKey] = video.Title
	}

	// Create a temporary file for downloading the video
	tmpFile, err := os.CreateTemp("", fmt.Sprintf("youtube-video-%s-*%s", video.ID, ext))
	if err != nil {
		return nil, false, fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	tmpFile.Close() // Close it so yt-dlp can write to it
	defer os.Remove(tmpPath)

	// Build video URL
	videoURL := fmt.Sprintf("https://www.youtube.com/watch?v=%s", video.ID)

	s.logger.Info("downloading youtube video via subprocess",
		"video_id", video.ID,
		"title", video.Title,
		"temp_file", tmpPath,
	)

	// Download video to temp file using yt-dlp subprocess with progress tracking
	if err := s.downloadYouTubeVideoViaSubprocess(ctx, videoURL, cookieFile, tmpPath, progress); err != nil {
		return nil, false, fmt.Errorf("failed to download video: %w", err)
	}

	// Get file size
	fileInfo, err := os.Stat(tmpPath)
	if err != nil {
		return nil, false, fmt.Errorf("failed to stat downloaded file: %w", err)
	}
	size := fileInfo.Size()

	// Upload to storage
	s.logger.Info("uploading video to storage",
		"video_id", video.ID,
		"key", key,
		"size_bytes", size,
	)

	// Open temp file for reading
	tmpFileRead, err := os.Open(tmpPath)
	if err != nil {
		return nil, false, fmt.Errorf("failed to open temp file for upload: %w", err)
	}
	defer tmpFileRead.Close()

	// Upload with progress tracking for S3 upload phase
	uploadProgressReader := newProgressReader(tmpFileRead, size, func(bytesRead, total int64, speed float64) {
		// Only report upload progress if we have a callback
		// This is the S3 upload phase, which is typically fast on local MinIO
		if progress != nil {
			progress(bytesRead, total, speed)
		}
	})
	defer uploadProgressReader.Close()

	if err := store.PutObject(ctx, bucketName, key, uploadProgressReader, contentType, metadata); err != nil {
		return nil, false, fmt.Errorf("failed to upload to storage: %w", err)
	}

	s.logger.Info("video download and upload complete",
		"video_id", video.ID,
		"key", key,
		"size_bytes", size,
	)

	return &YouTubeImportedItem{
		Title:       video.Title,
		Key:         key,
		VideoID:     video.ID,
		SizeBytes:   size,
		ContentType: contentType,
	}, false, nil
}

func buildYouTubeFilename(title, ext string) string {
	name := buildYouTubeBaseName(title)
	return fmt.Sprintf("%s%s", name, ext)
}

func buildYouTubeFilenameWithID(title, videoID, ext string) string {
	name := buildYouTubeBaseName(title)
	return fmt.Sprintf("%s-%s%s", name, videoID, ext)
}

func buildYouTubeBaseName(title string) string {
	name := sanitizeFileName(title)
	if name == "" {
		name = "youtube-video"
	}
	if len(name) > 80 {
		name = name[:80]
	}
	return name
}

func sanitizeFileName(value string) string {
	value = fileNameSanitizer.ReplaceAllString(value, "")
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "/", "-")
	value = strings.ReplaceAll(value, "\\", "-")
	value = strings.Join(strings.Fields(value), "-")
	return value
}

func metadataMatchesYouTubeVideo(metadata map[string]string, videoID string) bool {
	if len(metadata) == 0 || videoID == "" {
		return false
	}
	for key, value := range metadata {
		if strings.EqualFold(key, youtubeVideoIDMetadataKey) && value == videoID {
			return true
		}
	}
	return false
}

func normalizeObjectPrefix(prefix string) string {
	prefix = strings.TrimSpace(prefix)
	prefix = strings.TrimPrefix(prefix, "/")
	prefix = strings.ReplaceAll(prefix, "//", "/")
	prefix = strings.Trim(prefix, "/")
	if prefix == "" {
		return ""
	}
	return prefix + "/"
}

func emitProgress(progress func(YouTubeImportProgress), event YouTubeImportProgress) {
	if progress == nil {
		return
	}
	progress(event)
}

func createCookieFile(cookieHeader string) (string, error) {
	// Create a temporary file for Netscape cookie format
	tmpFile, err := os.CreateTemp("", "youtube-cookies-*.txt")
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	// Write cookie header directly to file
	_, err = tmpFile.WriteString(cookieHeader)
	if err != nil {
		return "", err
	}

	return tmpFile.Name(), nil
}

func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "notfound") || strings.Contains(msg, "status code: 404")
}

func computePercent(read, total int64) float64 {
	if total <= 0 {
		return 0
	}
	return (float64(read) / float64(total)) * 100
}

type progressReader struct {
	rc        io.ReadCloser
	total     int64
	read      int64
	lastBytes int64
	lastTime  time.Time
	callback  func(int64, int64, float64)
}

func newProgressReader(rc io.ReadCloser, total int64, cb func(int64, int64, float64)) *progressReader {
	return &progressReader{
		rc:       rc,
		total:    total,
		lastTime: time.Now(),
		callback: cb,
	}
}

func (p *progressReader) Read(b []byte) (int, error) {
	n, err := p.rc.Read(b)
	if n > 0 {
		p.read += int64(n)
		p.report(false)
	}
	if err == io.EOF {
		p.report(true)
	}
	return n, err
}

func (p *progressReader) report(force bool) {
	if p.callback == nil {
		return
	}
	now := time.Now()
	// Report more frequently (every 100ms instead of 500ms)
	if !force && now.Sub(p.lastTime) < 100*time.Millisecond {
		return
	}
	deltaBytes := p.read - p.lastBytes
	deltaTime := now.Sub(p.lastTime).Seconds()
	speed := 0.0
	if deltaTime > 0 {
		speed = float64(deltaBytes) / deltaTime
	}
	p.callback(p.read, p.total, speed)
	p.lastTime = now
	p.lastBytes = p.read
}

func (p *progressReader) Close() error {
	return p.rc.Close()
}

func (p *progressReader) BytesRead() int64 {
	return p.read
}
