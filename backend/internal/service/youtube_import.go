package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"bucketbird/backend/internal/storage"

	"github.com/google/uuid"
	"github.com/kkdai/youtube/v2"
)

type YouTubeImportInput struct {
	URL               string
	DestinationPrefix string
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

	client := s.youtubeClient
	if client == nil {
		client = &youtube.Client{}
		s.youtubeClient = client
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

	videos, kind, err := s.resolveYouTubeVideos(ctx, client, url, result, progress)
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

		item, skipped, downloadErr := s.downloadYouTubeVideo(ctx, store, bucketName, prefix, client, video, progressFn)
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

func (s *BucketService) resolveYouTubeVideos(
	ctx context.Context,
	client *youtube.Client,
	url string,
	result *YouTubeImportResult,
	progress func(YouTubeImportProgress),
) ([]*youtube.Video, string, error) {
	playlist, err := client.GetPlaylistContext(ctx, url)
	if err == nil {
		result.Kind = "playlist"
		return s.videosFromPlaylist(ctx, client, playlist, result, progress), "playlist", nil
	}

	if !errors.Is(err, youtube.ErrInvalidPlaylist) {
		return nil, "", fmt.Errorf("failed to load playlist: %w", err)
	}

	video, videoErr := client.GetVideoContext(ctx, url)
	if videoErr != nil {
		return nil, "", fmt.Errorf("failed to load video: %w", videoErr)
	}

	return []*youtube.Video{video}, "video", nil
}

func (s *BucketService) videosFromPlaylist(
	ctx context.Context,
	client *youtube.Client,
	playlist *youtube.Playlist,
	result *YouTubeImportResult,
	progress func(YouTubeImportProgress),
) []*youtube.Video {
	videos := make([]*youtube.Video, 0, len(playlist.Videos))
	for _, entry := range playlist.Videos {
		video, err := client.VideoFromPlaylistEntryContext(ctx, entry)
		if err != nil {
			s.logger.Warn("failed to load playlist entry", "video_id", entry.ID, "title", entry.Title, "error", err)
			result.Errors = append(result.Errors, YouTubeImportError{
				Title:   entry.Title,
				VideoID: entry.ID,
				Error:   err.Error(),
			})
			emitProgress(progress, YouTubeImportProgress{
				Stage:      "error",
				Kind:       "playlist",
				VideoTitle: entry.Title,
				VideoID:    entry.ID,
				Error:      err.Error(),
			})
			continue
		}
		videos = append(videos, video)
	}
	return videos
}

func (s *BucketService) downloadYouTubeVideo(
	ctx context.Context,
	store *storage.ObjectStore,
	bucketName string,
	prefix string,
	client *youtube.Client,
	video *youtube.Video,
	progress func(int64, int64, float64),
) (*YouTubeImportedItem, bool, error) {
	format, err := selectYouTubeFormat(video)
	if err != nil {
		return nil, false, err
	}

	stream, sizeHint, err := client.GetStreamContext(ctx, video, format)
	if err != nil {
		return nil, false, err
	}
	defer stream.Close()

	if format.ContentLength == 0 && sizeHint > 0 {
		format.ContentLength = sizeHint
	}

	contentType := contentTypeFromMime(format.MimeType)
	primaryFilename := buildYouTubeFilename(video.Title, format)
	primaryKey := primaryFilename
	if prefix != "" {
		primaryKey = prefix + primaryFilename
	}

	legacyFilename := buildYouTubeFilenameWithID(video.Title, video.ID, format)
	legacyKey := legacyFilename
	if prefix != "" {
		legacyKey = prefix + legacyFilename
	}

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
		// A file already exists with the desired title, fall back to the legacy naming that
		// includes the video ID to avoid overwriting unrelated content.
		key = legacyKey
	}

	metadata := map[string]string{
		youtubeVideoIDMetadataKey: video.ID,
	}
	if video.Title != "" {
		metadata[youtubeVideoTitleMetadataKey] = video.Title
	}

	progressReader := newProgressReader(stream, format.ContentLength, progress)
	defer progressReader.Close()

	if err := store.PutObject(ctx, bucketName, key, progressReader, contentType, metadata); err != nil {
		return nil, false, err
	}

	size := progressReader.BytesRead()
	if size == 0 && format.ContentLength > 0 {
		size = format.ContentLength
	}

	return &YouTubeImportedItem{
		Title:       video.Title,
		Key:         key,
		VideoID:     video.ID,
		SizeBytes:   size,
		ContentType: contentType,
	}, false, nil
}

func selectYouTubeFormat(video *youtube.Video) (*youtube.Format, error) {
	withAudio := video.Formats.WithAudioChannels()
	if len(withAudio) == 0 {
		return nil, fmt.Errorf("no downloadable formats with audio were found")
	}

	var mp4Formats youtube.FormatList
	for _, format := range withAudio {
		if strings.Contains(format.MimeType, "mp4") {
			mp4Formats = append(mp4Formats, format)
		}
	}

	candidate := withAudio
	if len(mp4Formats) > 0 {
		mp4Formats.Sort()
		candidate = mp4Formats
	} else {
		candidate.Sort()
	}

	selected := candidate[0]
	return &selected, nil
}

func buildYouTubeFilename(title string, format *youtube.Format) string {
	name := buildYouTubeBaseName(title)
	return fmt.Sprintf("%s%s", name, extensionFromMime(format.MimeType))
}

func buildYouTubeFilenameWithID(title, videoID string, format *youtube.Format) string {
	name := buildYouTubeBaseName(title)
	return fmt.Sprintf("%s-%s%s", name, videoID, extensionFromMime(format.MimeType))
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

func extensionFromMime(mimeType string) string {
	switch {
	case strings.Contains(mimeType, "audio/mp4"):
		return ".m4a"
	case strings.Contains(mimeType, "mp4"):
		return ".mp4"
	case strings.Contains(mimeType, "webm"):
		return ".webm"
	default:
		return ".bin"
	}
}

func contentTypeFromMime(mimeType string) string {
	if idx := strings.Index(mimeType, ";"); idx > -1 {
		mimeType = mimeType[:idx]
	}
	return strings.TrimSpace(mimeType)
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
