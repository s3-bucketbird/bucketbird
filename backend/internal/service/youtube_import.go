package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
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

	// Download the video using goutubedl
	// We need to get the video URL again with a fresh goutubedl instance
	// because URLs expire quickly
	opts := goutubedl.Options{
		Type: goutubedl.TypeSingle,
	}

	if cookieFile != "" {
		opts.Cookies = cookieFile
	}

	// Build a URL for this specific video
	videoURL := fmt.Sprintf("https://www.youtube.com/watch?v=%s", video.ID)

	if cookieFile != "" {
		s.logger.Info("downloading youtube video with cookies",
			"video_id", video.ID,
			"title", video.Title,
			"cookie_file", cookieFile,
			"yt-dlp_command", fmt.Sprintf("yt-dlp --cookies %s %s", cookieFile, videoURL),
		)
	} else {
		s.logger.Info("downloading youtube video without cookies",
			"video_id", video.ID,
			"title", video.Title,
			"yt-dlp_command", fmt.Sprintf("yt-dlp %s", videoURL),
		)
	}

	gResult, err := goutubedl.New(ctx, videoURL, opts)
	if err != nil {
		return nil, false, fmt.Errorf("failed to initialize download: %w", err)
	}

	// Find the format we want
	targetFormat := selectBestFormat(gResult.Info.Formats)
	if targetFormat == nil {
		return nil, false, errors.New("no suitable format found for download")
	}

	downloadResult, err := gResult.Download(ctx, targetFormat.FormatID)
	if err != nil {
		return nil, false, fmt.Errorf("failed to download video: %w", err)
	}
	defer downloadResult.Close()

	// Wrap with progress reader
	progressReader := newProgressReader(downloadResult, video.Filesize, progress)
	defer progressReader.Close()

	if err := store.PutObject(ctx, bucketName, key, progressReader, contentType, metadata); err != nil {
		return nil, false, err
	}

	size := progressReader.BytesRead()
	if size == 0 && video.Filesize > 0 {
		size = video.Filesize
	}

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
	if !force && now.Sub(p.lastTime) < 500*time.Millisecond {
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
